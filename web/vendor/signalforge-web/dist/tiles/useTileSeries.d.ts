import { GraphTile, TileAdapter } from '../types';
export type TileState = {
    status: "loading";
    tile: null;
} | {
    status: "ok";
    tile: GraphTile;
} | {
    status: "error";
    tile: null;
    error: string;
};
export declare function useTileSeries(adapter: TileAdapter, wallId: string, cardId: string, timeWindowMs: number, pollIntervalMs?: number): TileState;
//# sourceMappingURL=useTileSeries.d.ts.map