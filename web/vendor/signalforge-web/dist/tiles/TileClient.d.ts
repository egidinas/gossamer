import { GraphTile, TileAdapter } from '../types';
export type TileLevel = "live" | "minute" | "hour";
export declare function pickTileLevel(timeWindowMs: number): TileLevel;
export declare class TileClient {
    private adapter;
    private cache;
    private inflight;
    private ttlMs;
    constructor(adapter: TileAdapter, opts?: {
        ttlMs?: number;
    });
    private cacheKey;
    fetch(wallId: string, cardId: string, level: TileLevel): Promise<GraphTile>;
    fetchForViewport(wallId: string, cardId: string, timeWindowMs: number): Promise<GraphTile>;
    invalidate(wallId?: string): void;
}
//# sourceMappingURL=TileClient.d.ts.map