import { WallsHandle } from './useWalls';
export type WallManagerProps = {
    walls: WallsHandle;
    selectedWallId?: string;
    onSelect: (wallId: string) => void;
};
export declare function WallManager({ walls, selectedWallId, onSelect }: WallManagerProps): import("react/jsx-runtime").JSX.Element;
//# sourceMappingURL=WallManager.d.ts.map