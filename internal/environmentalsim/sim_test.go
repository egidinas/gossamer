package environmentalsim

import (
	"math"
	"testing"
	"time"
)

func TestRadiativeFluxUsesFourthPowerLawForLargeGradient(t *testing.T) {
	const (
		nodeDegC   = 40.0
		shroudDegC = -150.0
		areaM2     = 0.11
		emissivity = 0.82
		sigma      = 5.670374419e-8
	)

	got := radiativeFlux(nodeDegC, shroudDegC, areaM2, emissivity)
	want := emissivity * areaM2 * sigma * (math.Pow(kelvin(shroudDegC), 4) - math.Pow(kelvin(nodeDegC), 4))

	if !almostEqualRelative(got, want, 1e-12) {
		t.Fatalf("radiativeFlux() = %.12g W, want exact fourth-power law %.12g W", got, want)
	}
}

func TestRadiativeFluxAppliesViewFactor(t *testing.T) {
	full := radiativeFlux(42, -145, 0.08, 0.81)
	partial := radiativeFluxWithViewFactor(42, -145, 0.08, 0.81, 0.37)

	if !almostEqualRelative(partial, full*0.37, 1e-12) {
		t.Fatalf("partial-view radiative flux = %.12g W, want %.12g W", partial, full*0.37)
	}
	if got := radiativeFluxWithViewFactor(42, -145, 0.08, 0.81, -0.2); got != 0 {
		t.Fatalf("negative view factor radiative flux = %.12g W, want 0", got)
	}
}

func TestAirCouplingScaleFallsAwayInVacuum(t *testing.T) {
	if got := airCouplingScale("thermal_acceptance_fat", 1e-6); got != 1 {
		t.Fatalf("non-TVAC air coupling = %.12g, want 1", got)
	}
	if got := airCouplingScale("tvac_qualification", 101325); math.Abs(got-1) > 1e-12 {
		t.Fatalf("atmospheric TVAC air coupling = %.12g, want 1", got)
	}
	if got := airCouplingScale("tvac_qualification", 10132.5); got > 0.011 {
		t.Fatalf("0.1 atm TVAC air coupling = %.12g, want pressure-squared scale near 0.01", got)
	}
	if got := airCouplingScale("tvac_qualification", 0.001); got != 0 {
		t.Fatalf("high-vacuum TVAC air coupling = %.12g, want 0 residual convection", got)
	}
}

func TestGasConductanceHasMolecularTailAboveHighVacuumCutoff(t *testing.T) {
	params := componentParams{airConductanceWPerK: 0.40, molecularGasConductanceWPerK: 0.018}

	nearAtmosphere := gasConductanceWPerK("tvac_qualification", standardAtmospherePa, params)
	if math.Abs(nearAtmosphere-0.40) > 1e-12 {
		t.Fatalf("atmospheric gas conductance = %.12g W/K, want continuum conductance", nearAtmosphere)
	}
	if got := gasConductanceWPerK("tvac_qualification", 0.001, params); got != 0 {
		t.Fatalf("deep-vacuum gas conductance = %.12g W/K, want zero below cutoff", got)
	}
	if got := gasConductanceWPerK("tvac_qualification", 10, params); got <= 0 || got >= 0.003 {
		t.Fatalf("transitional molecular gas conductance = %.12g W/K, want small nonzero tail", got)
	}
}

func TestAdvanceComponentIsStableAcrossSubsteps(t *testing.T) {
	params := componentParams{
		capacitanceJPerK:      3200,
		airConductanceWPerK:   0.34,
		tableConductanceWPerK: 0.52,
		radiatingAreaM2:       0.045,
		emissivity:            0.78,
		baseSelfHeatW:         5.8,
		payloadSelfHeatW:      4.5,
		gateSelfHeatW:         15.0,
	}

	oneStep, _ := advanceComponent(35, -35, -28, -135, 0.05, "tvac_qualification", params, true, true, false, 5*time.Minute)

	substep := 35.0
	for i := 0; i < 5; i++ {
		next, _ := advanceComponent(substep, -35, -28, -135, 0.05, "tvac_qualification", params, true, true, false, time.Minute)
		substep = next
	}

	if math.Abs(oneStep-substep) > 0.25 {
		t.Fatalf("5-minute component step = %.4f C, five 1-minute steps = %.4f C, want within 0.25 C", oneStep, substep)
	}
}

