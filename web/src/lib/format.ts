// Turkish-locale number/price formatting. All instrument prices are TRY.
const tr = new Intl.NumberFormat("tr-TR", {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});
const tr4 = new Intl.NumberFormat("tr-TR", {
  minimumFractionDigits: 4,
  maximumFractionDigits: 4,
});

export function price(v: number): string {
  if (!isFinite(v)) return "—";
  return v >= 1 ? tr.format(v) : tr4.format(v);
}

export function tl(v: number): string {
  return "₺" + price(v);
}

export function signedPct(v: number): string {
  const s = v.toFixed(2);
  return (v >= 0 ? "+" : "") + s + "%";
}

export function signed(v: number): string {
  return (v >= 0 ? "+" : "") + price(v);
}

export function compact(v: number): string {
  return new Intl.NumberFormat("tr-TR", { notation: "compact", maximumFractionDigits: 1 }).format(v);
}

// Compact USD (e.g. $1,2 Mr). Used for crypto market cap / volume.
export function usdCompact(v: number): string {
  return (
    "$" + new Intl.NumberFormat("tr-TR", { notation: "compact", maximumFractionDigits: 2 }).format(v)
  );
}

export function usd(v: number): string {
  if (!isFinite(v)) return "—";
  const f = new Intl.NumberFormat("en-US", {
    minimumFractionDigits: v >= 1 ? 2 : 4,
    maximumFractionDigits: v >= 1 ? 2 : 4,
  });
  return "$" + f.format(v);
}

// Trims an ISO date to yyyy-mm-dd for compact display.
export function dateShort(iso: string): string {
  return iso ? iso.slice(0, 10) : "—";
}

export function timeAgo(iso: string): string {
  const d = new Date(iso).getTime();
  const s = Math.max(0, (Date.now() - d) / 1000);
  if (s < 60) return `${Math.floor(s)} sn önce`;
  if (s < 3600) return `${Math.floor(s / 60)} dk önce`;
  if (s < 86400) return `${Math.floor(s / 3600)} sa önce`;
  return `${Math.floor(s / 86400)} gün önce`;
}
