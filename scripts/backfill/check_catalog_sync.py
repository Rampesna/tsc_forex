#!/usr/bin/env python3
"""Fail CI when the Python backfill catalog drifts from the Go catalog."""

import re
from pathlib import Path

import catalog as C

ROOT = Path(__file__).resolve().parents[2]
GO = (ROOT / "internal/domain/instrument.go").read_text(encoding="utf-8")


def block(name: str) -> str:
    match = re.search(
        rf"(?:var {name} = \[\][^{{]+\{{|func {name}\(\) \[\]Instrument \{{\s*return \[\]Instrument\{{)(.*?)(?:\n\}}|\n\t\}})",
        GO,
        flags=re.S,
    )
    if not match:
        raise SystemExit(f"cannot parse Go catalog block: {name}")
    return match.group(1)


def tuple_fields(name: str):
    return re.findall(r'\{"([^"]+)",\s*"[^"]*",\s*"([^"]+)"', block(name))


def first_fields(name: str):
    return re.findall(r'\{"([^"]+)"', block(name))


def assert_equal(label, python_value, go_value):
    if python_value != go_value:
        missing = sorted(set(go_value) - set(python_value))
        extra = sorted(set(python_value) - set(go_value))
        raise SystemExit(
            f"{label} catalog drift: missing_in_python={missing} extra_in_python={extra}"
        )


def main():
    assert_equal("currencies", C.CURRENCIES, first_fields("Currencies"))
    assert_equal("cryptos", [code for code, _, _ in C.CRYPTOS], first_fields("Cryptos"))
    assert_equal("BIST stocks", C.STOCKS, first_fields("Stocks"))
    assert_equal(
        "global stocks",
        [code for code, _ in C.GLOBAL_STOCKS],
        first_fields("GlobalStocks"),
    )

    crypto_go = dict(tuple_fields("Cryptos"))
    crypto_python = {code: yahoo for code, yahoo, _ in C.CRYPTOS}
    assert_equal(
        "crypto Yahoo mappings",
        sorted(crypto_python.items()),
        sorted(crypto_go.items()),
    )
    stock_go = dict(tuple_fields("Stocks"))
    stock_python = {code: f"{code}.IS" for code in C.STOCKS}
    assert_equal(
        "BIST Yahoo mappings",
        sorted(stock_python.items()),
        sorted(stock_go.items()),
    )
    global_go = dict(tuple_fields("GlobalStocks"))
    assert_equal(
        "global Yahoo mappings",
        sorted(C.GLOBAL_STOCKS),
        sorted(global_go.items()),
    )

    gold_go = re.findall(r'Code:\s*"([^"]+)"', block("goldProducts"))
    assert_equal("gold products", [code for code, _, _ in C.GOLD_PRODUCTS], gold_go)

    total = (
        len(C.CURRENCIES)
        + len(C.GOLD_PRODUCTS)
        + len(C.CRYPTOS)
        + len(C.STOCKS)
        + len(C.GLOBAL_STOCKS)
    )
    if total != 410:
        raise SystemExit(f"canonical catalog count changed: expected 410, got {total}")
    print("catalog sync OK: 410 instruments")


if __name__ == "__main__":
    main()
