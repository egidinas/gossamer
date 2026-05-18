import { GraphTileCardRef, GraphWallCard, GraphWallModel, TileSeries } from '../types';
export declare const roleColors: Record<string, string>;
export declare const signalColors: Record<string, string>;
export declare const distinctivePalette: string[];
export declare function palette(index: number): string;
export declare function paletteForID(id: string, fallbackIndex: number): string;
export declare function colorForSignal(signal: Pick<TileSeries, "id" | "role" | "render_kind" | "kind" | "color"> | {
    id: string;
    role: string;
    kind?: string;
    color?: string;
}, index?: number): string;
export declare function semanticColor(id: string): "#ff6b35" | "#ffd400" | "#f8fafc" | "#00c8ff" | "#ff8a00" | "#b65cff" | "#ff315f" | "#00d6a3" | "#1f6fff" | "#ff7a35" | "#b079ff" | undefined;
export declare function signalPriority(signal: {
    id: string;
    label?: string;
    role?: string;
    kind?: string;
    render_kind?: string;
}): 1 | 10 | 2 | 3 | 0 | 5 | 6 | 4 | 7 | 8 | 9 | 20;
export declare function orderLegendSignals<T extends {
    id: string;
    label?: string;
    role?: string;
    kind?: string;
    render_kind?: string;
}>(signals: T[]): T[];
export declare function graphCardPriority(a: GraphWallCard, b: GraphWallCard): number;
export declare function graphSectionPriority(a: GraphWallModel["sections"][number], b: GraphWallModel["sections"][number]): number;
export declare function graphSectionRank(section: GraphWallModel["sections"][number]): number;
export declare function graphCardRank(card: GraphWallCard): 10 | 0 | 60 | 20 | 30 | 40 | 50 | 80 | 70;
export declare function cardPriority(card: GraphTileCardRef): number;
export declare function tileCardPriority(a: GraphTileCardRef, b: GraphTileCardRef): number;
export declare function eventColor(kind?: string): "#ff315f" | "#00d6a3" | "#1f6fff" | "#31d6ff" | "#b079ff" | "#ffb000";
export declare function blockLabel(label: string, value: number): string;
//# sourceMappingURL=visualPolicy.d.ts.map