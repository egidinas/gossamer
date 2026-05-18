import { GraphTile, TileSeries } from '../types';
export type PreparedSeriesPoint = {
    t: number;
    v: number;
};
export declare function viewportSeries(tile: GraphTile, series: TileSeries, viewportWidth: number): TileSeries;
export declare function lttb(points: TileSeries["points"], threshold: number, yValue: (value: number) => number): TileSeries["points"];
export declare function lttbPreservingGaps(points: TileSeries["points"], threshold: number, yValue: (value: number) => number): TileSeries["points"];
export declare function decimationValue(_tile: GraphTile, series: TileSeries, value: number): number;
export declare function resampleSeries(tile: GraphTile, series: TileSeries, xValues: number[], currentTimeMs?: number): Array<number | null>;
export declare function prepareSeriesPoints(series: TileSeries): PreparedSeriesPoint[];
export declare function resamplePreparedSeries(tile: GraphTile, series: TileSeries, points: PreparedSeriesPoint[], xValues: number[], currentTimeMs?: number): Array<number | null>;
export declare function commandCenterGapBreaks(tile: GraphTile, series: TileSeries): number[];
export declare function commandCenterGapBreaksFromPoints(tile: GraphTile, series: TileSeries, points: PreparedSeriesPoint[]): number[];
export declare function commandCenterTraceGapMs(tile: GraphTile, series: TileSeries): number;
export declare function commandCenterProjectedSeries(tile: GraphTile, series: TileSeries): boolean;
export declare function displayValue(_tile: GraphTile, series: TileSeries, value: number): number;
export declare function isDiscreteSeries(series: TileSeries): boolean;
export declare function interpolationValue(series: TileSeries, value: number): number;
export declare function valueFromInterpolation(series: TileSeries, value: number): number;
export declare function isPressureAxis(axisID?: string): axisID is "pressure_log" | "pressure_rate_log" | "pressure_mbar" | "pressure_rate";
export declare function isPressureLogAxis(axisID?: string): axisID is "pressure_log" | "pressure_rate_log";
//# sourceMappingURL=decimation.d.ts.map