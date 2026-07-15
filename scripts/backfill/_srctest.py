import requests

s = requests.Session()
s.headers.update({
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
    "Accept": "text/html,application/json,*/*",
})

# --- Stooq from the server (gold + a couple FX + a BIST-like) ---
for sym in ["xauusd", "usdtry", "eurtry"]:
    try:
        r = s.get("https://stooq.com/q/d/l/", params={"s": sym, "i": "d", "d1": "20000101", "d2": "20260713"}, timeout=25)
        head = r.text[:50].replace("\n", " ")
        rows = r.text.count("\n")
        print(f"STOOQ {sym} HTTP {r.status_code} rows~{rows} | {head}")
    except Exception as e:
        print("STOOQ", sym, "ERR", repr(e)[:80])

# --- Is Yatirim BIST with fuller browser headers ---
iy = requests.Session()
iy.headers.update({
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36",
    "Accept": "application/json, text/javascript, */*; q=0.01",
    "X-Requested-With": "XMLHttpRequest",
    "Referer": "https://www.isyatirim.com.tr/tr-tr/analiz/hisse/Sayfalar/default.aspx?hisse=THYAO",
})
url = "https://www.isyatirim.com.tr/_Layouts/15/IsYatirim.Website/Common/Data.aspx/HisseTekGetir"
try:
    r = iy.get(url, params={"hisse": "THYAO", "startdate": "01-01-2015", "enddate": "13-07-2026"}, timeout=25)
    print("ISYATIRIM HTTP", r.status_code, "len", len(r.text), "|", r.text[:80].replace("\n", " "))
    if r.status_code == 200:
        j = r.json(); rows = j.get("value") or []
        print("  rows", len(rows), "keys", (list(rows[0].keys()) if rows else None))
        if rows:
            print("  sample", {k: rows[0][k] for k in list(rows[0])[:6]})
except Exception as e:
    print("ISYATIRIM ERR", repr(e)[:100])
