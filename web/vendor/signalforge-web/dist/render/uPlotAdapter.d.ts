import { default as uPlot } from 'uplot';
import { GraphTile, HeroGraphModel, TileSeries } from '../types';
export type TimeRange = {
    start: number;
    end: number;
};
export type UPlotBuild = {
    data: uPlot.AlignedData;
    series: uPlot.Series[];
    scales: Record<string, uPlot.Scale>;
    axes: uPlot.Axis[];
};
export declare function uplotData(tile: GraphTile, currentTimeMs?: number, viewportWidth?: number): UPlotBuild;
export declare function seriesDrawOrder(a: TileSeries, b: TileSeries): number;
export declare function lineWidthFor(role: string): 1.55 | 0.9 | 0.75 | 1.05 | 1.1 | 0.95 | 0.85;
export declare function sharedTimeGrid(tile: GraphTile, tileSeries: TileSeries[]): number[];
export declare function buildScales(scaleKeys: Set<string>): Record<string, uPlot.Scale>;
export declare function buildAxes(scaleKeys: Set<string>, tile: GraphTile): uPlot.Axis[];
export declare function paddedRange(minPad: number, clamp?: [number, number]): uPlot.Range.Function;
export declare function logScale(scale: string): scale is "pressure_log" | "pressure_rate_log";
export declare function logSplits(min: number, max: number): number[];
export declare function ySplits(min: number, max: number): number[];
export declare function axisLabel(scale: string, _tile: GraphTile): string;
export declare function scaleForSeries(_tile: GraphTile, series: TileSeries): string;
export declare function stateBlocks(series: TileSeries, start: number, span: number): {
    key: string;
    left: number;
    width: number;
    value: number;
    label: string;
}[];
export declare function inTimeRange(timestamp: string, range: TimeRange): boolean;
export declare function renderKindFor(kind: string): "line" | "counter" | "swimlane" | "event_rail";
export declare function drawTileOverlays(plot: uPlot, tile: GraphTile, heroGraph?: HeroGraphModel, currentTimeMs?: number, hoverTimeMs?: number, timeRange?: TimeRange): void;
//# sourceMappingURL=uPlotAdapter.d.ts.map