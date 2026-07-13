import { useEffect, useRef } from "react";
import {
  CandlestickSeries,
  createChart,
  type IChartApi,
  type UTCTimestamp,
} from "lightweight-charts";
import type { Candle } from "../lib/types";

export function PriceChart({ candles, height = 320 }: { candles: Candle[]; height?: number }) {
  const ref = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);

  useEffect(() => {
    if (!ref.current) return;
    const chart = createChart(ref.current, {
      height,
      layout: {
        background: { color: "transparent" },
        textColor: "#8a94a6",
        fontFamily: "inherit",
      },
      grid: {
        vertLines: { color: "rgba(255,255,255,0.04)" },
        horzLines: { color: "rgba(255,255,255,0.04)" },
      },
      rightPriceScale: { borderColor: "rgba(255,255,255,0.06)" },
      timeScale: { borderColor: "rgba(255,255,255,0.06)", timeVisible: true, secondsVisible: false },
      crosshair: { mode: 0 },
      autoSize: true,
    });
    const series = chart.addSeries(CandlestickSeries, {
      upColor: "#0ecb81",
      downColor: "#f6465d",
      borderVisible: false,
      wickUpColor: "#0ecb81",
      wickDownColor: "#f6465d",
    });
    // De-dupe + sort by time (lightweight-charts requires strictly ascending).
    const seen = new Set<number>();
    const data = candles
      .map((c) => ({
        time: Math.floor(new Date(c.bucket).getTime() / 1000) as UTCTimestamp,
        open: c.open,
        high: c.high,
        low: c.low,
        close: c.close,
      }))
      .filter((c) => Number.isFinite(c.time))
      .sort((a, b) => a.time - b.time)
      .filter((c) => (seen.has(c.time) ? false : (seen.add(c.time), true)));
    series.setData(data);
    chart.timeScale().fitContent();
    chartRef.current = chart;
    return () => {
      chart.remove();
      chartRef.current = null;
    };
  }, [candles, height]);

  return <div ref={ref} style={{ width: "100%", height }} />;
}
