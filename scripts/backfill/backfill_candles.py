#!/usr/bin/env python3
"""Fetch historical OHLC candles for the complete TSC instrument catalog.

The script runs on a workstation (not the rate-limited production egress) and
emits loader-ready CSV on stdout:

    code,resolution,bucket_iso,open,high,low,close,count

Supported resolutions:
  * 1d: maximum reliable history (default start 2005)
  * 1h: provider-limited recent 729 days
  * 5m: provider-limited recent 59 days

Sources:
  * equities: Yahoo adjusted OHLC
  * fiat: Frankfurter/ECB daily, Yahoo direct/cross fallback and intraday
  * crypto: Binance OHLC, Yahoo fallback
  * gold: GC=F × USDTRY, transformed into every catalog gold product

Today's/current forming bucket is skipped so a backfill never overwrites the
live aggregator. A JSON quality manifest can be written with --manifest.
"""

import argparse
import bisect
import datetime as dt
import json
import math
import sys
from typing import Dict, Iterable, Optional, Tuple

import catalog as C
import fetch as F

Bar = Tuple[float, float, float, float]
Bars = Dict[dt.datetime, Bar]

USD_PEGS = {
    "SAR": 3.75,
    "AED": 3.6725,
    "QAR": 3.64,
    "BHD": 0.376,
    "OMR": 0.38449,
    "JOD": 0.709,
}
CROSS_PEGS = {
    "FKP": ("GBP", 1.0),
    "GIP": ("GBP", 1.0),
    "SHP": ("GBP", 1.0),
    "BTN": ("INR", 1.0),
}
EUR_PEGS = {"BGN": 1.95583}

YAHOO_DIRECT_INTRADAY = {"EUR", "GBP", "JPY", "CHF", "CAD", "AUD", "NZD", "AED"}

RESOLUTION_SECONDS = {"5m": 300, "1h": 3600, "1d": 86400}
MIN_SERIES_COUNTS = {"5m": 100, "1h": 100, "1d": 200}
MAX_GAP_SECONDS = {
    "5m": 5 * 86400,   # weekends + exchange holidays
    "1h": 5 * 86400,
    "1d": 14 * 86400,  # long exchange/FX holiday windows
}


def log(*args):
    print(*args, file=sys.stderr, flush=True)


def utc_midnight(value: dt.date) -> dt.datetime:
    return dt.datetime(value.year, value.month, value.day, tzinfo=dt.timezone.utc)


def normalize_daily(raw: Dict[str, tuple]) -> Bars:
    return {utc_midnight(dt.date.fromisoformat(day)): tuple(map(float, bar)) for day, bar in raw.items()}


def close_bars(series: Dict[str, float]) -> Bars:
    return {
        utc_midnight(dt.date.fromisoformat(day)): (float(value),) * 4
        for day, value in series.items()
        if value and value > 0
    }


def ffill(bars: Bars):
    keys = sorted(bars)

    def at(bucket: dt.datetime) -> Optional[Bar]:
        index = bisect.bisect_right(keys, bucket) - 1
        return bars[keys[index]] if index >= 0 else None

    return at


def multiply_bars(left: Bars, right: Bars, scale: float = 1.0) -> Bars:
    """Multiply positive OHLC series; highs/lows use conservative extremes."""
    right_at = ffill(right)
    out: Bars = {}
    for bucket, (lo, lh, ll, lc) in left.items():
        rb = right_at(bucket)
        if rb is None:
            continue
        ro, rh, rl, rc = rb
        out[bucket] = (
            lo * ro * scale,
            lh * rh * scale,
            ll * rl * scale,
            lc * rc * scale,
        )
    return out


def ratio_bars(numerator: Bars, denominator: Bars) -> Bars:
    """OHLC ratio; high uses numerator-high/denominator-low and vice versa."""
    denominator_at = ffill(denominator)
    out: Bars = {}
    for bucket, (no, nh, nl, nc) in numerator.items():
        den = denominator_at(bucket)
        if den is None or min(den) <= 0:
            continue
        do, dh, dl, dc = den
        out[bucket] = (no / do, nh / dl, nl / dh, nc / dc)
    return out


def scaled_bars(bars: Bars, divisor: float) -> Bars:
    return {
        bucket: tuple(value / divisor for value in bar)
        for bucket, bar in bars.items()
        if divisor > 0
    }


