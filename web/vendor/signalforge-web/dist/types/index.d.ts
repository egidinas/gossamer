export type GraphPoint = {
    timestamp: string;
    value: number;
};
export type GraphTimeAxis = {
    start: string;
    end: string;
    anchor: string;
    now?: string;
    range_seconds: number;
    clamp: boolean;
    latest_policy: string;
    default_window_start?: string;
    default_window_end?: string;
};
export type GraphYAxis = {
    id: string;
    label: string;
    units: string;
    scale: string;
    min: number;
    max: number;
    side: string;
    format: string;
};
export type GraphTrace = {
    id: string;
    label: string;
    role: string;
    units: string;
    axis_id: string;
    source: string;
    values: GraphPoint[];
};
export type CompanionGraphGroup = {
    id: string;
    label: string;
    axes: GraphYAxis[];
    traces: GraphTrace[];
};
export type HeroGraphModel = {
    id: string;
    title: string;
    owner: string;
    provenance: string;
    time_axis: GraphTimeAxis;
    axes: GraphYAxis[];
    traces: GraphTrace[];
    phase_bands: GraphBand[];
    dwell_windows: GraphBand[];
    markers: GraphMarker[];
    companion_groups: CompanionGraphGroup[];
    execution?: {
        now?: string;
    };
};
export type GraphWallSignal = {
    id: string;
    label: string;
    unit?: string;
    source: string;
    source_family: string;
    kind: string;
    category: string;
    role: string;
    subsystem: string;
    axis_id?: string;
    section_id: string;
    value_table?: Record<string, string>;
};
export type GraphCardPlacement = {
    section_id: string;
    group_id: string;
    order: number;
    height_weight: number;
    default_visible: boolean;
    pinned: boolean;
    colocated_with?: string;
    resize_policy: string;
};
export type GraphWallCard = {
    id: string;
    title: string;
    kind: "line" | "counter" | "state" | "event" | string;
    role: string;
    placement: GraphCardPlacement;
    transport: string;
    direction: string;
    unit?: string;
    axis_policy: string;
    source_family: string;
    overview?: boolean;
    bucket?: string;
    note?: string;
    render_kind?: string;
    tile_endpoint?: string;
    latest_endpoint?: string;
    collapsible?: boolean;
    default_expanded?: boolean;
    supports_time_zoom?: boolean;
    supports_y_zoom?: boolean;
    include_markers?: boolean;
    signals: GraphWallSignal[];
};
export type GraphWallTimeRange = {
    start: string;
    end: string;
    anchor: string;
    range_seconds: number;
    mode: string;
    source: string;
};
export type GraphTilePolicy = {
    default_points: number;
    max_points: number;
    live_tile_min_refresh_ms: number;
    history_tile_max_count: number;
    viewport_prefetch_px: number;
    tile_buffer_max_entries: number;
    tile_buffer_ttl_ms: number;
    resolution_levels: string[];
    subscriber_role: string;
    shared_timebase_required: boolean;
    legend_may_affect_plot_width: boolean;
    malformed_svg_path_hard_failure: boolean;
};
export type GraphSection = {
    id: string;
    title: string;
    group_id: string;
    transport: string;
    direction: string;
    status: string;
    unplotted_count: number;
    cards: GraphWallCard[];
};
export type GraphWallModel = {
    id: string;
    title: string;
    generated_at: string;
    source_mode: string;
    graph_version: string;
    owner: string;
    provenance: string;
    time_range: GraphWallTimeRange;
    tile_policy: GraphTilePolicy;
    graph_groups: unknown[];
    sections: GraphSection[];
};
export type GraphTileCardRef = {
    card_id: string;
    section_id?: string;
    title: string;
    render_kind: string;
    unit?: string;
    axis_policy: string;
    tile_endpoint: string;
    latest_endpoint: string;
    tile_files?: unknown[];
    default_expanded: boolean;
    collapsible: boolean;
    supports_time_zoom: boolean;
    supports_y_zoom: boolean;
    include_markers?: boolean;
    signals: GraphWallSignal[];
    evidence_links?: unknown[];
};
export type TilePoint = {
    timestamp: string;
    value: number;
};
export type TileSpan = {
    start: string;
    end: string;
    value?: number;
    state?: string;
    label?: string;
    severity?: string;
};
export type TileSeries = {
    id: string;
    label: string;
    role: string;
    color?: string;
    unit?: string;
    units?: string;
    kind?: string;
    axis_id?: string;
    source: string;
    source_family?: string;
    render_kind?: string;
    step?: boolean;
    value_table?: Record<string, string>;
    points?: TilePoint[];
    spans?: TileSpan[];
};
export type TileDiagnostics = {
    source?: string;
    mode?: string;
    level?: string;
    requested_t0?: string;
    requested_t1?: string;
    raw_point_count?: number;
    point_count?: number;
    decimated?: boolean;
    decimation?: string;
    time_span_ms?: number;
    freshness_ms?: number;
    source_quality?: string;
    status?: string;
    renderer?: string;
    series_count?: number;
};
export type TileProvenance = {
    source?: string;
    source_node?: string;
    source_family?: string;
    mode?: string;
    generated_at?: string;
    synthetic?: boolean;
};
export type TileBand = {
    id: string;
    label?: string;
    kind?: string;
    start: string;
    end: string;
    series_a?: string;
    series_b?: string;
    fill?: string;
};
export type GraphBand = {
    id: string;
    label?: string;
    kind?: string;
    start: string;
    end: string;
    cycle_index?: number;
    target_deg_c?: number;
    result?: string;
};
export type GraphMarker = {
    id: string;
    label: string;
    timestamp: string;
    kind: string;
    role?: string;
    axis_id?: string;
    result?: string;
    requirement_id?: string;
    cycle_index?: number;
    value?: number;
    severity?: string;
    evidence_ref?: string;
};
export type TileEvent = {
    id: string;
    label: string;
    timestamp: string;
    kind: string;
    result?: string;
    requirement_id?: string;
    value?: number;
    severity?: string;
    evidence_ref?: string;
};
export type GraphTile = {
    schema_version: number | string;
    id: string;
    tile_id?: string;
    kind?: string;
    title?: string;
    renderer?: string;
    generated_at?: string;
    campaign_id?: string;
    card_id: string;
    level: string;
    t0: string;
    t1: string;
    time_window_ms?: number;
    axes?: unknown[];
    series: TileSeries[];
    bands: TileBand[];
    markers: GraphMarker[];
    events: TileEvent[];
    diagnostics: TileDiagnostics;
    provenance: TileProvenance;
};
export type WallConfig = {
    wall_id: string;
    label: string;
    preset?: boolean;
};
export type Assignment = {
    wall_id: string;
    tile_id: string;
    target_id: string;
    kind: string;
    param_id: number;
    device_id: string;
    instance: number;
    options: Record<string, unknown>;
};
export type SemanticSignal = {
    id: number;
    sid?: string;
    name: string;
    group: string;
    subgroup: string;
    role: "monitor" | "control";
    kind: "float" | "int" | "enum";
    unit?: string;
    type?: string;
    writable?: boolean;
    dangerous?: boolean;
    min?: number;
    max?: number;
    enum?: Record<string, string>;
    applicableModes?: string[];
    cmd?: string;
};
export type Channel = {
    device_id: string;
    instance: number;
    role: string;
    label: string;
    endpoint?: string;
};
export interface SignalCatalogueAdapter {
    list(): SemanticSignal[];
    channels(): Channel[];
    channelsForSignal(signal: SemanticSignal): Channel[];
    subscribeLive(deviceId: string, paramId: number, instance: number, cb: (snap: {
        value: number | null;
        quality: string;
    }) => void): () => void;
    formatValue(value: number | null | undefined, unit?: string, paramId?: number): string;
    write?(deviceId: string, command: unknown, leaseToken: string): Promise<void>;
    roleForParam(paramId: number): string;
    colorForRole(role: string): string;
}
export interface TileAdapter {
    fetchTile(wallId: string, cardId: string, level: "live" | "minute" | "hour"): Promise<GraphTile>;
}
export interface AssignmentsStoreOptions {
    namespace: string;
}
//# sourceMappingURL=index.d.ts.map