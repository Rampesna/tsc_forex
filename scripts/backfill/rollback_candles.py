#!/usr/bin/env python3
"""Restore candle rows captured by load_candles.py --backup-table."""

import argparse
import os
import re

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
        application_name="tsc-candle-rollback",
    )


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("backup_table")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--drop-backup", action="store_true")
    args = parser.parse_args()
    if not re.fullmatch(r"[a-z][a-z0-9_]{0,62}", args.backup_table):
        parser.error("backup table name must match [a-z][a-z0-9_]{0,62}")

    table = sql.Identifier(args.backup_table)
    conn = connect()
    locked = False
    try:
        with conn.cursor() as cur:
            cur.execute("SELECT pg_try_advisory_lock(%s)", (LOCK_KEY,))
            locked = bool(cur.fetchone()[0])
            if not locked:
                raise SystemExit("another candle loader holds the advisory lock")
            cur.execute(
                """
                SELECT column_name
                FROM information_schema.columns
                WHERE table_schema = current_schema() AND table_name = %s
                """,
                (args.backup_table,),
            )
            columns = {row[0] for row in cur.fetchall()}
            required = {
                "code", "resolution", "bucket", "old_open", "old_high",
                "old_low", "old_close", "old_count", "existed",
            }
            if not required.issubset(columns):
                raise SystemExit("table is not a compatible candle backup")
            cur.execute(sql.SQL("SELECT count(*), count(*) FILTER (WHERE existed) FROM {}").format(table))
            total, existing = cur.fetchone()
            print(f"backup rows: {total}; rows to restore: {existing}; inserted rows to remove: {total - existing}")
            if args.dry_run:
                conn.rollback()
                print("dry-run: backup is compatible; database unchanged")
                return

            cur.execute(
                sql.SQL(
                    """
                    DELETE FROM candles c
                    USING {} b
                    WHERE c.code = b.code
                      AND c.resolution = b.resolution
                      AND c.bucket = b.bucket
                    """
                ).format(table)
            )
            deleted = cur.rowcount
            cur.execute(
                sql.SQL(
                    """
                    INSERT INTO candles
                      (code,resolution,bucket,open,high,low,close,count)
                    SELECT
                      code,resolution,bucket,old_open,old_high,old_low,old_close,old_count
                    FROM {}
                    WHERE existed
                    ORDER BY code,resolution,bucket
                    """
                ).format(table)
            )
            restored = cur.rowcount
            if args.drop_backup:
                cur.execute(sql.SQL("DROP TABLE {}").format(table))
        conn.commit()
        print(f"rollback complete: deleted={deleted}; restored={restored}")
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