def needs_daily_fallback(bars: Bars, end: dt.datetime) -> bool:
    keys = sorted(bars)
    if len(keys) < MIN_SERIES_COUNTS["1d"]:
        return True
    if keys[-1] < end - dt.timedelta(days=10):
        return True
    return any(
        (current - previous).total_seconds() > MAX_GAP_SECONDS["1d"]
        for previous, current in zip(keys, keys[1:])
    )


def resolution_range(
    resolution: str,
    daily_start: dt.date,
    daily_end: dt.date,
    five_min_days: int,
    hourly_days: int,
) -> Tuple[dt.datetime, dt.datetime]:
    now = dt.datetime.now(dt.timezone.utc)
    if resolution == "5m":
        return now - dt.timedelta(days=five_min_days), now
    if resolution == "1h":
        return now - dt.timedelta(days=hourly_days), now
    return utc_midnight(daily_start), utc_midnight(daily_end + dt.timedelta(days=1))


def current_bucket(resolution: str) -> dt.datetime:
    now = dt.datetime.now(dt.timezone.utc)
    seconds = RESOLUTION_SECONDS[resolution]
    epoch = int(now.timestamp())
    return dt.datetime.fromtimestamp(epoch - epoch % seconds, tz=dt.timezone.utc)


def valid_bar(bar: Bar) -> bool:
    o, h, low, c = bar
    return (
        all(value is not None and math.isfinite(value) and value > 0 for value in bar)
        and h >= max(o, c)
        and low <= min(o, c)
        and h >= low
    )


def normalize_bar(bar: Bar) -> Tuple[Optional[Bar], bool]:
    """Expand a provider's H/L envelope around O/C when settlement is outside.

    Yahoo occasionally reports futures settlement or corporate-action-adjusted
    close outside the quoted intraday range. Preserving O/C and expanding H/L
    is deterministic and auditable; non-positive/non-finite values remain hard
    failures.
    """
    if not all(value is not None and math.isfinite(value) and value > 0 for value in bar):
        return None, False
    o, high, low, close = bar
    normalized = (o, max(high, o, close), min(low, o, close), close)
    return normalized, normalized != bar


def emit(
    code: str,
    resolution: str,
    bars: Bars,
    writer,
    source: str,
    manifest: dict,
    min_count: int,
    listing_starts: Dict[str, dt.date],
) -> int:
    cutoff = current_bucket(resolution)
    emitted = 0
    invalid = 0
    normalized_count = 0
    first = None
    last = None
    emitted_buckets = []
    for bucket in sorted(bars):
        if bucket >= cutoff:
            continue
        bar, normalized = normalize_bar(bars[bucket])
        if bar is None or not valid_bar(bar):
            invalid += 1
            continue
        if normalized:
            normalized_count += 1
        o, h, low, c = bar
        stamp = bucket.astimezone(dt.timezone.utc).strftime("%Y-%m-%d %H:%M:%S+00")
        writer.write(
            f"{code},{resolution},{stamp},{o:.12g},{h:.12g},{low:.12g},{c:.12g},0\n"
        )
        emitted += 1
        emitted_buckets.append(bucket)
        first = first or bucket
        last = bucket

    key = f"{code}|{resolution}"
    max_gap_seconds = 0
    max_gap_from = None
    max_gap_to = None
    for previous, current in zip(emitted_buckets, emitted_buckets[1:]):
        gap = int((current - previous).total_seconds())
        if gap > max_gap_seconds:
            max_gap_seconds = gap
            max_gap_from = previous
            max_gap_to = current
    manifest["series"][key] = {
        "code": code,
        "resolution": resolution,
        "source": source,
        "count": emitted,
        "invalid": invalid,
        "normalized": normalized_count,
        "from": first.isoformat() if first else None,
        "to": last.isoformat() if last else None,
        "max_gap_seconds": max_gap_seconds,
    }
    if emitted == 0:
        manifest["empty"].append(key)
    if invalid:
        manifest["invalid"].append(key)
    if normalized_count:
        manifest["normalized"].append({"series": key, "count": normalized_count})
    if emitted and emitted < min_count:
        listing_start = listing_starts.get(code)
        listing_matches = bool(
            listing_start
            and first
            and abs((first.date() - listing_start).days) <= 31
        )
        item = {
            "series": key,
            "count": emitted,
            "minimum": min_count,
            "first": first.isoformat() if first else None,
        }
        if listing_matches:
            item["listing_start"] = listing_start.isoformat()
            manifest["listing_exceptions"].append(item)
        else:
            manifest["short"].append(item)
    if max_gap_seconds > MAX_GAP_SECONDS[resolution]:
        manifest["gaps"].append({
            "series": key,
            "seconds": max_gap_seconds,
            "from": max_gap_from.isoformat(),
            "to": max_gap_to.isoformat(),
        })
    log(f"  {code:16s} {resolution:3s} {emitted:6d} bars ({source})")
    return emitted


