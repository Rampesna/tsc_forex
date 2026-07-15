#!/usr/bin/env python3
"""Check whether the LIVE engine (provider='yahoo') is getting gold + stocks.

If ONS_GOLD/THYAO have fresh yahoo rows, the cluster egress reaches Yahoo's
query API and we can run the historical backfill from inside a pod.
"""
import os
import psycopg2

CHECK = ["USDTRY", "ONS_GOLD", "GRAM_GOLD", "THYAO", "GARAN", "BTC", "RUBTRY", "PKRTRY"]


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
    print("=== latest LIVE (provider=yahoo) tick per code ===")
    for code in CHECK:
        cur.execute("SELECT count(*), max(created_at), "
                    "(SELECT price FROM price_ticks WHERE code=%s AND provider='yahoo' ORDER BY created_at DESC LIMIT 1) "
                    "FROM price_ticks WHERE code=%s AND provider='yahoo' "
                    "AND created_at > now() - interval '2 hours'",
                    (code, code))
        n, mx, price = cur.fetchone()
        print(f"  {code:10s} last2h={n:<6d} latest={str(mx)[:19] if mx else '-':19s} price={price}")
    conn.close()


if __name__ == "__main__":
    main()
