package environmentalsim

import (
	"strings"
	"testing"
	"time"

	"github.com/egidinas/signalforge/contracts"
)

func TestSimulateRejectsMalformedProgramsWithoutPanic(t *testing.T) {
	for _, tc := range []struct {
		name    string
		program *contracts.ThermalProgram
	}{
		{name: "nil", program: nil},
		{name: "empty cycles", program: &contracts.ThermalProgram{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Simulate("malformed", tc.program, time.Unix(0, 0))
			if len(got.Samples) != 0 {
				t.Fatalf("samples = %d, want 0", len(got.Samples))
			}
			if got.Provenance.Model != ModelName {
				t.Fatalf("model = %q, want %q", got.Provenance.Model, ModelName)
			}
			if !strings.Contains(got.Provenance.Source, "invalid thermal program") {
				t.Fatalf("source = %q, want invalid program marker", got.Provenance.Source)
			}
		})
	}
}
