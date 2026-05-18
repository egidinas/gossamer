import { WallConfig } from '../types';
export declare function loadWalls(namespace: string): WallConfig[];
export declare function saveWalls(list: WallConfig[], namespace: string): void;
export type WallsHandle = {
    walls: WallConfig[];
    add(label: string): WallConfig;
    rename(wallId: string, label: string): void;
    remove(wallId: string): void;
    wallForDevice(deviceId: string): WallConfig;
};
export declare function useWalls(namespace: string): WallsHandle;
//# sourceMappingURL=useWalls.d.ts.map