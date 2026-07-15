"""Free historical daily data fetchers (no API keys).

Sources:
  - Yahoo Finance chart API  (FX pairs, gold futures, crypto, BIST equities)
  - Frankfurter (ECB)        (long-history fiat vs TRY, ~30 currencies since 1999)
  - Binance klines           (crypto fallback, USDT quote) via data-api.binance.vision
  - Stooq CSV                (optional; often behind a JS challenge)

Every fetcher returns a dict {date_iso: close_float} for daily bars.
"""

import csv
import io
import time
import datetime as dt
from typing import Dict

import requests

_UA = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/124.0 Safari/537.36"
)
_SESSION = requests.Session()
_SESSION.headers.update({"User-Agent": _UA, "Accept": "application/json,text/csv,*/*"})

# Polite delay between remote calls to avoid throttling.
REQUEST_DELAY = 0.6

# Global pacers: free feeds rate-limit bursts, so space requests and back off.
_YAHOO_MIN_INTERVAL = 2.5
_FRANK_MIN_INTERVAL = 0.8
_last_yahoo_at = 0.0
_last_frank_at = 0.0


def _pace(kind: str):
    global _last_yahoo_at, _last_frank_at
    if kind == "yahoo":
        gap, last = _YAHOO_MIN_INTERVAL, _last_yahoo_at
    else:
        gap, last = _FRANK_MIN_INTERVAL, _last_frank_at
    wait = gap - (time.monotonic() - last)
    if wait > 0:
        time.sleep(wait)
    if kind == "yahoo":
        _last_yahoo_at = time.monotonic()
    else:
        _last_frank_at = time.monotonic()

# Currencies covered by the ECB / Frankfurter feed (long daily history since 1999).
FRANKFURTER_CCYS = {
    "AUD", "BGN", "BRL", "CAD", "CHF", "CNY", "CZK", "DKK", "EUR", "GBP",
    "HKD", "HUF", "IDR", "ILS", "INR", "ISK", "JPY", "KRW", "MXN", "MYR",
    "NOK", "NZD", "PHP", "PLN", "RON", "SEK", "SGD", "THB", "TRY", "USD", "ZAR",
}


def _epoch(d: dt.date) -> int:
    return int(dt.datetime(d.year, d.month, d.day, tzinfo=dt.timezone.utc).timestamp())


def yahoo_daily(symbol: str, start: dt.date, end: dt.date,
                adjusted: bool = False) -> Dict[str, float]:
    """Daily closes from Yahoo chart API with pacing + 429 backoff. adjClose when adjusted."""
    params = {
        "period1": _epoch(start),
        "period2": _epoch(end + dt.timedelta(days=1)),
        "interval": "1d",
        "events": "div,splits",
        "includeAdjustedClose": "true",
    }
    delay = 3.0
    for attempt in range(9):
        host = "query1" if attempt % 2 == 0 else "query2"
        u = f"https://{host}.finance.yahoo.com/v8/finance/chart/{symbol}"
        try:
            _pace("yahoo")
            r = _SESSION.get(u, params=params, timeout=30)
            if r.status_code == 429:
                time.sleep(delay)
                delay = min(delay * 1.7, 45)
                continue
            if r.status_code != 200:
                time.sleep(REQUEST_DELAY)
                continue
            data = r.json()
            res = (data.get("chart") or {}).get("result") or []
            if not res:
                return {}
            node = res[0]
            ts = node.get("timestamp") or []
            quote = (node.get("indicators", {}).get("quote") or [{}])[0]
            closes = quote.get("close") or []
            adj = None
            if adjusted:
                adjnode = node.get("indicators", {}).get("adjclose") or []
                if adjnode:
                    adj = adjnode[0].get("adjclose")
            series = adj if (adjusted and adj) else closes
            out: Dict[str, float] = {}
            for t, c in zip(ts, series):
                if c is None:
                    continue
                d = dt.datetime.fromtimestamp(t, tz=dt.timezone.utc).date().isoformat()
                out[d] = float(c)
            return out
        except Exception:
            time.sleep(delay)
            delay = min(delay * 2, 20)
    return {}


def frankfurter_try(base_iso: str, start: dt.date, end: dt.date) -> Dict[str, float]:
    """<base_iso>TRY daily via ECB (Frankfurter). Returns {date: TRY per 1 base}."""
    if base_iso not in FRANKFURTER_CCYS or base_iso == "TRY":
        return {}
    # Two mirrors of the same ECB dataset; retry with backoff on throttle.
    hosts = ["https://api.frankfurter.app", "https://api.frankfurter.dev/v1"]
    delay = 2.0
    for attempt in range(6):
        host = hosts[attempt % len(hosts)]
        url = f"{host}/{start.isoformat()}..{end.isoformat()}"
        try:
            _pace("frank")
            r = _SESSION.get(url, params={"from": base_iso, "to": "TRY"}, timeout=30)
            if r.status_code == 200:
                rates = (r.json() or {}).get("rates") or {}
                out: Dict[str, float] = {}
                for d, obj in rates.items():
                    v = obj.get("TRY")
                    if v:
                        out[d] = float(v)
                return out
            if r.status_code in (429, 502, 503, 504):
                time.sleep(delay)
                delay = min(delay * 1.7, 30)
                continue
            return {}
        except Exception:
            time.sleep(delay)
            delay = min(delay * 1.7, 30)
    return {}


def binance_daily(symbol: str, start: dt.date, end: dt.date) -> Dict[str, float]:
    """Daily closes from Binance public data mirror (USDT quote)."""
    out: Dict[str, float] = {}
    cur = _epoch(start) * 1000
    end_ms = _epoch(end + dt.timedelta(days=1)) * 1000
    for host in ("https://data-api.binance.vision", "https://api.binance.com"):
        cur = _epoch(start) * 1000
        out = {}
        ok = False
        while cur < end_ms:
            try:
                r = _SESSION.get(host + "/api/v3/klines",
                                 params={"symbol": symbol, "interval": "1d",
                                         "startTime": cur, "endTime": end_ms, "limit": 1000},
                                 timeout=30)
                if r.status_code != 200:
                    break
                rows = r.json()
                if not rows:
                    ok = True
                    break
                for k in rows:
                    d = dt.datetime.fromtimestamp(int(k[0]) / 1000, tz=dt.timezone.utc).date().isoformat()
                    out[d] = float(k[4])
                cur = int(rows[-1][0]) + 86_400_000
                ok = True
                if len(rows) < 1000:
                    break
                time.sleep(0.2)
            except Exception:
                break
        if ok and out:
            return out
    return out


def stooq_daily(symbol: str, start: dt.date, end: dt.date) -> Dict[str, float]:
    """Daily closes from Stooq CSV (often JS-challenged; returns {} if blocked)."""
    try:
        r = _SESSION.get("https://stooq.com/q/d/l/",
                         params={"s": symbol, "i": "d",
                                 "d1": start.strftime("%Y%m%d"), "d2": end.strftime("%Y%m%d")},
                         timeout=30)
        if r.status_code != 200 or "Date" not in r.text[:64]:
            return {}
        out: Dict[str, float] = {}
        for row in csv.DictReader(io.StringIO(r.text)):
            c, d = row.get("Close"), row.get("Date")
            if c and d and c != "N/D":
                try:
                    out[d] = float(c)
                except ValueError:
                    pass
        return out
    except Exception:
        return {}


def merge_prefer(primary: Dict[str, float], fallback: Dict[str, float]) -> Dict[str, float]:
    """Fill gaps in `primary` using `fallback` (primary wins on overlap)."""
    out = dict(fallback)
    out.update(primary)
    return out
