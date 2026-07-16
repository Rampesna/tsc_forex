#!/usr/bin/env python3
"""Audit historical coverage in both price_ticks and candles.

Groups by provider, then for provider='backfill' lists per-code coverage
(count, min/max date). The canonical chart audit compares every catalog code
against the compact candles table and can emit a machine-readable gap manifest.
"""
import argparse
import datetime as dt
import json
import os
import psycopg2

import catalog as C

EXPECTED = {}
for iso in C.CURRENCIES:
    EXPECTED[f"{iso}TRY"] = "currency"
for code, _, _ in C.GOLD_PRODUCTS:
    EXPECTED[code] = "gold"
for code, _, _ in C.CRYPTOS:
    EXPECTED[code] = "crypto"
for code in C.STOCKS:
    EXPECTED[code] = "stock"
for code, _ in C.GLOBAL_STOCKS:
    EXPECTED[code] = "global_stock"


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--resolution", default="1d")
    ap.add_argument("--json-out", help="optional path for the coverage/gap manifest")
    ap.add_argument("--short-threshold", type=int, default=20)
    args = ap.parse_args()

    conn = psycopg2.connect(
        host=os.getenv("PGHOST", "127.0.0.1"),
        port=int(os.getenv("PGPORT", "5432")),
        dbname=os.getenv("PGDATABASE", "tsc_forex"),
        user=os.getenv("PGUSER", "tsc_forex"),
        password=os.getenv("PGPASSWORD"),
        connect_timeout=15,
    )
    cur = conn.cursor()

    print("=== providers overview ===")
    cur.execute("SELECT provider, count(*), min(created_at), max(created_at) "
                "FROM price_ticks GROUP BY provider ORDER BY provider;")
    for prov, n, lo, hi in cur.fetchall():
        print(f"  {str(prov):12s} rows={n:<10d} {lo} .. {hi}")

    print("\n=== backfill coverage per code ===")
    cur.execute("SELECT code, kind, count(*), min(created_at), max(created_at) "
                "FROM price_ticks WHERE provider='backfill' "
                "GROUP BY code, kind ORDER BY code;")
    have = {}
    for code, kind, n, lo, hi in cur.fetchall():
        have[code] = (kind, n, lo, hi)
        print(f"  {code:16s} {kind:9s} n={n:<6d} {str(lo)[:10]} .. {str(hi)[:10]}")

    print("\n=== catalog vs. backfill ===")
    by_kind_missing = {}
    by_kind_done = {}
    for code, kind in sorted(EXPECTED.items()):
        if code in have:
            by_kind_done.setdefault(kind, []).append(code)
        else:
            by_kind_missing.setdefault(kind, []).append(code)

    for kind in ("currency", "gold", "crypto", "stock", "global_stock"):
        done = by_kind_done.get(kind, [])
        miss = by_kind_missing.get(kind, [])
        total = len(done) + len(miss)
        print(f"\n  [{kind}] {len(done)}/{total} present")
        if miss:
            print(f"    MISSING ({len(miss)}): {', '.join(miss)}")

    # Also flag codes present in DB that aren't in the catalog (renamed/removed).
    extra = sorted(set(have) - set(EXPECTED))
    if extra:
        print(f"\n  EXTRA backfill codes not in catalog: {', '.join(extra)}")

    print(f"\n=== candles coverage ({args.resolution}) ===")
    cur.execute(
        """
        SELECT code, count(*), min(bucket), max(bucket)
        FROM candles
        WHERE resolution = %s
        GROUP BY code
        ORDER BY code
        """,
        (args.resolution,),
    )
    candle_have = {
        code: {"count": int(n), "from": lo.isoformat(), "to": hi.isoformat()}
        for code, n, lo, hi in cur.fetchall()
    }

    # Large holes are suspicious even for exchange-traded instruments. Ordinary
    # weekends/holidays are intentionally ignored by using a 14-day threshold.
    cur.execute(
        """
        WITH ordered AS (
          SELECT code, bucket,
                 lag(bucket) OVER (PARTITION BY code ORDER BY bucket) AS prev
          FROM candles
          WHERE resolution = %s
        )
        , gaps AS (
          SELECT code, prev, bucket, bucket - prev AS gap
          FROM ordered
          WHERE prev IS NOT NULL AND bucket - prev > interval '14 days'
        )
        SELECT DISTINCT ON (code)
               code,
               count(*) OVER (PARTITION BY code),
               gap,
               prev,
               bucket
        FROM gaps
        ORDER BY code, gap DESC
        """,
        (args.resolution,),
    )
    large_gaps = {
        code: {
            "count": int(n),
            "max_gap": str(gap),
            "from": previous.isoformat(),
            "to": current.isoformat(),
        }
        for code, n, gap, previous, current in cur.fetchall()
    }
    cur.execute(
        """
        SELECT code, count(*)
        FROM candles
        WHERE resolution = %s
          AND (
            open <= 0 OR high <= 0 OR low <= 0 OR close <= 0
            OR high < GREATEST(open, close)
            OR low > LEAST(open, close)
            OR high < low
          )
        GROUP BY code
        ORDER BY code
        """,
        (args.resolution,),
    )
    invalid_ohlc = {code: int(count) for code, count in cur.fetchall()}

    stale_before = dt.datetime.now(dt.timezone.utc) - dt.timedelta(days=10)
    missing, short, stale = [], [], []
    by_kind = {}
    for code, kind in sorted(EXPECTED.items()):
        row = candle_have.get(code)
        status = "ok"
        if row is None:
            missing.append(code)
            status = "missing"
            print(f"  {code:16s} {kind:12s} MISSING")
        else:
            last = dt.datetime.fromisoformat(row["to"])
            if row["count"] < args.short_threshold:
                short.append(code)
                status = "short"
            if last < stale_before:
                stale.append(code)
                status = "stale" if status == "ok" else status + "+stale"
            gap_note = ""
            if code in large_gaps:
                gap_note = f" gaps>14d={large_gaps[code]['count']} max={large_gaps[code]['max_gap']}"
            if code in invalid_ohlc:
                gap_note += f" INVALID_OHLC={invalid_ohlc[code]}"
            print(
                f"  {code:16s} {kind:12s} n={row['count']:<6d} "
                f"{row['from'][:10]} .. {row['to'][:10]} {status}{gap_note}"
            )
        by_kind.setdefault(kind, {"total": 0, "present": 0})
        by_kind[kind]["total"] += 1
        if row is not None:
            by_kind[kind]["present"] += 1

    print("\n=== candle audit summary ===")
    for kind, counts in sorted(by_kind.items()):
        print(f"  {kind:12s} {counts['present']}/{counts['total']} present")
    print(f"  missing={len(missing)} short={len(short)} stale={len(stale)}")
    print(f"  invalid_ohlc_series={len(invalid_ohlc)}")

    manifest = {
        "generated_at": dt.datetime.now(dt.timezone.utc).isoformat(),
        "resolution": args.resolution,
        "catalog_total": len(EXPECTED),
        "by_kind": by_kind,
        "missing": missing,
        "short": short,
        "stale": stale,
        "large_gaps": large_gaps,
        "invalid_ohlc": invalid_ohlc,
        "coverage": {code: candle_have.get(code) for code in sorted(EXPECTED)},
    }
    if args.json_out:
        with open(args.json_out, "w", encoding="utf-8") as f:
            json.dump(manifest, f, indent=2, sort_keys=True)
        print(f"\nmanifest written: {args.json_out}")

    conn.close()


if __name__ == "__main__":
    main()
