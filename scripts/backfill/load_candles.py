#!/usr/bin/env python3
"""Idempotently load candle CSV through PostgreSQL staging.

The loader is safe to rerun. It validates every staged row, serializes loaders
with an advisory lock, applies one code/resolution per short transaction and
retries deadlocks. Optional key-scoped backup tables make production rollback
possible without copying the entire candles table.
"""

import argparse
import json
import os
import re
import time

import psycopg2
from psycopg2 import sql

LOCK_KEY = 917326452


def connect():
    return psycopg2.connect(
        host=os.getenv("PGHOST", "127.0.0.1"),
        port=int(os.getenv("PGPORT", "15432")),
        dbname=os.getenv("PGDATABASE", "tsc_forex"),
        user=os.getenv("PGUSER", "tsc_forex"),
        password=os.getenv("PGPASSWORD"),
        connect_timeout=20,
        application_name="tsc-candle-loader",
    )


def validate_manifest(path: str, allow_incomplete: bool):
    if not path:
        return
    with open(path, encoding="utf-8") as handle:
        manifest = json.load(handle)
    if not manifest.get("quality_pass") and not allow_incomplete:
        raise SystemExit(
            "quality manifest failed; inspect empty/invalid/short/gap series or pass "
            "--allow-incomplete for an explicitly targeted partial load"
        )


def create_stage(conn, csv_path: str):
    with conn.cursor() as cur:
        cur.execute(
            """
            CREATE TEMP TABLE candle_stage (
              code text NOT NULL,
              resolution text NOT NULL,
              bucket timestamptz NOT NULL,
              open double precision NOT NULL,
              high double precision NOT NULL,
              low double precision NOT NULL,
              close double precision NOT NULL,
              count bigint NOT NULL
            ) ON COMMIT PRESERVE ROWS
            """
        )
        with open(csv_path, "r", encoding="utf-8") as handle:
            cur.copy_expert(
                """
                COPY candle_stage
                  (code,resolution,bucket,open,high,low,close,count)
                FROM STDIN WITH (FORMAT csv)
                """,
                handle,
            )
        cur.execute(
            """
            SELECT
              count(*) FILTER (
                WHERE open <= 0 OR high <= 0 OR low <= 0 OR close <= 0
                   OR high < GREATEST(open, close)
                   OR low > LEAST(open, close)
                   OR high < low
              ),
              count(*) - count(DISTINCT (code, resolution, bucket)),
              count(*)
            FROM candle_stage
            """
        )
        invalid, duplicates, total = cur.fetchone()
        if invalid or duplicates:
            conn.rollback()
            raise SystemExit(
                f"staging quality gate failed: invalid={invalid} duplicates={duplicates}"
            )
        cur.execute(
            """
            SELECT count(*)
            FROM candle_stage
            WHERE resolution NOT IN ('5m','1h','1d')
               OR (resolution = '1d' AND bucket >= date_trunc('day', now()))
               OR (resolution = '1h' AND bucket >= date_trunc('hour', now()))
               OR (
                 resolution = '5m'
                 AND bucket >= to_timestamp(
                   floor(extract(epoch FROM now()) / 300) * 300
                 )
               )
            """
        )
        forming = cur.fetchone()[0]
        if forming:
            conn.rollback()
            raise SystemExit(f"staging includes {forming} forming/unsupported candles")
    conn.commit()
    return total


def create_backup(conn, table_name: str):
    if not re.fullmatch(r"[a-z][a-z0-9_]{0,62}", table_name):
        raise SystemExit("backup table name must match [a-z][a-z0-9_]{0,62}")
    with conn.cursor() as cur:
        cur.execute(
            sql.SQL(
                """
                CREATE TABLE {} AS
                SELECT
                  s.code,
                  s.resolution,
                  s.bucket,
                  c.open AS old_open,
                  c.high AS old_high,
                  c.low AS old_low,
                  c.close AS old_close,
                  c.count AS old_count,
                  c.code IS NOT NULL AS existed,
                  now() AS backed_up_at
                FROM candle_stage s
                LEFT JOIN candles c
                  ON c.code = s.code
                 AND c.resolution = s.resolution
                 AND c.bucket = s.bucket
                ORDER BY s.code, s.resolution, s.bucket
                """
            ).format(sql.Identifier(table_name))
        )
        rows = cur.rowcount
    conn.commit()
    print(f"backup table {table_name}: {rows} rows")


UPSERT = """
INSERT INTO candles (code,resolution,bucket,open,high,low,close,count)
SELECT code,resolution,bucket,open,high,low,close,count
FROM candle_stage
WHERE code = %s AND resolution = %s
ORDER BY bucket
ON CONFLICT (code,resolution,bucket) DO UPDATE SET
  open = EXCLUDED.open,
  high = EXCLUDED.high,
  low = EXCLUDED.low,
  close = EXCLUDED.close,
  count = GREATEST(candles.count, EXCLUDED.count)
"""


def apply_groups(conn, retries: int):
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT code, resolution, count(*)
            FROM candle_stage
            GROUP BY code, resolution
            ORDER BY code, resolution
            """
        )
        groups = cur.fetchall()
    conn.commit()

    done = 0
    for code, resolution, count in groups:
        for attempt in range(retries + 1):
            try:
                with conn.cursor() as cur:
                    cur.execute("SET LOCAL lock_timeout = '15s'")
                    cur.execute("SET LOCAL statement_timeout = '5min'")
                    cur.execute(UPSERT, (code, resolution))
                conn.commit()
                done += count
                print(f"loaded {code:16s} {resolution:3s}: {count}")
                break
            except psycopg2.Error as exc:
                conn.rollback()
                if exc.pgcode not in ("40P01", "55P03") or attempt >= retries:
                    raise
                wait = min(2 ** attempt, 15)
                print(
                    f"retry {code}|{resolution} after {exc.pgcode} "
                    f"({attempt + 1}/{retries})"
                )
                time.sleep(wait)
    return done


def audit(conn):
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT s.code, s.resolution, count(*), min(c.bucket), max(c.bucket)
            FROM (SELECT DISTINCT code, resolution FROM candle_stage) s
            JOIN candles c USING (code, resolution)
            GROUP BY s.code, s.resolution
            ORDER BY s.code, s.resolution
            """
        )
        return cur.fetchall()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("csv")
    parser.add_argument("--manifest")
    parser.add_argument("--allow-incomplete", action="store_true")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--backup-table")
    parser.add_argument("--deadlock-retries", type=int, default=6)
    args = parser.parse_args()

    validate_manifest(args.manifest, args.allow_incomplete)
    conn = connect()
    locked = False
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT pg_try_advisory_lock(%s)", (LOCK_KEY,))
            locked = bool(cur.fetchone()[0])
        conn.commit()
        if not locked:
            raise SystemExit("another candle loader holds the advisory lock")

        total = create_stage(conn, args.csv)
        print(f"staged rows: {total}")
        if args.dry_run:
            print("dry-run: quality gate passed; database unchanged")
            return
        if args.backup_table:
            create_backup(conn, args.backup_table)
        loaded = apply_groups(conn, args.deadlock_retries)
        rows = audit(conn)
        print(f"loaded rows: {loaded}; audited series: {len(rows)}")
        for code, resolution, count, first, last in rows:
            print(f"  {code:16s} {resolution:3s} n={count:<7d} {first} .. {last}")
    finally:
        if locked:
            try:
                with conn.cursor() as cur:
                    cur.execute("SELECT pg_advisory_unlock(%s)", (LOCK_KEY,))
                conn.commit()
            except Exception:
                pass
        conn.close()


if __name__ == "__main__":
    main()
