# Candle Backfill Runbook

The workstation fetches provider data; production only validates and loads the
result. Never fetch the full catalog from production because provider egress can
be rate-limited.

## 1. Fetch and enforce quality

```bash
python3 scripts/backfill/backfill_candles.py \
  --kinds currency,gold,crypto,stock,global \
  --resolutions 1d \
  --start-year 2005 \
  --strict \
  --manifest /tmp/candles_manifest.json \
  > /tmp/candles.csv
```

`--strict` fails on an empty or unexpectedly short series, invalid OHLC, or a
large date gap. A genuinely recent listing can be declared in a reviewed JSON
file (`{"CODE":"YYYY-MM-DD"}`) with `--listing-exceptions`; accepted exceptions
remain visible in the manifest.

## 2. Production dry-run

Copy the CSV, manifest, loader, and `requirements.txt` to an operations
directory. Point the loader at PostgreSQL with `PGHOST`, `PGPORT`, `PGDATABASE`,
`PGUSER`, and `PGPASSWORD`, then run:

```bash
python3 load_candles.py /tmp/candles.csv \
  --manifest /tmp/candles_manifest.json \
  --dry-run
```

The staging gate independently rejects malformed OHLC, duplicate keys, unknown
resolutions, and current forming buckets.

## 3. Backup and idempotent load

Use a unique lowercase backup table name:

```bash
python3 load_candles.py /tmp/candles.csv \
  --manifest /tmp/candles_manifest.json \
  --backup-table candle_backup_YYYYMMDD
```

The loader holds a PostgreSQL advisory lock, sorts code/resolution groups, uses
short transactions, retries deadlocks, and upserts on
`(code,resolution,bucket)`.

## 4. Canary and rollback

Verify `NVDA`, `AAPL`, `THYAO`, `USDTRY`, `GRAM_GOLD`, and `BTC` through the
coverage audit and the `1Y`, `5Y`, and `MAX` API paths before approving the full
load.

To inspect or restore the exact key set:

```bash
python3 rollback_candles.py candle_backup_YYYYMMDD --dry-run
python3 rollback_candles.py candle_backup_YYYYMMDD
```

The rollback removes rows inserted by the load and restores overwritten rows.
Keep the backup table until post-deploy verification is complete.
