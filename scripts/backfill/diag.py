#!/usr/bin/env python3
"""Report what backfill has actually written to price_ticks.

Groups by provider, then for provider='backfill' lists per-code coverage
(count, min/max date). Prints a summary vs. the full instrument catalog so
we can see exactly what's done and what's still missing.
"""
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


def main():
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

    for kind in ("currency", "gold", "crypto", "stock"):
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

    conn.close()


if __name__ == "__main__":
    main()