def yahoo_bars(symbol: str, start: dt.datetime, end: dt.datetime, resolution: str, adjusted=False) -> Bars:
    if resolution == "1d":
        return normalize_daily(F.yahoo_ohlc(symbol, start.date(), (end - dt.timedelta(days=1)).date(), adjusted=adjusted))
    return F.yahoo_ohlc_interval(symbol, start, end, resolution, adjusted=adjusted)


def usdtry_bars(start: dt.datetime, end: dt.datetime, resolution: str) -> Bars:
    if resolution == "1d":
        daily = F.frankfurter_try("USD", start.date(), (end - dt.timedelta(days=1)).date())
        yahoo = yahoo_bars("USDTRY=X", start, end, resolution)
        # Keep real Yahoo OHLC where available; fill older ECB days as flat bars.
        merged = close_bars(daily)
        merged.update(yahoo)
        return merged
    return yahoo_bars("USDTRY=X", start, end, resolution)


def currency_bars(
    iso: str,
    usdtry: Bars,
    start: dt.datetime,
    end: dt.datetime,
    resolution: str,
) -> Tuple[Bars, str]:
    if iso == "USD":
        return dict(usdtry), "Frankfurter/Yahoo" if resolution == "1d" else "Yahoo"

    if resolution == "1d":
        if iso in USD_PEGS:
            return scaled_bars(usdtry, USD_PEGS[iso]), "USD peg"
        if iso in CROSS_PEGS:
            peg_iso, mult = CROSS_PEGS[iso]
            peg_bars, src = currency_bars(peg_iso, usdtry, start, end, resolution)
            if mult == 1.0:
                return peg_bars, f"cross-peg {peg_iso} ({src})"
            return (
                {
                    ts: (o * mult, h * mult, lo * mult, c * mult)
                    for ts, (o, h, lo, c) in peg_bars.items()
                },
                f"cross-peg {peg_iso}×{mult} ({src})",
            )
        if iso in EUR_PEGS:
            eur = close_bars(
                F.frankfurter_try(
                    "EUR",
                    start.date(),
                    (end - dt.timedelta(days=1)).date(),
                )
            )
            eur.update(yahoo_bars("EURTRY=X", start, end, resolution))
            return scaled_bars(eur, EUR_PEGS[iso]), "EUR irrevocable peg"

        ecb = close_bars(
            F.frankfurter_try(
                iso,
                start.date(),
                (end - dt.timedelta(days=1)).date(),
            )
        )
        if not needs_daily_fallback(ecb, end):
            return ecb, "Frankfurter (close-only)"

        yahoo_direct = yahoo_bars(f"{iso}TRY=X", start, end, resolution)
        merged = dict(yahoo_direct)
        merged.update(ecb)
        if not needs_daily_fallback(merged, end):
            return merged, "Frankfurter + Yahoo direct"

        # Yahoo "<ISO>=X" is units of ISO per USD. XXXTRY = USDTRY / USDXXX.
        cross = yahoo_bars(f"{iso}=X", start, end, resolution)
        merged = ratio_bars(usdtry, cross)
        merged.update(yahoo_direct)
        merged.update(ecb)
        if not needs_daily_fallback(merged, end):
            return merged, "Frankfurter + Yahoo cross"

        nbp = close_bars(
            F.nbp_try(
                iso,
                start.date(),
                (end - dt.timedelta(days=1)).date(),
            )
        )
        nbp.update(merged)
        return nbp, "Frankfurter/Yahoo + NBP PLN cross"

    # Intraday: only request Yahoo pairs known to carry a deep series. Probing
    # 55 mostly-unsupported pairs can trigger a long 429 storm. For the rest,
    # apply a daily ECB/Yahoo cross to the single liquid USDTRY intraday leg.
    direct_yahoo: Bars = {}
    if resolution == "1d" or iso in YAHOO_DIRECT_INTRADAY:
        direct_yahoo = yahoo_bars(f"{iso}TRY=X", start, end, resolution)
        threshold = 20 if resolution == "1d" else 3
        if len(direct_yahoo) >= threshold:
            return direct_yahoo, "Yahoo direct"

    if resolution != "1d":
        direct_daily = F.frankfurter_try(iso, start.date(), end.date())
        usd_daily = F.frankfurter_try("USD", start.date(), end.date())
        if direct_daily and usd_daily:
            # (TRY/ISO) / (TRY/USD) = USD/ISO; multiply by intraday TRY/USD.
            factor = ratio_bars(close_bars(direct_daily), close_bars(usd_daily))
            return multiply_bars(usdtry, factor), "USDTRY×ECB daily cross"
        if iso in USD_PEGS:
            return scaled_bars(usdtry, USD_PEGS[iso]), "USD peg"

    # Yahoo "<ISO>=X" is units of ISO per USD. XXXTRY = USDTRY / USDXXX.
    cross_resolution = "1d" if resolution != "1d" else resolution
    cross = yahoo_bars(f"{iso}=X", start, end, cross_resolution)
    if cross:
        derived = ratio_bars(usdtry, cross)
        if direct_yahoo:
            derived.update(direct_yahoo)
        return derived, "Yahoo cross"
    return direct_yahoo, "Yahoo direct"


