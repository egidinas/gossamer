import { Assignment, AssignmentsStoreOptions } from '../types';
export declare function loadAssignments(opts: AssignmentsStoreOptions): Assignment[];
export declare function saveAssignments(list: Assignment[], opts: AssignmentsStoreOptions): void;
export declare function makeAssignment(wallId: string, paramId: number, deviceId: string, instance?: number): Assignment;
export type AssignmentsHandle = {
    list: Assignment[];
    add(wallId: string, paramId: number, deviceId: string, instance?: number): void;
    remove(wallId: string, paramId: number, deviceId: string, instance?: number): void;
    forWall(wallId: string): Assignment[];
    hasAssignment(wallId: string, paramId: number, deviceId: string, instance?: number): boolean;
};
export declare function useAssignments(opts: AssignmentsStoreOptions): AssignmentsHandle;
//# sourceMappingURL=useAssignments.d.ts.map