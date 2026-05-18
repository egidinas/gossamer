import { GraphTile, HeroGraphModel } from '../types';
export type UPlotTileRendererProps = {
    tile: GraphTile;
    heroGraph?: HeroGraphModel;
    height?: number;
    currentTimeMs?: number;
    hoverTimeMs?: number;
    className?: string;
    dataGraphRenderer?: string;
    syncKey?: string;
};
export declare function UPlotTileRenderer({ tile, heroGraph, height, currentTimeMs, hoverTimeMs, className, dataGraphRenderer, syncKey, }: UPlotTileRendererProps): import("react/jsx-runtime").JSX.Element;
//# sourceMappingURL=UPlotTileRenderer.d.ts.map