def crypto_bars(
    code: str,
    yahoo_symbol: str,
    binance_symbol: str,
    usdtry: Bars,
    start: dt.datetime,
    end: dt.datetime,
    resolution: str,
) -> Tuple[Bars, str]:
    def usd_leg(yahoo: str, binance: str) -> Tuple[Bars, str]:
        bars = F.binance_ohlc_interval(binance, start, end, resolution)
        if len(bars) >= 20:
            return bars, "Binance"
        return yahoo_bars(yahoo, start, end, resolution), "Yahoo"

    usd, source = usd_leg(yahoo_symbol, binance_symbol)
    if code == "MATIC":
        legacy, legacy_source = usd_leg("MATIC-USD", "MATICUSDT")
        legacy.update(usd)  # Polygon's September 2024 MATIC→POL migration was 1:1.
        usd = legacy
        source = f"{legacy_source} MATIC + {source} POL"
    return multiply_bars(usd, usdtry), source + "×USDTRY"


def requested_codes(codes: Optional[str]) -> Optional[set]:
    return {code.strip() for code in codes.split(",") if code.strip()} if codes else None


def want(code: str, selected: Optional[set]) -> bool:
    return selected is None or code in selected


def expected_codes(kinds: set, selected: Optional[set]) -> Iterable[str]:
    values = []
    if "currency" in kinds:
        values.extend(f"{iso}TRY" for iso in C.CURRENCIES)
    if "gold" in kinds:
        values.extend(code for code, _, _ in C.GOLD_PRODUCTS)
    if "crypto" in kinds:
        values.extend(code for code, _, _ in C.CRYPTOS)
    if "stock" in kinds:
        values.extend(C.STOCKS)
    if "global" in kinds:
        values.extend(code for code, _ in C.GLOBAL_STOCKS)
    return [code for code in values if want(code, selected)]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--start-year", type=int, default=2005)
    parser.add_argument("--kinds", default="currency,gold,crypto,stock,global")
    parser.add_argument("--resolutions", default="1d")
    parser.add_argument("--codes", help="comma-separated instrument codes")
    parser.add_argument("--five-minute-days", type=int, default=59)
    parser.add_argument("--hourly-days", type=int, default=729)
    parser.add_argument("--manifest", help="write JSON quality/coverage manifest")
    parser.add_argument("--strict", action="store_true", help="exit non-zero when any quality gate fails")
    parser.add_argument("--min-count", type=int, help="override the resolution-specific minimum series length")
    parser.add_argument(
        "--listing-exceptions",
        help="JSON object mapping known IPO/listing codes to YYYY-MM-DD",
    )
    args = parser.parse_args()

    kinds = {value.strip() for value in args.kinds.split(",") if value.strip()}
    resolutions = [value.strip() for value in args.resolutions.split(",") if value.strip()]
    unknown = set(resolutions) - set(RESOLUTION_SECONDS)
    if unknown:
        parser.error(f"unsupported resolutions: {', '.join(sorted(unknown))}")
    selected = requested_codes(args.codes)
    daily_start = dt.date(args.start_year, 1, 1)
    daily_end = dt.date.today() - dt.timedelta(days=1)
    expected = list(expected_codes(kinds, selected))
    listing_starts: Dict[str, dt.date] = {}
    if args.listing_exceptions:
        with open(args.listing_exceptions, encoding="utf-8") as handle:
            listing_starts = {
                str(code).upper(): dt.date.fromisoformat(value)
                for code, value in json.load(handle).items()
            }

    manifest = {
        "generated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
        "requested_codes": expected,
        "resolutions": resolutions,
        "series": {},
        "empty": [],
        "invalid": [],
        "short": [],
        "gaps": [],
        "listing_exceptions": [],
        "normalized": [],
        "total_rows": 0,
    }

    writer = sys.stdout
    total = 0
    for resolution in resolutions:
        start, end = resolution_range(
            resolution,
            daily_start,
            daily_end,
            args.five_minute_days,
            args.hourly_days,
        )
        log(f"\n=== {resolution} {start.isoformat()} .. {end.isoformat()} ===")
        min_count = args.min_count if args.min_count is not None else MIN_SERIES_COUNTS[resolution]
        usdtry = usdtry_bars(start, end, resolution)
        log(f"  shared USDTRY: {len(usdtry)} bars")

        if "currency" in kinds:
            for iso in C.CURRENCIES:
                code = f"{iso}TRY"
                if not want(code, selected):
                    continue
                bars, source = currency_bars(iso, usdtry, start, end, resolution)
                total += emit(code, resolution, bars, writer, source, manifest, min_count, listing_starts)

        if "gold" in kinds:
            xau = yahoo_bars("GC=F", start, end, resolution)
            for code, grams, fineness in C.GOLD_PRODUCTS:
                if not want(code, selected):
                    continue
                scale = grams * fineness / C.TROY_OUNCE_TO_GRAM
                bars = multiply_bars(xau, usdtry, scale=scale)
                total += emit(code, resolution, bars, writer, "Yahoo GC=F×USDTRY", manifest, min_count, listing_starts)

        if "crypto" in kinds:
            for code, yahoo_symbol, binance_symbol in C.CRYPTOS:
                if not want(code, selected):
                    continue
                bars, source = crypto_bars(
                    code,
                    yahoo_symbol,
                    binance_symbol,
                    usdtry,
                    start,
                    end,
                    resolution,
                )
                total += emit(code, resolution, bars, writer, source, manifest, min_count, listing_starts)

        if "stock" in kinds:
            for code in C.STOCKS:
                if not want(code, selected):
                    continue
                bars = yahoo_bars(f"{code}.IS", start, end, resolution, adjusted=True)
                source = "Yahoo adjusted"
                if resolution == "1d" and needs_daily_fallback(bars, end):
                    fallback = close_bars(
                        F.isyatirim_close(
                            code,
                            start.date(),
                            (end - dt.timedelta(days=1)).date(),
                        )
                    )
                    fallback.update(bars)
                    bars = fallback
                    source += " + İş Yatırım close-only"
                total += emit(code, resolution, bars, writer, source, manifest, min_count, listing_starts)

        if "global" in kinds:
            for code, symbol in C.GLOBAL_STOCKS:
                if not want(code, selected):
                    continue
                bars = yahoo_bars(symbol, start, end, resolution, adjusted=True)
                total += emit(code, resolution, bars, writer, "Yahoo adjusted", manifest, min_count, listing_starts)

    expected_series = {f"{code}|{resolution}" for code in expected for resolution in resolutions}
    produced = set(manifest["series"])
    manifest["empty"].extend(sorted(expected_series - produced))
    manifest["empty"] = sorted(set(manifest["empty"]))
    manifest["invalid"] = sorted(set(manifest["invalid"]))
    manifest["total_rows"] = total
    manifest["quality_pass"] = not any(
        manifest[name] for name in ("empty", "invalid", "short", "gaps")
    )

    if args.manifest:
        with open(args.manifest, "w", encoding="utf-8") as handle:
            json.dump(manifest, handle, indent=2, sort_keys=True)
        log(f"\nmanifest: {args.manifest}")
    log(
        f"\nDONE rows={total} series={len(manifest['series'])}/{len(expected_series)} "
        f"empty={len(manifest['empty'])} invalid={len(manifest['invalid'])} "
        f"short={len(manifest['short'])} gaps={len(manifest['gaps'])} "
        f"normalized_series={len(manifest['normalized'])} "
        f"listing_exceptions={len(manifest['listing_exceptions'])}"
    )
    if args.strict and not manifest["quality_pass"]:
        raise SystemExit(2)


if __name__ == "__main__":
    main()
