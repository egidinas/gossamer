package api

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egidinas/gossamer/internal/report"
	"github.com/egidinas/gossamer/internal/synthetic"
	"github.com/egidinas/signalforge/contracts"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	if err := synthetic.WritePublicFixtures(dir); err != nil {
		t.Fatal(err)
	}
	if err := report.Write(dir, "thermal_acceptance_fat"); err != nil {
		t.Fatal(err)
	}
	if err := report.Write(dir, "tvac_qualification"); err != nil {
		t.Fatal(err)
	}
	if err := report.Write(dir, "flatsat_derisking"); err != nil {
		t.Fatal(err)
	}
	if err := report.Write(dir, "integrated_system_fat"); err != nil {
		t.Fatal(err)
	}
	return New(dir)
}

func TestHealthz(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"name":"Gossamer"`) {
		t.Fatalf("body missing name: %s", rec.Body.String())
	}
}

func TestManifestHasNoStoreHeader(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/manifest", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache header = %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"name": "Gossamer"`) {
		t.Fatalf("body missing manifest name: %s", rec.Body.String())
	}
}

func TestMissingCampaignReturnsNotFound(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/nope", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestTelemetryQueryRejectsCallerSQL(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/thermal_acceptance_fat/query?q=SELECT%20*%20FROM%20telemetry", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "unsupported query") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestTelemetryQueryRejectsOutOfRangeLimit(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/thermal_acceptance_fat/query?q=preview&limit=10001", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestCommandRequiresLease(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/command-authority/mock-command", nil)
	req.Header.Set("X-Operator-ID", "test-operator")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestCommandRequiresLeaseToken(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/command-authority/request-lease", nil)
	req.Header.Set("X-Operator-ID", "test-operator")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("request lease status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var lease struct {
		LeaseToken string `json:"lease_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &lease); err != nil {
		t.Fatalf("decode lease response: %v", err)
	}
	if lease.LeaseToken == "" {
		t.Fatalf("lease response did not include token: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/command-authority/mock-command", nil)
	req.Header.Set("X-Operator-ID", "test-operator")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("command without token status = %d, want 403", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/command-authority/mock-command", nil)
	req.Header.Set("X-Operator-ID", "test-operator")
	req.Header.Set("X-Lease-Token", lease.LeaseToken)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("command with token status = %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestReleaseRequiresLeaseToken(t *testing.T) {
	server := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/command-authority/request-lease", nil)
	req.Header.Set("X-Operator-ID", "test-operator")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("request lease status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var lease struct {
		LeaseToken string `json:"lease_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &lease); err != nil {
		t.Fatalf("decode lease response: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/command-authority/release-lease", nil)
	req.Header.Set("X-Operator-ID", "test-operator")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("release without token status = %d, want 403", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/command-authority/release-lease", nil)
	req.Header.Set("X-Operator-ID", "test-operator")
	req.Header.Set("X-Lease-Token", lease.LeaseToken)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("release with token status = %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestLeaseManagerReturnsImmutableStateCopies(t *testing.T) {
	leases := NewInMemoryLeaseManager()
	state := leases.GetState()
	state.AllowedCommands[0] = "mutated"
	state.OperatorLog[0].Action = "mutated"

	fresh := leases.GetState()
	if fresh.AllowedCommands[0] == "mutated" {
		t.Fatalf("AllowedCommands alias leaked into manager state")
	}
	if fresh.OperatorLog[0].Action == "mutated" {
		t.Fatalf("OperatorLog alias leaked into manager state")
	}
}

func TestSupervisorEndpointServesOverview(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/supervisor", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"lanes"`) || !strings.Contains(rec.Body.String(), `"hero_graphs"`) {
		t.Fatalf("body missing supervisor lanes or hero graphs: %s", rec.Body.String())
	}
}

func TestBusTapEndpointServesTMAndTCEvents(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/bus-tap", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"direction": "TM"`) || !strings.Contains(body, `"direction": "TC"`) {
		t.Fatalf("body missing TM or TC events: %s", body)
	}
}

func TestTileManifestEndpointServesTileArchitecture(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/thermal_acceptance_fat/tile-manifest", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var manifest contracts.GraphTileManifest
	if err := json.Unmarshal(rec.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.CampaignID != "thermal_acceptance_fat" {
		t.Fatalf("campaign_id = %q", manifest.CampaignID)
	}
	if len(manifest.Cards) == 0 || len(manifest.Levels) == 0 {
		t.Fatalf("manifest missing cards or levels: %+v", manifest)
	}
	if !strings.Contains(rec.Body.String(), `"source_format":"legacy_csv"`) {
		t.Fatalf("manifest missing DataLens legacy CSV translation: %s", rec.Body.String())
	}
}

func TestGraphShellEndpointOmitsRawTraceValues(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/thermal_acceptance_fat/graph-shell", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var model contracts.GraphModel
	if err := json.Unmarshal(rec.Body.Bytes(), &model); err != nil {
		t.Fatalf("decode shell: %v", err)
	}
	if model.HeroGraph == nil || len(model.HeroGraph.Traces) == 0 {
		t.Fatalf("shell missing hero graph metadata")
	}
	for _, trace := range model.HeroGraph.Traces {
		if len(trace.Values) != 0 {
			t.Fatalf("shell trace %s retained %d raw values", trace.ID, len(trace.Values))
		}
	}
}

func TestLiveCampaignStreamsCursorEvents(t *testing.T) {
	server := httptest.NewServer(newTestServer(t))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/api/campaigns/thermal_acceptance_fat/live")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content type = %q", got)
	}
	scanner := bufio.NewScanner(resp.Body)
	foundEvent := false
	foundPayload := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "event: cursor" {
			foundEvent = true
		}
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"source":"simulated_live_replay"`) {
			foundPayload = true
			break
		}
	}
	if !foundEvent || !foundPayload {
		t.Fatalf("stream missing cursor event or payload")
	}
}

func TestTileEndpointUsesCachedGraphModelAfterManifestLoad(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/thermal_acceptance_fat/tile-manifest", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("manifest status = %d, body: %s", rec.Code, rec.Body.String())
	}

	modelPath := filepath.Join(server.root, "fixtures", "public", "graph_models", "thermal_acceptance_fat.json")
	hiddenPath := modelPath + ".hidden"
	if err := os.Rename(modelPath, hiddenPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Rename(hiddenPath, modelPath)
	})

	req = httptest.NewRequest(http.MethodGet, "/api/campaigns/thermal_acceptance_fat/tiles?card_id=thermal_program&level=minute", nil)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("tile status = %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestTileEndpointRespectsTimeRangeAndEvidence(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/thermal_acceptance_fat/tiles?card_id=thermal_program&level=minute&t0=2026-01-15T10:30:00Z&t1=2026-01-15T22:30:00Z", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var tile contracts.GraphTile
	if err := json.Unmarshal(rec.Body.Bytes(), &tile); err != nil {
		t.Fatalf("decode tile: %v", err)
	}
	if tile.CardID != "thermal_program" {
		t.Fatalf("card_id = %q", tile.CardID)
	}
	if tile.Diagnostics.PointCount <= 0 {
		t.Fatalf("point_count = %d, want positive", tile.Diagnostics.PointCount)
	}
	for _, series := range tile.Series {
		for _, point := range series.Points {
			if point.Timestamp < tile.T0 || point.Timestamp > tile.T1 {
				t.Fatalf("point %s outside tile range %s..%s", point.Timestamp, tile.T0, tile.T1)
			}
		}
	}
	if len(tile.Markers) == 0 {
		t.Fatalf("tile missing evidence/functional markers")
	}
}

func TestTileEndpointKeepsFullReplayPhaseBandsForClientSideReveal(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/tvac_qualification/tiles?card_id=thermal_program&level=minute", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var tile contracts.GraphTile
	if err := json.Unmarshal(rec.Body.Bytes(), &tile); err != nil {
		t.Fatalf("decode tile: %v", err)
	}
	if len(tile.Bands) == 0 {
		t.Fatal("tile missing phase/dwell bands")
	}
	tileEnd := mustParseTime(tile.T1)
	foundFutureBand := false
	for _, band := range tile.Bands {
		start := mustParseTime(band.Start)
		end := mustParseTime(band.End)
		if start.Before(mustParseTime(tile.T0)) || end.After(tileEnd) {
			t.Fatalf("band %s outside tile range: %s..%s not within %s..%s", band.ID, band.Start, band.End, tile.T0, tile.T1)
		}
		if end.After(mustParseTime("2026-01-25T08:48:36Z")) {
			foundFutureBand = true
		}
	}
	if !foundFutureBand {
		t.Fatalf("expected full replay bands beyond the initial as-run cursor for frontend masking")
	}
}

func TestSwimlaneTileCarriesStateSignals(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns/thermal_acceptance_fat/tiles?card_id=state_change_swimlane&level=minute", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var tile contracts.GraphTile
	if err := json.Unmarshal(rec.Body.Bytes(), &tile); err != nil {
		t.Fatalf("decode tile: %v", err)
	}
	if tile.CardID != "state_change_swimlane" {
		t.Fatalf("card_id = %q", tile.CardID)
	}
	if tile.Diagnostics.PointCount == 0 {
		t.Fatalf("swimlane tile has no state spans")
	}
	series := map[string]contracts.TileSeries{}
	for _, s := range tile.Series {
		series[s.ID] = s
		if !s.Step {
			t.Fatalf("state series %s is not step-rendered", s.ID)
		}
	}
	for _, id := range []string{"trace.phase_enum", "trace.functional_gate_active", "trace.dut_ready", "trace.payload_active", "trace.fault_flag"} {
		if len(series[id].Spans) == 0 {
			t.Fatalf("swimlane missing populated span series %s", id)
		}
		if len(series[id].Points) != 0 {
			t.Fatalf("swimlane series %s should not rely on fake analog points", id)
		}
	}
}

func TestStaticIndexServedWhenConfigured(t *testing.T) {
	server := newTestServerWithStatic(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `<div id="root"></div>`) {
		t.Fatalf("body missing static index: %s", rec.Body.String())
	}
}

func TestStaticAssetServedWhenConfigured(t *testing.T) {
	server := newTestServerWithStatic(t)
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `console.log("gossamer")` {
		t.Fatalf("asset body = %q", got)
	}
}

func TestStaticSymlinkOutsideWebDirIsNotServed(t *testing.T) {
	dir := t.TempDir()
	webDir := filepath.Join(dir, "web")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(webDir, "leak")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	server := NewWithStatic(dir, webDir)
	req := httptest.NewRequest(http.MethodGet, "/leak", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("symlink leak served status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound && rec.Code != http.StatusForbidden {
		t.Fatalf("symlink request status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestStaticDataBundleServedWhenConfigured(t *testing.T) {
	server := newTestServerWithStatic(t)
	dataDir := filepath.Join(server.root, "fixtures", "public_tiles", "current")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "manifest.json"), []byte(`{"schema_version":1,"data_version":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/data/current/manifest.json", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-cache") {
		t.Fatalf("cache header = %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"data_version":"test"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestStaticDataBundleServedWithRelativeRoot(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	dataDir := filepath.Join("fixtures", "public_tiles", "current")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "manifest.json"), []byte(`{"schema_version":1,"data_version":"relative"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	server := New(".")
	req := httptest.NewRequest(http.MethodGet, "/data/current/manifest.json", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"data_version":"relative"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestStaticDataBundleSymlinkOutsideRootIsNotServed(t *testing.T) {
	server := newTestServerWithStatic(t)
	dataDir := filepath.Join(server.root, "fixtures", "public_tiles", "current")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(server.root, "secret.json")
	if err := os.WriteFile(outside, []byte(`{"secret":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dataDir, "leak.json")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/data/current/leak.json", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("data symlink leak served status=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("symlink request status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestStaticArrowServesGzipOnlyBundle(t *testing.T) {
	server := newTestServerWithStatic(t)
	dataDir := filepath.Join(server.root, "fixtures", "public_tiles", "current", "campaigns", "thermal_acceptance_fat")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dataDir, "telemetry.arrow.gz"))
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	if _, err := gz.Write([]byte("arrow-ipc")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/data/current/campaigns/thermal_acceptance_fat/telemetry.arrow", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("content encoding = %q", got)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/vnd.apache.arrow.stream") {
		t.Fatalf("content type = %q", got)
	}
}

func TestUnknownStaticPathFallsBackToIndex(t *testing.T) {
	server := newTestServerWithStatic(t)
	req := httptest.NewRequest(http.MethodGet, "/operator-demo", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `<div id="root"></div>`) {
		t.Fatalf("body missing fallback index: %s", rec.Body.String())
	}
}

func TestUnknownStaticAssetReturnsNotFound(t *testing.T) {
	server := newTestServerWithStatic(t)
	req := httptest.NewRequest(http.MethodGet, "/assets/missing-hash.js", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestUnknownStaticFileReturnsNotFound(t *testing.T) {
	server := newTestServerWithStatic(t)
	req := httptest.NewRequest(http.MethodGet, "/ui-version.txt", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func newTestServerWithStatic(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	if err := synthetic.WritePublicFixtures(dir); err != nil {
		t.Fatal(err)
	}
	webDir := filepath.Join(dir, "web", "dist")
	if err := os.MkdirAll(filepath.Join(webDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte(`<!doctype html><div id="root"></div>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "assets", "app.js"), []byte(`console.log("gossamer")`), 0o644); err != nil {
		t.Fatal(err)
	}
	return NewWithStatic(dir, webDir)
}
