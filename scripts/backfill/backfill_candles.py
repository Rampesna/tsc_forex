#!/usr/bin/env python3
"""Backfill historical DAILY (1d) candles for gold, BIST and global equities.

Charts read straight from the compact `candles` table (1w/1mo are derived in
memory from the 1d base), so writing 1d candles is enough to light up the
1A/1Y/5Y/MAX ranges without touching the live pipeline.

This script FETCHES from Yahoo (run it from a machine whose IP isn't rate
limited — e.g. a laptop, not the shared server egress) and EMITS CSV rows on
stdout for the `candles` table:

    code,resolution,bucket_iso,open,high,low,close,count

Values are in each instrument's native currency (gold/BIST in TRY, global
equities in USD) to match the live engine. Today's bucket is skipped so the
loader never clobbers the live forming candle.

Usage (fetch → load via SSH, see load_candles.sh):
    python3 backfill_candles.py --kinds gold,stock,global --start-year 2005 > candles.csv
"""

import argparse
import bisect
import datetime as dt
import sys
from typing import Dict

import catalog as C
import fetch as F


def log(*a):
    print(*a, file=sys.stderr, flush=True)


def ffill(values: Dict[str, float]):
    keys = sorted(values.keys())

    def f(d: str) -> float:
        i = bisect.bisect_right(keys, d) - 1
        return values[keys[i]] if i >= 0 else 0.0

    return f


def emit(code: str, ohlc: Dict[str, tuple], today_iso: str, w):
    n = 0
    for d in sorted(ohlc.keys()):
        if d >= today_iso:  # never write today's (live) bucket
            continue
        o, h, l, c = ohlc[d]
        if not all(x and x > 0 for x in (o, h, l, c)):
            continue
        bucket = f"{d} 00:00:00+00"
        w.write(f"{code},1d,{bucket},{o:.10g},{h:.10g},{l:.10g},{c:.10g},0\n")
        n += 1
    log(f"  {code}: {n} candles")
    return n


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--start-year", type=int, default=2005)
    ap.add_argument("--kinds", default="gold,stock,global")
    ap.add_argument("--codes", default=None, help="comma list to restrict")
    args = ap.parse_args()

    start = dt.date(args.start_year, 1, 1)
    end = dt.date.today() - dt.timedelta(days=1)
    today_iso = dt.date.today().isoformat()
    kinds = {k.strip() for k in args.kinds.split(",") if k.strip()}
    codes_filter = {c.strip() for c in args.codes.split(",")} if args.codes else None

    def want(code):
        return codes_filter is None or code in codes_filter

    w = sys.stdout
    total = 0

    # --- Gold (TRY) : GC=F OHLC (USD) × USDTRY close ---
    if "gold" in kinds:
        log("Fetching USDTRY (Frankfurter)...")
        usdtry = F.frankfurter_try("USD", start, end)
        if len(usdtry) < 200:
            usdtry = F.merge_prefer(usdtry, F.yahoo_daily("USDTRY=X", start, end))
        log(f"  USDTRY: {len(usdtry)} pts")
        log("Fetching XAUUSD (GC=F) OHLC...")
        xau = F.yahoo_ohlc("GC=F", start, end)
        log(f"  GC=F: {len(xau)} pts")
        f_usd = ffill(usdtry)
        for code, gram, fine in C.GOLD_PRODUCTS:
            if not want(code):
                continue
            ohlc = {}
            for d, (o, h, l, c) in xau.items():
                rate = f_usd(d)
                if rate <= 0:
                    continue
                ohlc[d] = (
                    C.gold_price(o, rate, gram, fine),
                    C.gold_price(h, rate, gram, fine),
                    C.gold_price(l, rate, gram, fine),
                    C.gold_price(c, rate, gram, fine),
                )
            total += emit(code, ohlc, today_iso, w)

    # --- BIST equities (native TRY, adjusted) ---
    if "stock" in kinds:
        for code in C.STOCKS:
            if not want(code):
                continue
            ohlc = F.yahoo_ohlc(f"{code}.IS", start, end, adjusted=True)
            total += emit(code, ohlc, today_iso, w)

    # --- Global equities (native USD, adjusted) ---
    if "global" in kinds:
        for code, ysym in C.GLOBAL_STOCKS:
            if not want(code):
                continue
            ohlc = F.yahoo_ohlc(ysym, start, end, adjusted=True)
            total += emit(code, ohlc, today_iso, w)

    log(f"\nDONE. {total} candle rows emitted.")


if __name__ == "__main__":
    main()