func TestAdvanceComponentPairExchangesRadiationBetweenNodes(t *testing.T) {
	fastParams := componentParams{
		capacitanceJPerK:             3200,
		tableConductanceWPerK:        0,
		radiatingAreaM2:              0,
		emissivity:                   0.78,
		coupledRadiatingAreaM2:       0.035,
		coupledRadiatingEmissivity:   0.72,
		coupledRadiatingViewFactor:   0.64,
		molecularGasConductanceWPerK: 0,
	}
	lazyParams := componentParams{
		capacitanceJPerK:             10500,
		tableConductanceWPerK:        0,
		radiatingAreaM2:              0,
		emissivity:                   0.84,
		coupledRadiatingAreaM2:       0.035,
		coupledRadiatingEmissivity:   0.72,
		coupledRadiatingViewFactor:   0.64,
		molecularGasConductanceWPerK: 0,
	}

	fastNext, lazyNext, fastFlux, lazyFlux := advanceComponentPair(70, 10, -20, -20, -120, 0.001, "tvac_qualification", fastParams, lazyParams, false, false, false, time.Minute)

	if fastNext >= 70 {
		t.Fatalf("hot fast node next = %.4f C, want cooling into colder coupled node", fastNext)
	}
	if lazyNext <= 10 {
		t.Fatalf("cold lazy node next = %.4f C, want warming from coupled node", lazyNext)
	}
	if !almostEqualRelative(fastFlux.coupled, -lazyFlux.coupled, 1e-12) {
		t.Fatalf("coupled radiation is not equal/opposite: fast %.12g W lazy %.12g W", fastFlux.coupled, lazyFlux.coupled)
	}
}

func TestSolvePressureStepUsesExponentialPumpLoadModel(t *testing.T) {
	const (
		previous       = 95000.0
		pumpRatePerMin = 0.44
		virtualLeak    = 0.000004
		outgasRate     = 0.000012
		dtMin          = 5.0
	)

	got := solvePressureStep(previous, pumpRatePerMin, virtualLeak, outgasRate, dtMin)
	equilibrium := (virtualLeak + outgasRate) / pumpRatePerMin
	want := equilibrium + (previous-equilibrium)*math.Exp(-pumpRatePerMin*dtMin)

	if !almostEqualRelative(got, want, 1e-12) {
		t.Fatalf("solvePressureStep() = %.12g Pa, want exponential pump/load result %.12g Pa", got, want)
	}
	if got >= previous {
		t.Fatalf("solvePressureStep() = %.12g Pa, want monotonic pumpdown below %.12g Pa", got, previous)
	}
}

func TestPumpRatesReflectCrossoverAndCryopumping(t *testing.T) {
	roughHigh, turboHigh := pumpRatesPerMin(600, -20)
	if roughHigh <= 0 || turboHigh != 0 {
		t.Fatalf("high-pressure pump rates rough=%.12g turbo=%.12g, want roughing only", roughHigh, turboHigh)
	}

	roughCross, turboWarm := pumpRatesPerMin(80, -20)
	_, turboCold := pumpRatesPerMin(80, -145)
	if roughCross <= 0 || turboWarm <= 0 {
		t.Fatalf("crossover pump rates rough=%.12g turbo=%.12g, want both pumps active", roughCross, turboWarm)
	}
	if turboCold <= turboWarm {
		t.Fatalf("cold-shroud turbo/cryo rate = %.12g, want above warm %.12g", turboCold, turboWarm)
	}
}

func TestVolatilePoolDepletionUsesNamedCapacity(t *testing.T) {
	if volatileCapacityPressureMinutes <= 0 {
		t.Fatal("volatileCapacityPressureMinutes must be positive")
	}

	const testPressurePa = 4.5
	const testShroudDegC = 70.0
	_, nextPool, outgasRate, _, _, _, _ := advancePressure("tvac_qualification", testPressurePa, 0.72, 9*time.Hour, "hot_operational", 2, testShroudDegC, 84, 63, 5*time.Minute)
	resorption := stickingCoefficient * testPressurePa * coldSurfaceAreaM2 * coldSurfaceFactor(testShroudDegC) * 5 / volatileCapacityPressureMinutes
	want := clamp(0.72-outgasRate*5/volatileCapacityPressureMinutes+resorption, minimumVolatilePool, 1.0)

	if !almostEqualRelative(nextPool, want, 1e-12) {
		t.Fatalf("next volatile pool = %.12g, want outgas/capacity depletion %.12g", nextPool, want)
	}
}

func almostEqualRelative(got, want, tolerance float64) bool {
	scale := math.Max(1, math.Abs(want))
	return math.Abs(got-want) <= tolerance*scale
}
