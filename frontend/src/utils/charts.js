import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart, BarChart, PieChart } from 'echarts/charts';
import {
  GridComponent,
  TooltipComponent,
  LegendComponent
} from 'echarts/components';

use([
  CanvasRenderer,
  LineChart,
  BarChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  LegendComponent
]);

/** 面积折线：series [{ name, key, color? }]，rows [{ t, ...keys }] */
export function lineOption(rows, series) {
  return {
    tooltip: { trigger: 'axis' },
    legend: { bottom: 0, icon: 'circle', itemWidth: 8, itemHeight: 8 },
    grid: { top: 16, right: 12, bottom: 36, left: 8, containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: rows.map((r) => r.t),
      axisLine: { show: false },
      axisTick: { show: false }
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      splitLine: { lineStyle: { type: 'dashed' } }
    },
    series: series.map((s) => ({
      name: s.name,
      type: 'line',
      smooth: true,
      showSymbol: false,
      areaStyle: { opacity: 0.08 },
      lineStyle: { width: 2 },
      emphasis: { focus: 'series' },
      data: rows.map((r) => Number(r[s.key] || 0))
    }))
  };
}

/** 横向排行柱：items [{ k, v }] */
export function rankBarOption(items, label = (k) => k) {
  const list = [...items].reverse();
  return {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { top: 4, right: 24, bottom: 4, left: 4, containLabel: true },
    xAxis: {
      type: 'value',
      minInterval: 1,
      splitLine: { lineStyle: { type: 'dashed' } }
    },
    yAxis: {
      type: 'category',
      data: list.map((i) => label(i.k)),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { width: 110, overflow: 'truncate' }
    },
    series: [
      {
        type: 'bar',
        barMaxWidth: 16,
        itemStyle: { borderRadius: [0, 6, 6, 0] },
        label: { show: true, position: 'right', fontSize: 11 },
        data: list.map((i) => Number(i.v || 0))
      }
    ]
  };
}

/** 环形占比：items [{ k, v }] */
export function donutOption(items, label = (k) => k) {
  return {
    tooltip: { trigger: 'item' },
    legend: {
      orient: 'vertical',
      right: 0,
      top: 'middle',
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      textStyle: { width: 90, overflow: 'truncate' }
    },
    series: [
      {
        type: 'pie',
        radius: ['52%', '78%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: true,
        itemStyle: { borderRadius: 4, borderWidth: 2 },
        label: { show: false },
        data: items.map((i) => ({ name: label(i.k), value: Number(i.v || 0) }))
      }
    ]
  };
}

export const pct = (a, b) => (b > 0 ? Math.round((a / b) * 1000) / 10 : 0);
export const fmtNum = (v) => (v == null ? '-' : Number(v).toLocaleString());
export const fmtAmount = (v) =>
  v == null
    ? '-'
    : Number(v).toLocaleString(undefined, {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2
      });
