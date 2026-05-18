import { GraphTile } from '../types';
export declare const CANONICAL_TILE_RENDERER = "signalforge.tile.uplot";
export type SeriesRoleMeta = {
    readonly label: string;
    readonly rank: number;
    readonly className: string;
    readonly dash: string;
    readonly width: number;
    readonly opacity: number;
};
export declare const SERIES_ROLE_META: Readonly<Record<string, SeriesRoleMeta>>;
export type RenderedTileSeries = {
    key: string;
    tileId?: string;
    targetId?: unknown;
    label?: unknown;
    fullLabel?: unknown;
    role: string;
    seriesRole: string;
    roleRank: number;
    color?: string;
    unit: string;
    provenance?: unknown;
    source: unknown;
    paramId?: unknown;
    deviceId?: unknown;
    instance?: unknown;
    signalId?: unknown;
    history: {
        ts: number[];
        v: number[];
        q: string[];
    };
};
type LooseRecord = Record<string, unknown>;
export declare function seriesRoleMeta(role?: string): SeriesRoleMeta;
export declare function seriesRoleColor(role?: string, fallback?: string): string;
export declare function measuredElementWidth(el: Element | null | undefined): number;
export declare function emptyGraphTile(opts?: LooseRecord): GraphTile;
export declare function renderSeriesFromGraphTile(tile: unknown): RenderedTileSeries[];
export declare function normalizeGraphTile(tile: unknown, opts?: LooseRecord): GraphTile;
export {};
//# sourceMappingURL=tileModel.d.ts.map