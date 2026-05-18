import { GraphMarker, GraphTile, TileSeries } from '../types';
export type TimeRange = {
    start: number;
    end: number;
};
export type MarkerLabelRect = {
    x: number;
    y: number;
    width: number;
    height: number;
};
export declare function markerColor(marker: {
    role?: string;
    result?: string;
    kind?: string;
}): "rgba(255,112,67,0.98)" | "rgba(36,214,255,0.98)" | "rgba(146,255,111,0.98)" | "rgba(255,49,95,0.96)" | "rgba(176,121,255,0.96)" | "rgba(255,176,0,0.98)" | "rgba(0,214,163,0.96)" | "rgba(49,214,255,0.95)";
export declare function operatorMarkerLines(marker: GraphMarker, compact?: boolean): string[];
export declare function formatMarkerDateTime(value: string, compact?: boolean): string;
export declare function placeMarkerLabel({ x, y, labelWidth, labelHeight, left, top, width, height, placed, markerRadius }: {
    x: number;
    y: number;
    labelWidth: number;
    labelHeight: number;
    left: number;
    top: number;
    width: number;
    height: number;
    placed: MarkerLabelRect[];
    markerRadius: number;
}): MarkerLabelRect | null;
export declare function rectanglesOverlap(a: MarkerLabelRect, b: MarkerLabelRect): boolean;
export declare function fitCanvasText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string;
export declare function shortGateLabel(label: string): string;
export declare function legendReadouts(tile: GraphTile, visibleSignals: Array<{
    id: string;
    label: string;
}>, timeMs?: number, currentTimeMs?: number): Map<string, string>;
export declare function displayableLegendValue(series: TileSeries, value: number): boolean;
export declare function clampTime(timeMs: number, domain: number[]): number;
export declare function rawValueAt(series: TileSeries, timeMs: number, tile?: GraphTile): number | undefined;
export declare function stateAt(series: TileSeries, timeMs: number): string | undefined;
export declare function stateLabel(series: Pick<TileSeries, "value_table">, value?: number | string, state?: string, label?: string): string | undefined;
export declare function formatLegendValue(series: TileSeries, value: number): string;
export declare function formatScientific(value: number): string;
export declare function formatPressure(value: number): string;
export declare function unitForAxis(axisID?: string): "" | "bar" | "degC" | "mbar" | "W" | "ms" | "mbar/min" | "%" | "V" | "A" | "dB" | "Hz" | "Ω";
//# sourceMappingURL=markers.d.ts.map