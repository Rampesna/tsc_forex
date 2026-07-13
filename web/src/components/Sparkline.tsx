// Lightweight inline SVG sparkline (line only, no gradient fill) — matches the
// clean look used across the mobile apps.
export function Sparkline({
  data,
  width = 88,
  height = 30,
  color,
}: {
  data: number[];
  width?: number;
  height?: number;
  color?: string;
}) {
  if (!data || data.length < 2) {
    return <div style={{ width, height }} />;
  }
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const stroke = color ?? (data[data.length - 1] >= data[0] ? "var(--up)" : "var(--down)");
  const step = width / (data.length - 1);
  const pts = data
    .map((v, i) => {
      const x = i * step;
      const y = height - ((v - min) / range) * (height - 4) - 2;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
  return (
    <svg width={width} height={height} style={{ display: "block" }}>
      <polyline
        points={pts}
        fill="none"
        stroke={stroke}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
