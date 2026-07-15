import { useEffect, useRef, useState } from "react";
import {
  AreaSeries,
  CandlestickSeries,
  LineSeries,
  createChart,
  type IChartApi,
  type ISeriesApi,
  type UTCTimestamp,
} from "lightweight-charts";
import type { Candle } from "../lib/types";

export type ChartType = "candles" | "line" | "area";

const UP = "#4ec99a";
const DOWN = "#f16b7d";
const ACCENT = "#45c0d6";

function fmt(v: number): string {
  return new Intl.NumberFormat("tr-TR", {
    minimumFractionDigits: v >= 1 ? 2 : 4,
    maximumFractionDigits: v >= 1 ? 2 : 4,
  }).format(v);
}

type Bar = { time: UTCTimestamp; open: number; high: number; low: number; close: number };

// Interactive price chart (pan / zoom / crosshair) with a live OHLC legend that
// tracks the crosshair. The last bar updates in real time from the live WS price
// via series.update() — the chart is NOT rebuilt on every tick (Binance-style).
export function PriceChart({
  candles,
  type = "candles",
  height = 340,
  livePrice,
}: {
  candles: Candle[];
  type?: ChartType;
  height?: number;
  livePrice?: number;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const seriesRef = useRef<ISeriesApi<"Candlestick" | "Line" | "Area"> | null>(null);
  const lastBarRef = useRef<Bar | null>(null);
  const [legend, setLegend] = useState<{ o: number; h: number; l: number; c: number; up: boolean } | null>(null);

  // Create the chart + series once per type/height. Data is applied separately.
  useEffect(() => {
    if (!ref.current) return;
    const chart: IChartApi = createChart(ref.current, {
      height,
      layout: { background: { color: "transparent" }, textColor: "#9ca3af", fontFamily: "inherit", fontSize: 11 },
      grid: {
        vertLines: { color: "rgba(255,255,255,0.035)" },
        horzLines: { color: "rgba(255,255,255,0.035)" },
      },
      rightPriceScale: { borderColor: "rgba(255,255,255,0.06)" },
      timeScale: { borderColor: "rgba(255,255,255,0.06)", timeVisible: true, secondsVisible: false },
      crosshair: {
        mode: 1,
        vertLine: { color: "rgba(255,255,255,0.2)", width: 1, style: 2, labelBackgroundColor: "#1a1b1e" },
        horzLine: { color: "rgba(255,255,255,0.2)", width: 1, style: 2, labelBackgroundColor: "#1a1b1e" },
      },
      autoSize: true,
    });
    chartRef.current = chart;

    if (type === "candles") {
      seriesRef.current = chart.addSeries(CandlestickSeries, {
        upColor: UP, downColor: DOWN, borderVisible: false, wickUpColor: UP, wickDownColor: DOWN,
      });
    } else if (type === "line") {
      seriesRef.current = chart.addSeries(LineSeries, { color: ACCENT, lineWidth: 2 });
    } else {
      seriesRef.current = chart.addSeries(AreaSeries, {
        lineColor: ACCENT, topColor: "rgba(69,192,214,0.28)", bottomColor: "rgba(69,192,214,0.02)", lineWidth: 2,
      });
    }

    chart.subscribeCrosshairMove((param) => {
      const d = param.seriesData.get(seriesRef.current!) as
        | { open?: number; high?: number; low?: number; close?: number; value?: number }
        | undefined;
      const last = lastBarRef.current;
      if (!d) {
        if (last) setLegend({ o: last.open, h: last.high, l: last.low, c: last.close, up: last.close >= last.open });
        return;
      }
      const o = d.open ?? d.value ?? 0;
      const c = d.close ?? d.value ?? 0;
      setLegend({ o, h: d.high ?? c, l: d.low ?? c, c, up: c >= o });
    });

    return () => {
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
    };
  }, [type, height]);

  // Apply candle data whenever the fetched series changes (range switch/refetch).
  useEffect(() => {
    const series = seriesRef.current;
    if (!series) return;
    const seen = new Set<number>();
    const rows: Bar[] = candles
      .map((c) => ({
        time: Math.floor(new Date(c.bucket).getTime() / 1000) as UTCTimestamp,
        open: c.open,
        high: c.high,
        low: c.low,
        close: c.close,
      }))
      .filter((c) => Number.isFinite(c.time))
      .sort((a, b) => (a.time as number) - (b.time as number))
      .filter((c) => (seen.has(c.time as number) ? false : (seen.add(c.time as number), true)));

    if (type === "candles") {
      (series as ISeriesApi<"Candlestick">).setData(rows);
    } else {
      (series as ISeriesApi<"Line" | "Area">).setData(rows.map((r) => ({ time: r.time, value: r.close })));
    }
    const last = rows[rows.length - 1] ?? null;
    lastBarRef.current = last;
    if (last) setLegend({ o: last.open, h: last.high, l: last.low, c: last.close, up: last.close >= last.open });
    chartRef.current?.timeScale().fitContent();
  }, [candles, type]);

  // Live update: fold the incoming WS price into the last bar without a refetch.
  useEffect(() => {
    const series = seriesRef.current;
    const last = lastBarRef.current;
    if (!series || !last || livePrice == null || !Number.isFinite(livePrice) || livePrice <= 0) return;
    const bar: Bar = {
      time: last.time,
      open: last.open,
      high: Math.max(last.high, livePrice),
      low: Math.min(last.low, livePrice),
      close: livePrice,
    };
    lastBarRef.current = bar;
    if (type === "candles") {
      (series as ISeriesApi<"Candlestick">).update(bar);
    } else {
      (series as ISeriesApi<"Line" | "Area">).update({ time: bar.time, value: livePrice });
    }
    setLegend({ o: bar.open, h: bar.high, l: bar.low, c: bar.close, up: bar.close >= bar.open });
  }, [livePrice, type]);

  return (
    <div style={{ position: "relative" }}>
      {legend && (
        <div
          style={{
            position: "absolute",
            top: 8,
            left: 8,
            zIndex: 3,
            display: "flex",
            gap: 12,
            fontSize: 11.5,
            fontFamily: "var(--mono)",
            background: "rgba(18,19,21,0.7)",
            border: "1px solid var(--stroke)",
            borderRadius: 8,
            padding: "5px 10px",
            pointerEvents: "none",
          }}
        >
          <LegItem k="A" v={legend.o} />
          <LegItem k="Y" v={legend.h} />
          <LegItem k="D" v={legend.l} />
          <LegItem k="K" v={legend.c} color={legend.up ? UP : DOWN} />
        </div>
      )}
      <div ref={ref} style={{ width: "100%", height }} />
    </div>
  );
}

function LegItem({ k, v, color }: { k: string; v: number; color?: string }) {
  return (
    <span style={{ color: "var(--text-faint)" }}>
      {k} <span style={{ color: color ?? "var(--text)" }}>{fmt(v)}</span>
    </span>
  );
}
