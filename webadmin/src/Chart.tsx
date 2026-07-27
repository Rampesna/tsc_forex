import { useEffect, useRef } from "react";
import { init, use } from "echarts/core";
import { BarChart, LineChart, PieChart } from "echarts/charts";
import { GridComponent, LegendComponent, TooltipComponent } from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import type { EChartsOption } from "echarts";

use([BarChart, LineChart, PieChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer]);

export function Chart({
  option,
  height = 280,
  className = "",
}: {
  option: EChartsOption;
  height?: number;
  className?: string;
}) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref.current) return;
    const chart = init(ref.current, undefined, { renderer: "canvas" });
    chart.setOption(option, { notMerge: true });
    const observer = new ResizeObserver(() => chart.resize());
    observer.observe(ref.current);
    return () => {
      observer.disconnect();
      chart.dispose();
    };
  }, [option]);

  return <div ref={ref} className={className} style={{ width: "100%", height }} />;
}

export const chartColors = ["#39c6d8", "#6e67f5", "#f5a524", "#38c985", "#ef5b75", "#8f97aa"];

export const axisStyle = {
  axisLine: { lineStyle: { color: "#262a31" } },
  axisTick: { show: false },
  axisLabel: { color: "#737b8d", fontSize: 10 },
  splitLine: { lineStyle: { color: "#1e2229", type: "dashed" as const } },
};
