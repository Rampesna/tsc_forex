#!/usr/bin/env python3
"""Backfill historical DAILY prices into the tsc_forex `price_ticks` table.

Charts are served straight from price_ticks (OHLC is aggregated at query time),
so backfilling = inserting one TRY-denominated row per instrument per day with
provider='backfill'. Live 'yahoo' rows are never touched. Re-running is
idempotent: existing provider='backfill' rows for a code are deleted first.

DB connection (env, with sensible defaults for the SSH tunnel):
  PGHOST      (default 127.0.0.1)
  PGPORT      (default 15432)
  PGDATABASE  (default tsc_forex)
  PGUSER      (default tsc_forex)
  PGPASSWORD  (required)

Usage:
  PGPASSWORD=... python3 backfill.py                       # everything, 1995->yesterday
  PGPASSWORD=... python3 backfill.py --kinds gold,currency
  PGPASSWORD=... python3 backfill.py --codes USDTRY,GRAM_GOLD,BTC --start-year 2015
  PGPASSWORD=... python3 backfill.py --dry-run             # fetch + report, no writes
"""

import argparse
import bisect
import datetime as dt
import os
import sys
import time
from typing import Callable, Dict, List, Tuple

import psycopg2
from psycopg2.extras import execute_values

import catalog as C
import fetch as F

# End-of-day timestamp assigned to each daily bar (UTC). ~TR market close.
BAR_HOUR_UTC = 18

# USD-pegged currencies (units per 1 USD). Used when no market series exists;
# XXXTRY = USDTRY / peg. These pegs have held for decades, so a flat multiple
# of USD/TRY is a faithful historical approximation for charting.
USD_PEGS = {
    "SAR": 3.75, "AED": 3.6725, "QAR": 3.64, "BHD": 0.376,
    "OMR": 0.38449, "JOD": 0.709,
}


def log(*a):
    print(*a, flush=True)


def make_ffill(values: Dict[str, float]) -> Callable[[str], float]:
    """Return f(date_iso) -> value at date or most recent earlier date (0 if none)."""
    keys = sorted(values.keys())
    def f(d: str) -> float:
        i = bisect.bisect_right(keys, d) - 1
        if i < 0:
            return 0.0
        return values[keys[i]]
    return f


def currency_try_series(iso: str, usdtry: Dict[str, float],
                        start: dt.date, end: dt.date, exotic_fx: bool = False) -> Dict[str, float]:
    """Historical <ISO>TRY. ECB(Frankfurter) first, then USD peg, then single Yahoo cross."""
    if iso == "USD":
        return dict(usdtry)

    # 1) ECB / Frankfurter (long, clean daily history since 1999) for supported ISOs
    direct = F.frankfurter_try(iso, start, end)
    if len(direct) >= 200:
        return direct

    # 2) USD-pegged: XXXTRY = USDTRY / peg (flat multiple of USD/TRY)
    if iso in USD_PEGS:
        peg = USD_PEGS[iso]
        return {d: v / peg for d, v in usdtry.items() if v > 0}

    # 3) Yahoo cross (opt-in): heavy on Yahoo (throttle risk) and sparse for exotics.
    if not exotic_fx:
        return direct
    # "<ISO>=X" == USD<ISO> (units per USD) -> XXXTRY = USDTRY / rate
    units_per_usd = F.yahoo_daily(f"{iso}=X", start, end)
    if len(units_per_usd) < 30:
        return direct  # give up (maybe empty); no long backoff burned
    ftry = make_ffill(usdtry)
    out: Dict[str, float] = {}
    for d, rate in units_per_usd.items():
        t = ftry(d)
        if rate > 0 and t > 0:
            out[d] = t / rate
    return F.merge_prefer(out, direct)


