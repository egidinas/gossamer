package contracts

import "testing"

func TestValidateManifestAcceptsPublicSyntheticManifest(t *testing.T) {
	manifest := Manifest{
		Envelope:      Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Name:          "Gossamer",
		TestArticle:   "AuroraSat-1",
		Campaigns:     []string{"thermal_acceptance_fat"},
		PublicDemo:    true,
		SyntheticOnly: true,
	}

	if err := ValidateManifest(manifest); err != nil {
		t.Fatalf("expected valid manifest: %v", err)
	}
}

func TestValidateSourceCatalogueRejectsUnknownQuality(t *testing.T) {
	catalogue := SourceCatalogue{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		Sources: []Source{{
			ID:      "thermal_source",
			Owner:   "test_conductor",
			Quality: "mystery",
		}},
	}

	if err := ValidateSourceCatalogue(catalogue); err == nil {
		t.Fatal("expected invalid source quality to fail")
	}
}

func TestValidateGraphModelRejectsSeriesWithoutUnitsOrRole(t *testing.T) {
	model := GraphModel{
		Envelope:   Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		CampaignID: "thermal_acceptance_fat",
		Lanes: []GraphLane{{
			ID: "thermal",
			Series: []GraphSeries{{
				ID: "chamber_air_deg_c",
			}},
		}},
	}

	if err := ValidateGraphModel(model); err == nil {
		t.Fatal("expected missing units and role to fail")
	}
}

func TestValidateCampaignRejectsUnknownRequirementResult(t *testing.T) {
	campaign := Campaign{
		Envelope: Envelope{SchemaVersion: 1, GeneratedAt: "2026-01-15T10:00:00Z"},
		ID:       "thermal_acceptance_fat",
		Name:     "Thermal Acceptance FAT",
		Result:   "pass",
		Requirements: []Requirement{{
			ID:     "REQ-DATA-QUALITY",
			Result: "maybe",
		}},
	}

	if err := ValidateCampaign(campaign); err == nil {
		t.Fatal("expected unknown requirement result to fail")
	}
}

