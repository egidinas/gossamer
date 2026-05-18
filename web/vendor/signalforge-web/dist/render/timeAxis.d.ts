export type TimeRange = {
    start: number;
    end: number;
};
export declare function clampRange(range: TimeRange, fullRange: TimeRange, minSpan: number): TimeRange;
export declare function timeTicks(startISO: string, endISO: string, count: number): {
    iso: string;
    ratio: number;
    label: string;
}[];
export declare function chooseTickStep(spanMs: number, targetCount: number): number;
export declare function tickLabel(date: Date, stepMs: number): string;
export declare function TimeAxisTrack({ ticks, start, end, nowRatio, hoverTimeMs, peekTimeMs, compact }: {
    ticks: ReturnType<typeof timeTicks>;
    start: number;
    end: number;
    nowRatio?: number;
    hoverTimeMs?: number;
    peekTimeMs?: number;
    compact?: boolean;
}): import("react/jsx-runtime").JSX.Element;
export declare function HeroTopTimeAxis({ timeRange, currentTimeMs, hoverTimeMs, readoutTimeMs, tickCount }: {
    timeRange: TimeRange;
    currentTimeMs?: number;
    hoverTimeMs?: number;
    readoutTimeMs?: number;
    tickCount?: number;
}): import("react/jsx-runtime").JSX.Element;
export declare function SharedTimeAxis({ fullRange, timeRange, currentTimeMs, hoverTimeMs, peekTimeMs, plotBounds, onTimeRange, tickCount, }: {
    fullRange: TimeRange;
    timeRange: TimeRange;
    currentTimeMs?: number;
    hoverTimeMs?: number;
    peekTimeMs?: number;
    plotBounds?: {
        left: number;
        right: number;
    };
    onTimeRange: (range: TimeRange) => void;
    tickCount: number;
}): import("react/jsx-runtime").JSX.Element;
//# sourceMappingURL=timeAxis.d.ts.map