def build_all(kinds: List[str], codes_filter, start: dt.date, end: dt.date, exotic_fx: bool = False):
    """Fetch everything and return {code: (kind, {date: price})}."""
    result: Dict[str, Tuple[str, Dict[str, float]]] = {}

    def want(code: str) -> bool:
        return codes_filter is None or code in codes_filter

    # --- shared legs: USDTRY, XAUUSD ---
    need_gold = "gold" in kinds or (codes_filter and any(c.endswith("_GOLD") or c.startswith("GRAM_") for c in codes_filter))
    need_cur = "currency" in kinds
    need_crypto = "crypto" in kinds
    need_stock = "stock" in kinds

    log("Fetching USDTRY (ECB / Frankfurter)...")
    usdtry = F.frankfurter_try("USD", start, end)
    if len(usdtry) < 200:  # ECB unavailable -> fall back to Yahoo
        usdtry = F.merge_prefer(usdtry, F.yahoo_daily("USDTRY=X", start, end))
    log(f"  USDTRY points: {len(usdtry)}  range: {min(usdtry) if usdtry else '-'}..{max(usdtry) if usdtry else '-'}")

    xauusd: Dict[str, float] = {}
    if need_gold:
        log("Fetching XAUUSD (Yahoo GC=F)...")
        xauusd = F.yahoo_daily("GC=F", start, end)
        log(f"  XAUUSD points: {len(xauusd)}")

    # --- currencies ---
    if need_cur:
        for iso in C.CURRENCIES:
            code = f"{iso}TRY"
            if not want(code):
                continue
            s = currency_try_series(iso, usdtry, start, end, exotic_fx=exotic_fx)
            if s:
                result[code] = ("currency", s)
            log(f"  {code}: {len(s)} pts")

    # --- gold products (derived) ---
    if need_gold and xauusd and usdtry:
        f_usdtry = make_ffill(usdtry)
        for code, gram, fine in C.GOLD_PRODUCTS:
            if not want(code):
                continue
            series: Dict[str, float] = {}
            for d, xu in xauusd.items():
                rate = f_usdtry(d)
                if xu > 0 and rate > 0:
                    series[d] = C.gold_price(xu, rate, gram, fine)
            if series:
                result[code] = ("gold", series)
            log(f"  {code}: {len(series)} pts")

    # --- crypto (USD x USDTRY) ---
    if need_crypto:
        f_usdtry = make_ffill(usdtry)
        for code, ysym, bsym in C.CRYPTOS:
            if not want(code):
                continue
            usd = F.binance_daily(bsym, start, end)
            if len(usd) < 30:
                usd = F.merge_prefer(usd, F.yahoo_daily(ysym, start, end))
            series: Dict[str, float] = {}
            for d, u in usd.items():
                rate = f_usdtry(d)
                if u > 0 and rate > 0:
                    series[d] = u * rate
            if series:
                result[code] = ("crypto", series)
            log(f"  {code}: {len(series)} pts")
            time.sleep(F.REQUEST_DELAY)

    # --- BIST stocks (native TRY, adjusted close) ---
    if need_stock:
        for code in C.STOCKS:
            if not want(code):
                continue
            s = F.yahoo_daily(f"{code}.IS", start, end, adjusted=True)
            if s:
                result[code] = ("stock", s)
            log(f"  {code}: {len(s)} pts")
            time.sleep(F.REQUEST_DELAY)

    return result


def to_rows(code: str, kind: str, series: Dict[str, float], provider: str):
    spread = C.SPREAD_GOLD if kind == "gold" else C.SPREAD_OTHER
    rows = []
    for d, price in series.items():
        if price is None or price <= 0:
            continue
        y, m, day = (int(x) for x in d.split("-"))
        created = dt.datetime(y, m, day, BAR_HOUR_UTC, 0, 0, tzinfo=dt.timezone.utc)
        bid, ask = C.bid_ask(price, spread)
        rows.append((code, kind, price, bid, ask, provider, created))
    return rows


def write_db(conn, data, provider: str):
    total = 0
    with conn.cursor() as cur:
        for code, (kind, series) in sorted(data.items()):
            rows = to_rows(code, kind, series, provider)
            if not rows:
                continue
            cur.execute(
                "DELETE FROM price_ticks WHERE provider=%s AND code=%s",
                (provider, code),
            )
            execute_values(
                cur,
                "INSERT INTO price_ticks (id, code, kind, price, bid, ask, provider, created_at) "
                "VALUES %s",
                rows,
                template="(gen_random_uuid(), %s, %s, %s, %s, %s, %s, %s)",
                page_size=1000,
            )
            conn.commit()
            total += len(rows)
            log(f"  wrote {code}: {len(rows)} rows")
    return total


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--start-year", type=int, default=1995)
    ap.add_argument("--end", default=None, help="YYYY-MM-DD (default: yesterday)")
    ap.add_argument("--kinds", default="currency,gold,crypto,stock")
    ap.add_argument("--codes", default=None, help="comma list to restrict")
    ap.add_argument("--provider", default="backfill")
    ap.add_argument("--exotic-fx", action="store_true",
                    help="also try Yahoo per-currency crosses for exotics (heavier on Yahoo)")
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()

    start = dt.date(args.start_year, 1, 1)
    end = (dt.date.today() - dt.timedelta(days=1)) if not args.end else dt.date.fromisoformat(args.end)
    kinds = [k.strip() for k in args.kinds.split(",") if k.strip()]
    codes_filter = set(c.strip() for c in args.codes.split(",")) if args.codes else None

    log(f"Backfill {start}..{end}  kinds={kinds}  codes={codes_filter or 'ALL'}  provider={args.provider}")
    data = build_all(kinds, codes_filter, start, end, exotic_fx=args.exotic_fx)

    npts = sum(len(s) for _, s in data.values())
    log(f"\nFetched {len(data)} instruments, {npts} total daily points.")
    covered = {c: (min(s), max(s), len(s)) for c, (_, s) in data.items() if s}
    for c in sorted(covered):
        lo, hi, n = covered[c]
        log(f"  {c:16s} {lo} .. {hi}  ({n})")

    if args.dry_run:
        log("\n[dry-run] no DB writes.")
        return

    pw = os.getenv("PGPASSWORD")
    if not pw:
        log("ERROR: set PGPASSWORD (and optionally PGHOST/PGPORT/PGUSER/PGDATABASE).")
        sys.exit(2)
    conn = psycopg2.connect(
        host=os.getenv("PGHOST", "127.0.0.1"),
        port=int(os.getenv("PGPORT", "15432")),
        dbname=os.getenv("PGDATABASE", "tsc_forex"),
        user=os.getenv("PGUSER", "tsc_forex"),
        password=pw,
        connect_timeout=15,
    )
    try:
        total = write_db(conn, data, args.provider)
        log(f"\nDONE. Inserted {total} rows (provider={args.provider}).")
    finally:
        conn.close()


if __name__ == "__main__":
    main()
