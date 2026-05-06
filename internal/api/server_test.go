package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/egidinas/gossamer/internal/report"
	"github.com/egidinas/gossamer/internal/synthetic"
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

func TestCommandRequiresLease(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/command-authority/mock-command", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
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
