package environmentalsim

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/egidinas/gossamer/internal/contracts"
)

const (
	ModelName    = "multi_path_dut_thermal_rc_v2"
	ModelVersion = "2026.05.1"
)

type Result struct {
	Provenance contracts.SimulationProvenance
	Samples    []contracts.TelemetrySample
	HeroGraph  contracts.HeroGraphModel
}

type state struct {
	chamberAir             float64
	table                  float64
	shroud                 float64
	shroudInlet            float64
	shroudOutlet           float64
	exhaustCryoTemp        float64
	exhaustScavengedTemp   float64
	scavengerWaterReturn   float64
	exhaustColdRecoveryPct float64
	fastComponent          float64
	lazyComponent          float64
	pressure               float64
	volatilePool           float64
	outgasRate             float64
	virtualLeak            float64
	roughingRate           float64
	turboRate              float64
	totalPumpRate          float64
	tmPackets              float64
	tcPackets              float64
	drops                  float64
}

type componentParams struct {
	capacitanceJPerK      float64
	airConductanceWPerK   float64
	tableConductanceWPerK float64
	radiatingAreaM2       float64
	emissivity            float64
	baseSelfHeatW         float64
	payloadSelfHeatW      float64
	gateSelfHeatW         float64
	sensorBiasDegC        float64
	sensorNoiseDegC       float64
}

type heatFlux struct {
	air    float64
	iface  float64
	shroud float64
	self   float64
}

func Simulate(campaignID string, program *contracts.ThermalProgram, start time.Time) Result {
	seed := int64(42017)
	if campaignID == "tvac_qualification" {
		seed = 84031
	}
	rng := rand.New(rand.NewSource(seed))
	dt := 5 * time.Minute
	st := state{chamberAir: 22, table: 22, shroud: 22.4, shroudInlet: 22.2, shroudOutlet: 22.7, exhaustCryoTemp: 20, exhaustScavengedTemp: 20, scavengerWaterReturn: 15.8, fastComponent: 21.2, lazyComponent: 22.6, pressure: 101325, volatilePool: 1, tmPackets: 6000, tcPackets: 120}
	fastNode := componentParams{
		capacitanceJPerK: 3200, airConductanceWPerK: 0.34, tableConductanceWPerK: 0.52,
		radiatingAreaM2: 0.045, emissivity: 0.78, baseSelfHeatW: 5.8, payloadSelfHeatW: 4.5, gateSelfHeatW: 15.0,
		sensorBiasDegC: 0.14, sensorNoiseDegC: 0.08,
	}
	lazyNode := componentParams{
		capacitanceJPerK: 18500, airConductanceWPerK: 0.24, tableConductanceWPerK: 0.045,
		radiatingAreaM2: 0.105, emissivity: 0.82, baseSelfHeatW: 0.9, payloadSelfHeatW: 0.9, gateSelfHeatW: 2.8,
		sensorBiasDegC: -0.20, sensorNoiseDegC: 0.10,
	}
	samples := []contracts.TelemetrySample{}
	trace := sampleTrace{}

	appendPoint := func(t time.Time, cycle int, phaseID string, phase string, phaseCode float64, command float64, gate string, gateCode float64, targetPressure float64) {
		gateActive := gate != "" && gate != "none"
		survivalMode := isSurvivalPhase(phase)
		coolingDemand := clamp((st.chamberAir-command)/38, 0, 1)
		heatingDemand := clamp((command-st.chamberAir)/42, 0, 1)
		if phase == "ramp_cold" && command < 5 {
			coolingDemand = math.Max(coolingDemand, 0.78)
		}
		ln2Duty := 0.0
		if command < -5 || coolingDemand > 0.05 {
			ln2Duty = clamp(18+90*coolingDemand+4*math.Sin(float64(len(samples))*0.9), 0, 96)
		}
		heaterDuty := clamp(12+78*heatingDemand, 0, 96)
		tauAir := 18.0
		if phase == "ramp_cold" && math.Abs(command-program.ColdTargetDegC) < 8 {
			tauAir = 32.0
		}
		if isThermalDwellPhase(phase) {
			tauAir = 11.0
		}
		actuatorPush := 0.035 * (heaterDuty - ln2Duty)
		st.chamberAir += firstOrderDelta(st.chamberAir, command, tauAir, dt) + actuatorPush*0.05
		st.chamberAir += noise(rng, 0.08)

		tableTarget := st.chamberAir
		tableTau := 33.0
		if campaignID == "tvac_qualification" {
			tableTarget = command + 0.18*(st.chamberAir-command)
			tableTau = 44.0
		}
		st.table += firstOrderDelta(st.table, tableTarget, tableTau, dt) + functionalHeat(gateActive)*0.012
		st.table += noise(rng, 0.035)

		shroudTarget := st.chamberAir
		shroudTau := 16.0
		if campaignID == "tvac_qualification" {
			shroudTarget = command
			shroudTau = 28.0
			if phase == "ramp_cold" {
				shroudTau = 45.0
			}
			if isThermalDwellPhase(phase) {
				shroudTau = 18.0
			}
			if phase == "cold_survival" {
				shroudTarget = command - 38
				shroudTau = 22.0
			}
		}
		if campaignID == "tvac_qualification" {
			ln2Gradient := 4.5 + 0.20*ln2Duty
			if phase == "cold_survival" {
				ln2Gradient += 10
			}
			inletTarget := shroudTarget - ln2Gradient*0.55
			outletTarget := shroudTarget + ln2Gradient*0.45
			if heaterDuty > ln2Duty {
				inletTarget = shroudTarget - 1.4
				outletTarget = shroudTarget + 1.2
			}
			st.shroudInlet += firstOrderDelta(st.shroudInlet, inletTarget, shroudTau*0.82, dt)
			st.shroudOutlet += firstOrderDelta(st.shroudOutlet, outletTarget, shroudTau*1.18, dt)
			gradientRelax := firstOrderDelta(st.shroudOutlet-st.shroudInlet, ln2Gradient, 95, dt)
			st.shroudInlet -= gradientRelax * 0.12
			st.shroudOutlet += gradientRelax * 0.08
			st.shroud = 0.48*st.shroudInlet + 0.52*st.shroudOutlet
			st.shroudInlet += noise(rng, 0.055)
			st.shroudOutlet += noise(rng, 0.05)
			st.shroud += noise(rng, 0.04)
		} else {
			st.shroud += firstOrderDelta(st.shroud, shroudTarget, shroudTau, dt)
			st.shroud += noise(rng, 0.045)
			st.shroudInlet = st.shroud
			st.shroudOutlet = st.shroud
		}

		payloadActive := !survivalMode && (gateActive || isOperationalDwellPhase(phase))
		var fastFlux, lazyFlux heatFlux
		st.fastComponent, fastFlux = advanceComponent(st.fastComponent, st.chamberAir, st.table, st.shroud, st.pressure, campaignID, fastNode, gateActive, payloadActive, survivalMode, dt)
		st.lazyComponent, lazyFlux = advanceComponent(st.lazyComponent, st.chamberAir, st.table, st.shroud, st.pressure, campaignID, lazyNode, gateActive, payloadActive, survivalMode, dt)
		fastMeasured := st.fastComponent + fastNode.sensorBiasDegC + noise(rng, fastNode.sensorNoiseDegC)
		lazyMeasured := st.lazyComponent + lazyNode.sensorBiasDegC + noise(rng, lazyNode.sensorNoiseDegC)
		shroudMeasured := st.shroud + noise(rng, 0.06)
		shroudInletMeasured := st.shroudInlet + noise(rng, 0.07)
		shroudOutletMeasured := st.shroudOutlet + noise(rng, 0.07)
		tableMeasured := st.table + noise(rng, 0.04)
		internalGradient := math.Abs(fastMeasured - lazyMeasured)

		pressure, volatilePool, outgasRate, virtualLeak, roughingRate, turboRate, totalPumpRate := advancePressure(campaignID, st.pressure, st.volatilePool, t.Sub(start), phase, cycle, st.shroud, st.fastComponent, st.lazyComponent, dt)
		st.pressure, st.volatilePool, st.outgasRate, st.virtualLeak, st.roughingRate, st.turboRate, st.totalPumpRate = pressure, volatilePool, outgasRate, virtualLeak, roughingRate, turboRate, totalPumpRate
		stabilityReached, dwellActive, dwellComplete := dwellStateFor(program, phaseID, t)
		pressureGateReached := pressureGateState(campaignID, phase, st.pressure)
		load := 4.0 + 0.025*math.Max(0, heaterDuty) + functionalHeat(gateActive)*0.11 + 0.006*math.Max(0, st.fastComponent-22) + 0.18*math.Sin(float64(len(samples))/17)
		latency := 18.0 + 4*math.Sin(float64(len(samples))/13)
		if survivalMode {
			latency += 4
			st.tcPackets += 0.12
		} else if gateActive {
			latency += 18
			st.tcPackets += 5
		} else {
			st.tcPackets += 1
		}
		if campaignID == "tvac_qualification" && cycle == 6 && phase == "cold_operational" {
			latency += 9
			st.drops += 0.25
		}
		if survivalMode {
			st.tmPackets += 1.8
		} else {
			st.tmPackets += 12 + functionalHeat(gateActive)*0.8
		}
		quality := "fresh"
		freshness := 230.0 + 40*math.Abs(noise(rng, 1))
		interlock := "closed"
		interlockCode := 1.0
		if campaignID == "tvac_qualification" && cycle == 6 && phase == "cold_operational" {
			quality = "degraded"
			freshness = 5200
			interlock = "review"
			interlockCode = 2
		}
		dutReady := !survivalMode && (phase != "ambient_precheck" || gateActive)
		dutOperative := dutReady && interlock == "closed" && !survivalMode
		rfLocked := quality != "degraded" && !survivalMode
		faultFlag := interlock != "closed" || quality == "degraded"
		busVoltage := 28.1 - 0.035*load + noise(rng, 0.025)
		payloadPower := 12 + heaterDuty*0.42 + functionalHeat(gateActive)*1.8
		avionicsPower := 33 + 0.55*functionalHeat(gateActive) + 0.12*math.Max(0, st.fastComponent-22)
		linkPower := 8.0
		if rfLocked {
			linkPower = 10.5
		}
		if gateActive {
			linkPower += 4.2
		}
		thermalControlPower := math.Max(0, heaterDuty*0.36)
		if survivalMode {
			load = 0.9 + 0.04*math.Sin(float64(len(samples))/11)
			payloadPower = 0.8
			avionicsPower = 7.5
			linkPower = 0.5
			thermalControlPower = 0
		}
		totalPower := busVoltage*load + payloadPower + thermalControlPower + avionicsPower + linkPower
		subsystemPower := avionicsPower + linkPower + payloadPower
		ln2Line := 18.0
		if ln2Duty > 5 {
			ln2Line = -42 - ln2Duty*1.25 + 5*math.Sin(float64(len(samples))/5)
			if campaignID == "tvac_qualification" {
				ln2Line -= 36
			}
		}
		coolingWaterTemp := 15.2 + 0.018*heaterDuty + 0.012*ln2Duty + 0.35*math.Sin(float64(len(samples))/47) + noise(rng, 0.04)
		exhaustDuctSafe := false
		freezeMargin := clamp(18+0.10*st.table-0.045*ln2Duty, 2.8, 24)
		if campaignID == "tvac_qualification" {
			flowShortfall := clamp((ln2Duty-72)/34, 0, 1)
			cryoTarget := 18.0
			if ln2Duty > 4 {
				cryoTarget = clamp(-162+0.58*(st.shroudOutlet+120)+0.08*ln2Line, -172, 16)
			}
			st.exhaustCryoTemp += firstOrderDelta(st.exhaustCryoTemp, cryoTarget, 17, dt)
			coldLoad := clamp((18-st.exhaustCryoTemp)/186, 0, 1)
			waterFlowFactor := clamp(1.0-0.26*flowShortfall+0.03*math.Sin(float64(len(samples))/29), 0.62, 1.04)
			st.exhaustColdRecoveryPct = clamp(100*coldLoad*waterFlowFactor, 0, 92)
			scavengedTarget := clamp(16.5-2.8*coldLoad+7.0*(1-waterFlowFactor)+0.014*heaterDuty, 7.2, 23)
			st.exhaustScavengedTemp += firstOrderDelta(st.exhaustScavengedTemp, scavengedTarget, 12, dt)
			returnTarget := coolingWaterTemp + 1.2 + 8.8*coldLoad*waterFlowFactor
			st.scavengerWaterReturn += firstOrderDelta(st.scavengerWaterReturn, returnTarget, 16, dt)
			freezeMargin = clamp(5.4+0.50*(st.scavengerWaterReturn-coolingWaterTemp)-0.035*ln2Duty, 0.7, 24)
			exhaustDuctSafe = st.exhaustColdRecoveryPct > 30 && st.exhaustScavengedTemp > 4 && st.exhaustCryoTemp < 0
		} else {
			st.exhaustCryoTemp += firstOrderDelta(st.exhaustCryoTemp, 20, 28, dt)
			st.exhaustScavengedTemp += firstOrderDelta(st.exhaustScavengedTemp, 20, 28, dt)
			st.scavengerWaterReturn += firstOrderDelta(st.scavengerWaterReturn, coolingWaterTemp+0.4, 20, dt)
			st.exhaustColdRecoveryPct = 0
		}
		airSupply := 6.25 - 0.035*math.Max(coolingDemand, heatingDemand) - 0.012*functionalHeat(gateActive) + noise(rng, 0.006)
		airDewpoint := -42.0 + 0.018*coolingWaterTemp + noise(rng, 0.08)
		overallPackets := st.tmPackets + st.tcPackets
		if gate == "" {
			gate = "none"
		}
		sample := contracts.TelemetrySample{
			Timestamp: t.Format(time.RFC3339),
			Quality:   quality,
			Signals: map[string]float64{
				"eps_bus_voltage_v":                         round(busVoltage),
				"eps_bus_current_a":                         round(load),
				"obc_command_counter":                       round(1000 + float64(len(samples))*2 + st.tcPackets/5),
				"payload_sim_heater_w":                      round(payloadPower),
				"dut_power_total_w":                         round(totalPower),
				"dut_power_subsystem_w":                     round(subsystemPower),
				"dut_power_avionics_w":                      round(avionicsPower),
				"dut_power_payload_w":                       round(payloadPower),
				"dut_power_link_w":                          round(linkPower),
				"dut_power_thermal_control_w":               round(thermalControlPower),
				"thermal_cycle_index":                       float64(cycle),
				"thermal_phase_code":                        phaseCode,
				"chamber_setpoint_deg_c":                    round(command),
				"chamber_air_deg_c":                         round(st.chamberAir),
				"thermal_zone_1_deg_c":                      round(fastMeasured),
				"thermal_zone_2_deg_c":                      round(lazyMeasured),
				"dut_fast_component_deg_c":                  round(fastMeasured),
				"dut_lazy_component_deg_c":                  round(lazyMeasured),
				"dut_internal_gradient_deg_c":               round(internalGradient),
				"huber_table_deg_c":                         round(tableMeasured),
				"interface_plate_deg_c":                     round(tableMeasured),
				"thermal_shroud_deg_c":                      round(shroudMeasured),
				"thermal_shroud_inlet_deg_c":                round(shroudInletMeasured),
				"thermal_shroud_outlet_deg_c":               round(shroudOutletMeasured),
				"thermal_shroud_gradient_deg_c":             round(math.Abs(shroudOutletMeasured - shroudInletMeasured)),
				"dut_fast_air_flux_w":                       round(fastFlux.air),
				"dut_fast_interface_flux_w":                 round(fastFlux.iface),
				"dut_fast_shroud_flux_w":                    round(fastFlux.shroud),
				"dut_lazy_air_flux_w":                       round(lazyFlux.air),
				"dut_lazy_interface_flux_w":                 round(lazyFlux.iface),
				"dut_lazy_shroud_flux_w":                    round(lazyFlux.shroud),
				"dut_self_heat_w":                           round(fastFlux.self + lazyFlux.self),
				"ln2_line_temp_deg_c":                       round(ln2Line),
				"ln2_valve_duty_pct":                        round(ln2Duty),
				"cooling_water_freeze_margin_deg_c":         round(freezeMargin),
				"cooling_water_temp_deg_c":                  round(coolingWaterTemp),
				"tvac_cryo_exhaust_temp_deg_c":              round(st.exhaustCryoTemp),
				"tvac_scavenged_exhaust_temp_deg_c":         round(st.exhaustScavengedTemp),
				"tvac_scavenger_cooling_water_return_deg_c": round(st.scavengerWaterReturn),
				"tvac_exhaust_cold_recovery_pct":            round(st.exhaustColdRecoveryPct),
				"tvac_exhaust_duct_safe":                    boolValue(exhaustDuctSafe),
				"pressurized_air_supply_bar":                round(airSupply),
				"air_dewpoint_deg_c":                        round(airDewpoint),
				"tvac_pressure_pa":                          round(st.pressure),
				"tvac_pressure_mbar":                        round(st.pressure * 0.01),
				"tvac_outgassing_pa_per_min":                round(st.outgasRate),
				"tvac_outgassing_mbar_per_min":              round(st.outgasRate * 0.01),
				"tvac_virtual_leak_pa_per_min":              round(st.virtualLeak),
				"tvac_virtual_leak_mbar_per_min":            round(st.virtualLeak * 0.01),
				"tvac_roughing_removal_pa_per_min":          round(st.roughingRate),
				"tvac_roughing_removal_mbar_per_min":        round(st.roughingRate * 0.01),
				"tvac_turbo_removal_pa_per_min":             round(st.turboRate),
				"tvac_turbo_removal_mbar_per_min":           round(st.turboRate * 0.01),
				"tvac_pump_removal_pa_per_min":              round(st.totalPumpRate),
				"tvac_pump_removal_mbar_per_min":            round(st.totalPumpRate * 0.01),
				"tvac_pump_mode_code":                       pumpModeCode(st.pressure),
				"tvac_volatile_inventory_pct":               round(100 * st.volatilePool),
				"source_freshness_ms":                       round(freshness),
				"facility_interlock_code":                   interlockCode,
				"functional_gate_code":                      gateCode,
				"dut_ready":                                 boolValue(dutReady),
				"dut_operative":                             boolValue(dutOperative),
				"dut_survival_mode":                         boolValue(survivalMode),
				"stability_gate_reached":                    boolValue(stabilityReached),
				"dwell_active":                              boolValue(dwellActive),
				"dwell_complete":                            boolValue(dwellComplete),
				"pressure_gate_reached":                     boolValue(pressureGateReached),
				"payload_active":                            boolValue(payloadActive),
				"rf_link_locked":                            boolValue(rfLocked),
				"fault_flag":                                boolValue(faultFlag),
				"bus_latency_ms":                            round(latency),
				"tm_packet_counter":                         round(st.tmPackets),
				"tc_packet_counter":                         round(st.tcPackets),
				"overall_packet_counter":                    round(overallPackets),
				"dropped_frame_count":                       round(st.drops),
				"rf_link_margin_db":                         round(8.8 - 0.003*math.Abs(st.fastComponent-22) - 0.0015*internalGradient + noise(rng, 0.06)),
			},
			States: map[string]string{
				"obc_boot_state":           "nominal",
				"rf_link_sim_state":        stateName(rfLocked, "locked", "searching"),
				"facility_interlock_state": interlock,
				"thermal_phase":            phase,
				"functional_gate":          gate,
				"dut_ready_state":          stateName(dutReady, "ready", "not_ready"),
				"dut_operative_state":      stateName(dutOperative, "operative", "inhibited"),
				"dut_survival_state":       stateName(survivalMode, "survival", "nominal"),
				"stability_state":          stateName(stabilityReached, "stable", "stabilizing"),
				"dwell_state":              dwellStateName(dwellActive, dwellComplete),
				"pressure_gate_state":      stateName(pressureGateReached, "pressure_gate", "pressure_wait"),
				"payload_active_state":     stateName(payloadActive, "active", "standby"),
				"fault_flag_state":         stateName(faultFlag, "fault", "nominal"),
				"exhaust_duct_state":       stateName(exhaustDuctSafe, "duct_safe", "scavenger_warming"),
			},
		}
		samples = append(samples, sample)
		trace.add(sample, command, acceptanceTarget(command, program), gateActive, interlock != "closed")
	}

	firstCycle := program.Cycles[0]
	preEnd := mustTime(firstCycle.Start)
	preSteps := int(preEnd.Sub(start) / dt)
	if preSteps < 1 {
		preSteps = 1
	}
	for i := 0; i <= preSteps; i++ {
		t := start.Add(time.Duration(i) * dt)
		if t.After(preEnd) {
			t = preEnd
		}
		gate, gateCode := gateFor(program, "ambient_precheck", t)
		appendPoint(t, 0, "ambient_precheck", "ambient_precheck", thermalPhaseCode("ambient_precheck"), 22, gate, gateCode, 101325)
	}
	lastCommand := 22.0
	for _, cycle := range program.Cycles {
		for _, phase := range cycle.Phases {
			phaseStart := mustTime(phase.Start)
			phaseEnd := mustTime(phase.End)
			steps := int(phaseEnd.Sub(phaseStart) / dt)
			if steps < 1 {
				steps = 1
			}
			from := lastCommand
			startStep := 0
			if len(samples) > 0 && samples[len(samples)-1].Timestamp == phaseStart.Format(time.RFC3339) {
				startStep = 1
			}
			for i := startStep; i <= steps; i++ {
				t := phaseStart.Add(time.Duration(i) * dt)
				f := float64(i) / float64(steps)
				command := phase.TargetDegC
				if phase.Kind == "ramp_cold" || phase.Kind == "ramp_hot" {
					command = from + (phase.TargetDegC-from)*smoothRamp(f)
				}
				gate, gateCode := gateFor(program, phase.ID, t)
				appendPoint(t, cycle.Index, phase.ID, phase.Kind, thermalPhaseCode(phase.Kind), command, gate, gateCode, st.pressure)
			}
			lastCommand = phase.TargetDegC
		}
	}
	lastCycle := program.Cycles[len(program.Cycles)-1]
	postStart := mustTime(lastCycle.End)
	postDuration := thermalContextDuration(program)
	postSteps := int(postDuration / dt)
	if postSteps < 18 {
		postSteps = 18
	}
	vacuumHoldSteps := 0
	if campaignID == "tvac_qualification" {
		vacuumHoldSteps = int((2 * time.Hour) / dt)
		if vacuumHoldSteps < 8 {
			vacuumHoldSteps = 8
		}
		if vacuumHoldSteps > postSteps/2 {
			vacuumHoldSteps = postSteps / 2
		}
	}
	for i := 1; i <= postSteps; i++ {
		t := postStart.Add(time.Duration(i) * dt)
		f := float64(i) / float64(postSteps)
		command := lastCommand + (22-lastCommand)*smoothRamp(f)
		phaseID := "ambient_postcheck"
		if campaignID == "tvac_qualification" && i <= vacuumHoldSteps {
			phaseID = "ambient_postcheck_vacuum"
		}
		gate, gateCode := gateFor(program, phaseID, t)
		appendPoint(t, 0, phaseID, phaseID, thermalPhaseCode(phaseID), command, gate, gateCode, st.pressure)
	}

	provenance := contracts.SimulationProvenance{
		Model:         ModelName,
		ModelVersion:  ModelVersion,
		Seed:          seed,
		StepSeconds:   int(dt.Seconds()),
		Source:        "gossamer internal/environmentalsim deterministic fixture",
		Deterministic: true,
		Parameters: map[string]float64{
			"chamber_air_tau_min":                18,
			"interface_plate_tau_min":            33,
			"tvac_shroud_nominal_tau_min":        28,
			"fast_component_capacitance_j_per_k": fastNode.capacitanceJPerK,
			"lazy_component_capacitance_j_per_k": lazyNode.capacitanceJPerK,
			"fast_air_conductance_w_per_k":       fastNode.airConductanceWPerK,
			"lazy_air_conductance_w_per_k":       lazyNode.airConductanceWPerK,
			"fast_interface_conductance_w_per_k": fastNode.tableConductanceWPerK,
			"lazy_interface_conductance_w_per_k": lazyNode.tableConductanceWPerK,
			"lazy_radiating_area_m2":             lazyNode.radiatingAreaM2,
			"tvac_min_air_coupling_factor":       0.015,
			"tvac_effective_pump_rate_per_min":   0.185,
			"tvac_nominal_pump_rate_per_min":     0.185,
			"tvac_base_virtual_leak_pa_per_min":  0.0000051,
			"tvac_baked_ultimate_pressure_pa":    0.0000051 * (0.35 + 0.65*0.018) / 0.185,
			"tvac_baked_ultimate_pressure_mbar":  (0.0000051 * (0.35 + 0.65*0.018) / 0.185) * 0.01,
			"tvac_cryo_pump_max_multiplier":      3.2,
			"tvac_exhaust_scavenger_min_safe_c":  4,
			"tvac_exhaust_water_flow_nominal":    1.0,
			"functional_gate_fast_self_heat_w":   fastNode.gateSelfHeatW,
			"functional_gate_lazy_self_heat_w":   lazyNode.gateSelfHeatW,
		},
	}
	return Result{
		Provenance: provenance,
		Samples:    samples,
		HeroGraph:  buildHeroGraph(campaignID, program, provenance, samples, trace),
	}
}

type sampleTrace struct {
	command       []contracts.GraphPoint
	ghost         []contracts.GraphPoint
	actual        []contracts.GraphPoint
	zone1         []contracts.GraphPoint
	zone2         []contracts.GraphPoint
	table         []contracts.GraphPoint
	shroud        []contracts.GraphPoint
	shroudInlet   []contracts.GraphPoint
	shroudOutlet  []contracts.GraphPoint
	shroudDelta   []contracts.GraphPoint
	gradient      []contracts.GraphPoint
	fastAirFlux   []contracts.GraphPoint
	fastIFace     []contracts.GraphPoint
	fastShroud    []contracts.GraphPoint
	lazyAirFlux   []contracts.GraphPoint
	lazyIFace     []contracts.GraphPoint
	lazyShroud    []contracts.GraphPoint
	selfHeat      []contracts.GraphPoint
	pressureMbar  []contracts.GraphPoint
	outgasMbar    []contracts.GraphPoint
	virtualLeak   []contracts.GraphPoint
	roughingRate  []contracts.GraphPoint
	turboRate     []contracts.GraphPoint
	pumpRemoval   []contracts.GraphPoint
	pumpMode      []contracts.GraphPoint
	volatilePool  []contracts.GraphPoint
	ln2           []contracts.GraphPoint
	freeze        []contracts.GraphPoint
	cryoExhaust   []contracts.GraphPoint
	scavExhaust   []contracts.GraphPoint
	scavWater     []contracts.GraphPoint
	coldRecovery  []contracts.GraphPoint
	load          []contracts.GraphPoint
	powerTotal    []contracts.GraphPoint
	powerSubsys   []contracts.GraphPoint
	powerAvionics []contracts.GraphPoint
	powerPayload  []contracts.GraphPoint
	powerLink     []contracts.GraphPoint
	powerThermal  []contracts.GraphPoint
	busLatency    []contracts.GraphPoint
	quality       []contracts.GraphPoint
	overall       []contracts.GraphPoint
	tmCounter     []contracts.GraphPoint
	tcCounter     []contracts.GraphPoint
	dropCount     []contracts.GraphPoint
	coolingWater  []contracts.GraphPoint
	airSupply     []contracts.GraphPoint
	airDewpoint   []contracts.GraphPoint
	phase         []contracts.GraphPoint
	degraded      []contracts.GraphPoint
	gates         []contracts.GraphPoint
	interlocks    []contracts.GraphPoint
	evidence      []contracts.GraphPoint
	ready         []contracts.GraphPoint
	operative     []contracts.GraphPoint
	survival      []contracts.GraphPoint
	stability     []contracts.GraphPoint
	dwellActive   []contracts.GraphPoint
	dwellComplete []contracts.GraphPoint
	pressureGate  []contracts.GraphPoint
	exhaustSafe   []contracts.GraphPoint
	payload       []contracts.GraphPoint
	rfLocked      []contracts.GraphPoint
	fault         []contracts.GraphPoint
}

func (t *sampleTrace) add(sample contracts.TelemetrySample, command, acceptance float64, gate, interlock bool) {
	ts := sample.Timestamp
	t.command = append(t.command, point(ts, command))
	t.ghost = append(t.ghost, point(ts, command))
	t.actual = append(t.actual, point(ts, sample.Signals["chamber_air_deg_c"]))
	t.zone1 = append(t.zone1, point(ts, sample.Signals["dut_fast_component_deg_c"]))
	t.zone2 = append(t.zone2, point(ts, sample.Signals["dut_lazy_component_deg_c"]))
	t.table = append(t.table, point(ts, sample.Signals["interface_plate_deg_c"]))
	t.shroud = append(t.shroud, point(ts, sample.Signals["thermal_shroud_deg_c"]))
	t.shroudInlet = append(t.shroudInlet, point(ts, sample.Signals["thermal_shroud_inlet_deg_c"]))
	t.shroudOutlet = append(t.shroudOutlet, point(ts, sample.Signals["thermal_shroud_outlet_deg_c"]))
	t.shroudDelta = append(t.shroudDelta, point(ts, sample.Signals["thermal_shroud_gradient_deg_c"]))
	t.gradient = append(t.gradient, point(ts, sample.Signals["dut_internal_gradient_deg_c"]))
	t.fastAirFlux = append(t.fastAirFlux, point(ts, sample.Signals["dut_fast_air_flux_w"]))
	t.fastIFace = append(t.fastIFace, point(ts, sample.Signals["dut_fast_interface_flux_w"]))
	t.fastShroud = append(t.fastShroud, point(ts, sample.Signals["dut_fast_shroud_flux_w"]))
	t.lazyAirFlux = append(t.lazyAirFlux, point(ts, sample.Signals["dut_lazy_air_flux_w"]))
	t.lazyIFace = append(t.lazyIFace, point(ts, sample.Signals["dut_lazy_interface_flux_w"]))
	t.lazyShroud = append(t.lazyShroud, point(ts, sample.Signals["dut_lazy_shroud_flux_w"]))
	t.selfHeat = append(t.selfHeat, point(ts, sample.Signals["dut_self_heat_w"]))
	t.pressureMbar = append(t.pressureMbar, point(ts, sample.Signals["tvac_pressure_mbar"]))
	t.outgasMbar = append(t.outgasMbar, point(ts, sample.Signals["tvac_outgassing_mbar_per_min"]))
	t.virtualLeak = append(t.virtualLeak, point(ts, sample.Signals["tvac_virtual_leak_mbar_per_min"]))
	t.roughingRate = append(t.roughingRate, point(ts, sample.Signals["tvac_roughing_removal_mbar_per_min"]))
	t.turboRate = append(t.turboRate, point(ts, sample.Signals["tvac_turbo_removal_mbar_per_min"]))
	t.pumpRemoval = append(t.pumpRemoval, point(ts, sample.Signals["tvac_pump_removal_mbar_per_min"]))
	t.pumpMode = append(t.pumpMode, point(ts, sample.Signals["tvac_pump_mode_code"]))
	t.volatilePool = append(t.volatilePool, point(ts, sample.Signals["tvac_volatile_inventory_pct"]))
	t.ln2 = append(t.ln2, point(ts, sample.Signals["ln2_valve_duty_pct"]))
	t.freeze = append(t.freeze, point(ts, sample.Signals["cooling_water_freeze_margin_deg_c"]))
	t.cryoExhaust = append(t.cryoExhaust, point(ts, sample.Signals["tvac_cryo_exhaust_temp_deg_c"]))
	t.scavExhaust = append(t.scavExhaust, point(ts, sample.Signals["tvac_scavenged_exhaust_temp_deg_c"]))
	t.scavWater = append(t.scavWater, point(ts, sample.Signals["tvac_scavenger_cooling_water_return_deg_c"]))
	t.coldRecovery = append(t.coldRecovery, point(ts, sample.Signals["tvac_exhaust_cold_recovery_pct"]))
	t.load = append(t.load, point(ts, sample.Signals["eps_bus_current_a"]))
	t.powerTotal = append(t.powerTotal, point(ts, sample.Signals["dut_power_total_w"]))
	t.powerSubsys = append(t.powerSubsys, point(ts, sample.Signals["dut_power_subsystem_w"]))
	t.powerAvionics = append(t.powerAvionics, point(ts, sample.Signals["dut_power_avionics_w"]))
	t.powerPayload = append(t.powerPayload, point(ts, sample.Signals["dut_power_payload_w"]))
	t.powerLink = append(t.powerLink, point(ts, sample.Signals["dut_power_link_w"]))
	t.powerThermal = append(t.powerThermal, point(ts, sample.Signals["dut_power_thermal_control_w"]))
	t.busLatency = append(t.busLatency, point(ts, sample.Signals["bus_latency_ms"]))
	t.quality = append(t.quality, point(ts, sample.Signals["source_freshness_ms"]))
	t.overall = append(t.overall, point(ts, sample.Signals["overall_packet_counter"]))
	t.tmCounter = append(t.tmCounter, point(ts, sample.Signals["tm_packet_counter"]))
	t.tcCounter = append(t.tcCounter, point(ts, sample.Signals["tc_packet_counter"]))
	t.dropCount = append(t.dropCount, point(ts, sample.Signals["dropped_frame_count"]))
	t.coolingWater = append(t.coolingWater, point(ts, sample.Signals["cooling_water_temp_deg_c"]))
	t.airSupply = append(t.airSupply, point(ts, sample.Signals["pressurized_air_supply_bar"]))
	t.airDewpoint = append(t.airDewpoint, point(ts, sample.Signals["air_dewpoint_deg_c"]))
	t.phase = append(t.phase, point(ts, sample.Signals["thermal_phase_code"]))
	t.degraded = append(t.degraded, point(ts, boolValue(sample.Quality == "degraded")))
	t.gates = append(t.gates, point(ts, boolValue(gate)))
	t.interlocks = append(t.interlocks, point(ts, boolValue(interlock)))
	t.evidence = append(t.evidence, point(ts, boolValue(gate || interlock)))
	t.ready = append(t.ready, point(ts, sample.Signals["dut_ready"]))
	t.operative = append(t.operative, point(ts, sample.Signals["dut_operative"]))
	t.survival = append(t.survival, point(ts, sample.Signals["dut_survival_mode"]))
	t.stability = append(t.stability, point(ts, sample.Signals["stability_gate_reached"]))
	t.dwellActive = append(t.dwellActive, point(ts, sample.Signals["dwell_active"]))
	t.dwellComplete = append(t.dwellComplete, point(ts, sample.Signals["dwell_complete"]))
	t.pressureGate = append(t.pressureGate, point(ts, sample.Signals["pressure_gate_reached"]))
	t.exhaustSafe = append(t.exhaustSafe, point(ts, sample.Signals["tvac_exhaust_duct_safe"]))
	t.payload = append(t.payload, point(ts, sample.Signals["payload_active"]))
	t.rfLocked = append(t.rfLocked, point(ts, sample.Signals["rf_link_locked"]))
	t.fault = append(t.fault, point(ts, sample.Signals["fault_flag"]))
}

func buildHeroGraph(campaignID string, program *contracts.ThermalProgram, provenance contracts.SimulationProvenance, samples []contracts.TelemetrySample, trace sampleTrace) contracts.HeroGraphModel {
	start := samples[0].Timestamp
	end := samples[len(samples)-1].Timestamp
	execution := buildExecutionState(program, start, end)
	axes := []contracts.GraphYAxis{
		{ID: "temperature_c", Label: "Temperature", Units: "degC", Scale: "linear", Min: program.ColdTargetDegC - 12, Max: program.HotTargetDegC + 12, Side: "left", Format: "0.0"},
		{ID: "pressure_mbar", Label: "Pressure", Units: "mbar", Scale: "log10", Min: 0.00000001, Max: 1013.25, Side: "right", Format: "0.000000"},
		{ID: "percent", Label: "Duty", Units: "%", Scale: "linear", Min: 0, Max: 100, Side: "right", Format: "0"},
		{ID: "bus_ms", Label: "Bus latency", Units: "ms", Scale: "linear", Min: 0, Max: 80, Side: "right", Format: "0"},
		{ID: "state", Label: "State", Units: "state", Scale: "step", Min: 0, Max: 5, Side: "left", Format: "0"},
	}
	traces := []contracts.GraphTrace{
		{ID: "trace.command.chamber", Label: "Chamber command", Role: "command", Units: "degC", AxisID: "temperature_c", Source: "thermal_program", Values: trace.command},
		{ID: "trace.ghost.profile", Label: fmt.Sprintf("%d-cycle ghost profile", program.CycleCount), Role: "ghost", Units: "degC", AxisID: "temperature_c", Source: "thermal_program", Values: trace.ghost},
		{ID: "trace.actual.chamber_air", Label: "Chamber air actual", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.actual},
		{ID: "trace.dut_temp_a", Label: "High-dissipation DUT node", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "dut_thermal", Values: trace.zone1},
		{ID: "trace.dut_temp_b", Label: "Vacuum-detached DUT node", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "dut_thermal", Values: trace.zone2},
		{ID: "trace.table_loop", Label: "Interface plate", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.table},
		{ID: "trace.shroud", Label: "Thermal shroud", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.shroud},
		{ID: "trace.acceptance.temperature", Label: "Acceptance band center", Role: "acceptance_band", Units: "degC", AxisID: "temperature_c", Source: "requirements", Values: trace.command},
		{ID: "trace.event.functional", Label: "Functional gate", Role: "event", Units: "state", AxisID: "state", Source: "test_conductor", Values: trace.gates},
		{ID: "trace.interlock.facility", Label: "Interlock review", Role: "interlock", Units: "state", AxisID: "state", Source: "facility_safety", Values: trace.interlocks},
		{ID: "trace.evidence.markers", Label: "Evidence capture", Role: "evidence", Units: "state", AxisID: "state", Source: "evidence_report", Values: trace.evidence},
	}
	if campaignID == "tvac_qualification" {
		traces = append(traces,
			contracts.GraphTrace{ID: "trace.actual.tvac_pressure", Label: "TVac pressure", Role: "actual", Units: "mbar", AxisID: "pressure_mbar", Source: "facility_pressure", Values: trace.pressureMbar},
			contracts.GraphTrace{ID: "trace.shroud_inlet", Label: "Shroud inlet", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.shroudInlet},
			contracts.GraphTrace{ID: "trace.shroud_outlet", Label: "Shroud outlet", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.shroudOutlet},
		)
	}
	hero := contracts.HeroGraphModel{
		ID:         campaignID + "_hero_graph",
		Title:      program.Label,
		Owner:      "gossamer_backend_fixture_generator",
		Provenance: provenance.Source,
		TimeAxis: contracts.GraphTimeAxis{
			Start: start, End: end, Anchor: start, Now: execution.Now, RangeSeconds: int(mustTime(end).Sub(mustTime(start)).Seconds()), Clamp: true, LatestPolicy: "accelerated_fixture_cursor",
		},
		Execution:       &execution,
		Axes:            axes,
		Traces:          traces,
		CompanionGroups: companionGroups(campaignID, axes, trace),
		ThermalDiagram:  buildTestItemThermalDiagram(campaignID),
	}
	for _, cycle := range program.Cycles {
		for _, phase := range cycle.Phases {
			hero.PhaseBands = append(hero.PhaseBands, contracts.GraphBand{ID: phase.ID, Label: phase.Label, Kind: phase.Kind, Start: phase.Start, End: phase.End, CycleIndex: cycle.Index, TargetDegC: phase.TargetDegC})
		}
	}
	for _, window := range program.DwellWindows {
		hero.DwellWindows = append(hero.DwellWindows, contracts.GraphBand{ID: window.ID, Label: window.Label, Kind: window.Kind, Start: window.Start, End: window.End, CycleIndex: window.CycleIndex, TargetDegC: window.TargetDegC, Result: "pass"})
	}
	for _, gate := range program.FunctionalGates {
		hero.Markers = append(hero.Markers, contracts.GraphMarker{ID: gate.ID, Label: gate.Label, Kind: "functional_gate", Role: "event", Timestamp: gate.Timestamp, CycleIndex: gate.CycleIndex, Result: gate.Result, EvidenceRef: gate.EvidenceRef})
	}
	for _, window := range program.InterlockWindows {
		hero.Markers = append(hero.Markers, contracts.GraphMarker{ID: window.ID, Label: window.Label, Kind: "interlock", Role: "interlock", Timestamp: window.Start, Result: window.State, Severity: window.Severity, EvidenceRef: window.EvidenceRef})
	}
	for _, marker := range program.EvidenceMarkers {
		hero.Markers = append(hero.Markers, contracts.GraphMarker{ID: marker.ID, Label: marker.Label, Kind: marker.Kind, Role: "evidence", Timestamp: marker.Timestamp, Result: marker.Result, EvidenceRef: marker.EvidenceRef})
	}
	return hero
}

func buildExecutionState(program *contracts.ThermalProgram, start, end string) contracts.ExecutionState {
	startTime := mustTime(start)
	endTime := mustTime(end)
	now := startTime.Add(time.Duration(float64(endTime.Sub(startTime)) * 0.60)).UTC()
	completedCycles := 0
	currentCycle := 0
	currentPhase := "precheck"
	nextMilestone := ""
	for _, cycle := range program.Cycles {
		cycleEnd := mustTime(cycle.End)
		if !cycleEnd.After(now) {
			completedCycles++
		}
		if !mustTime(cycle.Start).After(now) && now.Before(cycleEnd) {
			currentCycle = cycle.Index
			for _, phase := range cycle.Phases {
				if !mustTime(phase.Start).After(now) && now.Before(mustTime(phase.End)) {
					currentPhase = phase.Kind
					nextMilestone = phase.ID + " complete"
					break
				}
			}
		}
	}
	completedDwell := 0
	dwellContributors := []string{}
	for _, window := range program.DwellWindows {
		if !mustTime(window.End).After(now) {
			completedDwell++
			dwellContributors = append(dwellContributors, window.ID)
		}
	}
	completedGates := 0
	gateContributors := []string{}
	for _, gate := range program.FunctionalGates {
		if !mustTime(gate.Timestamp).After(now) {
			completedGates++
			gateContributors = append(gateContributors, gate.ID)
		}
	}
	cycleContributors := []string{}
	for _, cycle := range program.Cycles {
		if !mustTime(cycle.End).After(now) {
			cycleContributors = append(cycleContributors, cycle.Label)
		}
	}
	return contracts.ExecutionState{
		Mode:             "accelerated_live_replay",
		Now:              now.Format(time.RFC3339),
		PercentComplete:  60,
		Acceleration:     "1 simulated hour per wall-clock minute",
		PastDataPolicy:   "real simulated telemetry up to cursor",
		FutureDataPolicy: "ghost trace and planned evidence only after cursor",
		CompletedCycles:  completedCycles,
		TargetCycles:     program.CycleCount,
		CurrentCycle:     currentCycle,
		CurrentPhase:     currentPhase,
		RequirementProgress: []contracts.RequirementProgress{
			{
				ID: "REQ-CYCLE-COUNT", Label: "Thermal cycles completed", Completed: completedCycles, Target: program.CycleCount,
				Percent: progressPercent(completedCycles, program.CycleCount), State: progressState(completedCycles, program.CycleCount), Contributors: cycleContributors,
				NextMilestone: nextMilestone, EvidenceSource: "thermal cycle phase timestamps",
			},
			{
				ID: "REQ-STABILITY", Label: "Stabilized dwell windows", Completed: completedDwell, Target: program.CycleCount * 2,
				Percent: progressPercent(completedDwell, program.CycleCount*2), State: progressState(completedDwell, program.CycleCount*2), Contributors: dwellContributors,
				NextMilestone: nextMilestone, EvidenceSource: "dwell window stability evidence",
			},
			{
				ID: "REQ-FUNC-GATE", Label: "Functional gates executed", Completed: completedGates, Target: len(program.FunctionalGates),
				Percent: progressPercent(completedGates, len(program.FunctionalGates)), State: progressState(completedGates, len(program.FunctionalGates)), Contributors: gateContributors,
				EvidenceSource: "functional gate event markers",
			},
		},
	}
}

func firstOrderDelta(current, target, tauMin float64, dt time.Duration) float64 {
	if tauMin <= 0 {
		return target - current
	}
	alpha := 1 - math.Exp(-dt.Minutes()/tauMin)
	return (target - current) * clamp(alpha, 0, 1)
}

func advanceComponent(temp, chamberAir, table, shroud, pressure float64, campaignID string, params componentParams, gateActive, payloadActive, survivalMode bool, dt time.Duration) (float64, heatFlux) {
	airScale := airCouplingScale(campaignID, pressure)
	selfHeat := params.baseSelfHeatW
	if survivalMode {
		selfHeat *= 0.18
	}
	if payloadActive {
		selfHeat += params.payloadSelfHeatW
	}
	if gateActive {
		selfHeat += params.gateSelfHeatW
	}
	flux := heatFlux{
		air:    params.airConductanceWPerK * airScale * (chamberAir - temp),
		iface:  params.tableConductanceWPerK * (table - temp),
		shroud: radiativeFlux(temp, shroud, params.radiatingAreaM2, params.emissivity),
		self:   selfHeat,
	}
	next := temp + ((flux.air + flux.iface + flux.shroud + flux.self) / params.capacitanceJPerK * dt.Seconds())
	return clamp(next, -80, 100), flux
}

func airCouplingScale(campaignID string, pressure float64) float64 {
	if campaignID != "tvac_qualification" {
		return 1
	}
	normalizedPressure := clamp(pressure/101325, 0, 1)
	return clamp(math.Pow(normalizedPressure, 0.35), 0.015, 1)
}

func radiativeFlux(nodeDegC, shroudDegC, areaM2, emissivity float64) float64 {
	if areaM2 <= 0 || emissivity <= 0 {
		return 0
	}
	nodeK := kelvin(nodeDegC)
	shroudK := kelvin(shroudDegC)
	meanK := clamp((nodeK+shroudK)/2, 160, 380)
	radiativeConductance := 4 * emissivity * areaM2 * 5.670374419e-8 * math.Pow(meanK, 3)
	flux := radiativeConductance * (shroudK - nodeK)
	return clamp(flux, -45, 45)
}

func kelvin(degC float64) float64 {
	return degC + 273.15
}

func advancePressure(campaignID string, previous, volatilePool float64, elapsed time.Duration, phase string, cycle int, shroudDegC, fastComponentDegC, lazyComponentDegC float64, dt time.Duration) (pressure, nextPool, outgasRate, virtualLeak, roughingRemoval, turboRemoval, totalRemoval float64) {
	if campaignID != "tvac_qualification" {
		return 101325, volatilePool, 0, 0, 0, 0, 0
	}
	if volatilePool <= 0 {
		volatilePool = 0.02
	}
	if phase == "ambient_postcheck" {
		ventRatePerMin := 0.42
		delta := (101325 - previous) * (1 - math.Exp(-ventRatePerMin*dt.Minutes()))
		return math.Min(101325, previous+delta), volatilePool, 0, 0, 0, 0, 0
	}
	h := elapsed.Hours()
	if phase == "ambient_precheck" && h < 2.0 {
		return 101325, volatilePool, 0, 0, 0, 0, 0
	}
	if phase == "ambient_precheck" && previous > 101000 {
		previous = 101325
	}

	// Pressure is modeled as a single effective volume with pump removal, a small
	// virtual leak, and a finite volatile inventory that desorbs faster when hot.
	dtMin := dt.Minutes()
	cryoPump := clamp((-70-shroudDegC)/85, 0, 1)
	roughingRatePerMin := 0.0
	turboRatePerMin := 0.0
	if previous > 120 {
		roughingRatePerMin = 0.44
	}
	if previous < 260 {
		crossover := 1 - clamp((previous-120)/140, 0, 1)
		turboRatePerMin = 0.17 * crossover * (1 + 2.4*cryoPump)
	}
	if previous <= 120 {
		roughingRatePerMin = 0.018
	}
	pumpRatePerMin := roughingRatePerMin + turboRatePerMin
	if pumpRatePerMin <= 0 {
		pumpRatePerMin = 0.12
	}
	virtualLeak = 0.0000051 * (0.35 + 0.65*volatilePool)
	hotNode := math.Max(math.Max(fastComponentDegC, lazyComponentDegC), shroudDegC)
	tempK := kelvin(hotNode)
	referenceK := kelvin(22)
	arrhenius := math.Exp(-3600.0/tempK + 3600.0/referenceK)
	tempFactor := clamp((hotNode+35)/115, 0.02, 1.7)
	hotWave := clamp((hotNode-35)/60, 0, 1)
	cycleMemory := math.Exp(-0.28 * math.Max(0, float64(cycle-1)))
	if phase == "ramp_hot" || phase == "hot_survival" || phase == "hot_operational" {
		outgasRate = (0.00007 + 0.036*hotWave*arrhenius*tempFactor) * volatilePool * cycleMemory
	} else if phase == "ramp_cold" {
		outgasRate = 0.00004 * volatilePool * cycleMemory
	} else if phase == "ambient_postcheck_vacuum" {
		outgasRate = 0.000025 * volatilePool
	} else {
		outgasRate = 0.000012 * volatilePool
	}
	outgasRate *= 1 - 0.55*cryoPump
	ultimatePressure := virtualLeak / pumpRatePerMin
	equilibrium := (virtualLeak + outgasRate) / pumpRatePerMin
	next := equilibrium + (previous-equilibrium)*math.Exp(-pumpRatePerMin*dtMin)
	totalRemoval = math.Max(0, (previous-next)/dtMin+outgasRate+virtualLeak)
	if pumpRatePerMin > 0 {
		roughingRemoval = totalRemoval * roughingRatePerMin / pumpRatePerMin
		turboRemoval = totalRemoval * turboRatePerMin / pumpRatePerMin
	}
	if h > 2.4 {
		next = math.Min(next, previous*0.92+outgasRate*dtMin+virtualLeak*dtMin)
	}
	next = math.Max(ultimatePressure, next)
	depletion := outgasRate * dtMin * 2.6
	nextPool = clamp(volatilePool-depletion, 0.018, 1.0)
	return next, nextPool, outgasRate, virtualLeak, roughingRemoval, turboRemoval, totalRemoval
}

func pumpModeCode(pressurePa float64) float64 {
	switch {
	case pressurePa > 101000:
		return 0
	case pressurePa > 260:
		return 1
	case pressurePa > 120:
		return 2
	default:
		return 3
	}
}

func buildTestItemThermalDiagram(campaignID string) *contracts.TestItemThermalDiagram {
	diagram := &contracts.TestItemThermalDiagram{
		ID:      campaignID + "_test_item_thermal_paths",
		Label:   "Generic test item thermal paths",
		Context: "thermal_chamber",
		Summary: "Chamber air and a fluid-controlled interface drive two representative DUT thermal nodes with different mass, coupling, self-heating, and sensor behavior.",
		Nodes: []contracts.ThermalDiagramNode{
			{ID: "chamber_air", Label: "Chamber air", Kind: "environment", Role: "convective_boundary", Signal: "chamber_air_deg_c", X: 9, Y: 33},
			{ID: "interface_plate", Label: "Fluid interface", Kind: "interface", Role: "conductive_boundary", Signal: "interface_plate_deg_c", X: 28, Y: 69},
			{ID: "test_item", Label: "Test item", Kind: "test_item", Role: "enclosure", X: 50, Y: 48},
			{ID: "fast_node", Label: "High-power node", Kind: "component", Role: "fast_thermal_response", Signal: "dut_fast_component_deg_c", X: 72, Y: 32},
			{ID: "lazy_node", Label: "Isolated node", Kind: "component", Role: "slow_thermal_response", Signal: "dut_lazy_component_deg_c", X: 73, Y: 66},
		},
		Links: []contracts.ThermalDiagramLink{
			{ID: "air_to_fast", Source: "chamber_air", Target: "fast_node", Kind: "convection", Label: "air convection", Strength: 0.58, Signal: "dut_fast_air_flux_w"},
			{ID: "air_to_lazy", Source: "chamber_air", Target: "lazy_node", Kind: "convection", Label: "air convection", Strength: 0.42, Signal: "dut_lazy_air_flux_w"},
			{ID: "interface_to_fast", Source: "interface_plate", Target: "fast_node", Kind: "conduction", Label: "fluid interface conduction", Strength: 0.74, Signal: "dut_fast_interface_flux_w"},
			{ID: "interface_to_lazy", Source: "interface_plate", Target: "lazy_node", Kind: "conduction", Label: "weak interface conduction", Strength: 0.24, Signal: "dut_lazy_interface_flux_w"},
		},
		Notes: []string{
			"High-power node responds quickly and self-heats during functional gates.",
			"Isolated node is deliberately slower, showing delayed stabilization evidence.",
		},
	}
	if campaignID == "tvac_qualification" {
		diagram.Context = "thermal_vacuum"
		diagram.Summary = "Vacuum reduces air coupling, so the thermal shroud becomes the dominant radiative boundary while the platen/interface remains a weaker conductive path for a generic satellite-like test item."
		diagram.Nodes = []contracts.ThermalDiagramNode{
			{ID: "thermal_shroud", Label: "Thermal shroud", Kind: "environment", Role: "radiative_boundary", Signal: "thermal_shroud_deg_c", X: 9, Y: 31},
			{ID: "platen", Label: "Platen", Kind: "interface", Role: "conductive_boundary", Signal: "interface_plate_deg_c", X: 29, Y: 72},
			{ID: "test_item", Label: "Test item", Kind: "test_item", Role: "enclosure", X: 50, Y: 49},
			{ID: "fast_node", Label: "High-power node", Kind: "component", Role: "fast_thermal_response", Signal: "dut_fast_component_deg_c", X: 73, Y: 32},
			{ID: "lazy_node", Label: "Isolated node", Kind: "component", Role: "slow_thermal_response", Signal: "dut_lazy_component_deg_c", X: 74, Y: 66},
			{ID: "pressure", Label: "Vacuum pressure", Kind: "environment", Role: "coupling_modifier", Signal: "tvac_pressure_mbar", X: 24, Y: 19},
		}
		diagram.Links = []contracts.ThermalDiagramLink{
			{ID: "shroud_to_fast", Source: "thermal_shroud", Target: "fast_node", Kind: "radiation", Label: "radiative exchange", Strength: 0.58, Signal: "dut_fast_shroud_flux_w"},
			{ID: "shroud_to_lazy", Source: "thermal_shroud", Target: "lazy_node", Kind: "radiation", Label: "radiative exchange", Strength: 0.82, Signal: "dut_lazy_shroud_flux_w"},
			{ID: "platen_to_fast", Source: "platen", Target: "fast_node", Kind: "conduction", Label: "platen conduction", Strength: 0.46, Signal: "dut_fast_interface_flux_w"},
			{ID: "platen_to_lazy", Source: "platen", Target: "lazy_node", Kind: "conduction", Label: "weak platen conduction", Strength: 0.16, Signal: "dut_lazy_interface_flux_w"},
			{ID: "pressure_air_coupling", Source: "pressure", Target: "test_item", Kind: "modifier", Label: "low pressure suppresses air coupling", Strength: 0.35, Signal: "tvac_pressure_mbar"},
		}
		diagram.Notes = []string{
			"Air convection collapses during pumpdown, making shroud radiation visible in the slow node.",
			"The platen/interface is present but intentionally weaker for this generic satellite-like configuration.",
		}
	}
	return diagram
}

func companionGroups(campaignID string, axes []contracts.GraphYAxis, trace sampleTrace) []contracts.CompanionGraphGroup {
	temperatureAxis := axes[0]
	percentAxis := axes[2]
	busAxis := axes[3]
	stateAxis := axes[4]
	actuationLabel := "Cooling actuator"
	if campaignID == "tvac_qualification" {
		actuationLabel = "LN2 valve"
	}
	stateTraces := []contracts.GraphTrace{
		{ID: "trace.phase_enum", Label: "Thermal phase", Role: "event", Units: "state", AxisID: "state", Source: "thermal_program", Values: trace.phase},
		{ID: "trace.functional_gate_active", Label: "Functional gate", Role: "event", Units: "bool", AxisID: "state", Source: "test_conductor", Values: trace.gates},
		{ID: "trace.stability_reached", Label: "Stability reached", Role: "state", Units: "bool", AxisID: "state", Source: "thermal_program", Values: trace.stability},
		{ID: "trace.dwell_active", Label: "Dwell active", Role: "state", Units: "bool", AxisID: "state", Source: "thermal_program", Values: trace.dwellActive},
		{ID: "trace.dwell_complete", Label: "Dwell complete", Role: "state", Units: "bool", AxisID: "state", Source: "thermal_program", Values: trace.dwellComplete},
		{ID: "trace.interlock_review", Label: "Interlock review", Role: "interlock", Units: "bool", AxisID: "state", Source: "facility_safety", Values: trace.interlocks},
		{ID: "trace.source_degraded", Label: "Source degraded", Role: "source_quality", Units: "bool", AxisID: "state", Source: "demo_quality", Values: trace.degraded},
		{ID: "trace.evidence_capture", Label: "Evidence capture", Role: "evidence", Units: "bool", AxisID: "state", Source: "evidence_report", Values: trace.evidence},
		{ID: "trace.dut_ready", Label: "DUT ready", Role: "state", Units: "bool", AxisID: "state", Source: "dut_control", Values: trace.ready},
		{ID: "trace.dut_operative", Label: "DUT operative", Role: "state", Units: "bool", AxisID: "state", Source: "dut_control", Values: trace.operative},
		{ID: "trace.payload_active", Label: "Payload active", Role: "state", Units: "bool", AxisID: "state", Source: "dut_power", Values: trace.payload},
		{ID: "trace.rf_link_locked", Label: "RF link locked", Role: "state", Units: "bool", AxisID: "state", Source: "dut_link", Values: trace.rfLocked},
		{ID: "trace.fault_flag", Label: "Fault flag", Role: "interlock", Units: "bool", AxisID: "state", Source: "demo_quality", Values: trace.fault},
	}
	if campaignID == "tvac_qualification" {
		stateTraces = append(stateTraces[:5], append([]contracts.GraphTrace{{ID: "trace.pressure_gate", Label: "Pressure gate", Role: "state", Units: "bool", AxisID: "state", Source: "facility_pressure", Values: trace.pressureGate}}, stateTraces[5:]...)...)
		stateTraces = append(stateTraces[:6], append([]contracts.GraphTrace{{ID: "trace.pump_mode", Label: "Pump mode", Role: "state", Units: "enum", AxisID: "state", Source: "facility_pressure", Values: trace.pumpMode}}, stateTraces[6:]...)...)
		stateTraces = append(stateTraces[:7], append([]contracts.GraphTrace{{ID: "trace.exhaust_duct_safe", Label: "Exhaust duct safe", Role: "state", Units: "bool", AxisID: "state", Source: "facility_infrastructure", Values: trace.exhaustSafe}}, stateTraces[7:]...)...)
	}
	groups := []contracts.CompanionGraphGroup{
		{
			ID:    "dut_temperature_response",
			Label: "DUT temperature response",
			Axes:  []contracts.GraphYAxis{temperatureAxis},
			Traces: []contracts.GraphTrace{
				{ID: "trace.context.chamber_air", Label: "Chamber air", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.actual},
				{ID: "trace.table_loop", Label: "Interface plate", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.table},
				{ID: "trace.shroud", Label: "Thermal shroud", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.shroud},
				{ID: "trace.shroud_inlet", Label: "Shroud inlet", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.shroudInlet},
				{ID: "trace.shroud_outlet", Label: "Shroud outlet", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.shroudOutlet},
				{ID: "trace.dut_temp_a", Label: "High-dissipation DUT node", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "dut_thermal", Values: trace.zone1},
				{ID: "trace.dut_temp_b", Label: "Vacuum-detached DUT node", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "dut_thermal", Values: trace.zone2},
				{ID: "trace.dut_gradient", Label: "Internal gradient", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "dut_thermal", Values: trace.gradient},
				{ID: "trace.shroud_gradient", Label: "Shroud gradient", Role: "source_quality", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.shroudDelta},
			},
		},
		{
			ID:    "dut_heat_flux_paths",
			Label: "DUT heat-flux paths",
			Axes:  []contracts.GraphYAxis{{ID: "heat_flux_w", Label: "Heat flux", Units: "W", Scale: "linear", Min: -35, Max: 35, Side: "left", Format: "0.0"}},
			Traces: []contracts.GraphTrace{
				{ID: "trace.fast_air_flux", Label: "Fast node air flux", Role: "actual", Units: "W", AxisID: "heat_flux_w", Source: "dut_thermal", Values: trace.fastAirFlux},
				{ID: "trace.fast_interface_flux", Label: "Fast node interface flux", Role: "actual", Units: "W", AxisID: "heat_flux_w", Source: "dut_thermal", Values: trace.fastIFace},
				{ID: "trace.lazy_air_flux", Label: "Lazy node air flux", Role: "actual", Units: "W", AxisID: "heat_flux_w", Source: "dut_thermal", Values: trace.lazyAirFlux},
				{ID: "trace.lazy_interface_flux", Label: "Lazy node interface flux", Role: "actual", Units: "W", AxisID: "heat_flux_w", Source: "dut_thermal", Values: trace.lazyIFace},
				{ID: "trace.lazy_shroud_flux", Label: "Lazy node shroud flux", Role: "actual", Units: "W", AxisID: "heat_flux_w", Source: "dut_thermal", Values: trace.lazyShroud},
				{ID: "trace.dut_self_heat", Label: "DUT self heat", Role: "actual", Units: "W", AxisID: "heat_flux_w", Source: "dut_power", Values: trace.selfHeat},
			},
		},
		{
			ID:    "facility_actuation",
			Label: "Facility actuation",
			Axes:  []contracts.GraphYAxis{percentAxis},
			Traces: []contracts.GraphTrace{
				{ID: "trace.ln2_duty", Label: actuationLabel, Role: "actual", Units: "%", AxisID: "percent", Source: "facility_thermal", Values: trace.ln2},
			},
		},
		{
			ID:    "building_infrastructure",
			Label: "Building infrastructure",
			Axes:  []contracts.GraphYAxis{temperatureAxis, contracts.GraphYAxis{ID: "pressure_bar", Label: "Pressure", Units: "bar", Scale: "linear", Min: 5.8, Max: 6.5, Side: "right", Format: "0.00"}},
			Traces: []contracts.GraphTrace{
				{ID: "trace.cooling_water_temp", Label: "Cooling water temp", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_infrastructure", Values: trace.coolingWater},
				{ID: "trace.pressurized_air_supply", Label: "Pressurized air supply", Role: "actual", Units: "bar", AxisID: "pressure_bar", Source: "facility_infrastructure", Values: trace.airSupply},
				{ID: "trace.air_dewpoint", Label: "Air dew point", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_infrastructure", Values: trace.airDewpoint},
			},
		},
		{
			ID:    "dut_power_response",
			Label: "DUT power budgets",
			Axes:  []contracts.GraphYAxis{{ID: "power_w", Label: "Power", Units: "W", Scale: "linear", Min: 0, Max: 260, Side: "left", Format: "0"}},
			Traces: []contracts.GraphTrace{
				{ID: "trace.power_total", Label: "Total power", Role: "actual", Units: "W", AxisID: "power_w", Source: "dut_power", Values: trace.powerTotal},
				{ID: "trace.power_subsystem", Label: "Subsystem budget", Role: "actual", Units: "W", AxisID: "power_w", Source: "dut_power", Values: trace.powerSubsys},
				{ID: "trace.power_payload", Label: "Payload/FT load", Role: "actual", Units: "W", AxisID: "power_w", Source: "dut_power", Values: trace.powerPayload},
				{ID: "trace.power_avionics", Label: "Avionics", Role: "actual", Units: "W", AxisID: "power_w", Source: "dut_power", Values: trace.powerAvionics},
				{ID: "trace.power_link", Label: "Link subsystem", Role: "actual", Units: "W", AxisID: "power_w", Source: "dut_power", Values: trace.powerLink},
				{ID: "trace.power_thermal_control", Label: "Thermal control", Role: "actual", Units: "W", AxisID: "power_w", Source: "dut_power", Values: trace.powerThermal},
			},
		},
		{
			ID:    "tmtc_bus_response",
			Label: "TM/TC bus response",
			Axes:  []contracts.GraphYAxis{busAxis},
			Traces: []contracts.GraphTrace{
				{ID: "trace.bus_latency", Label: "Bus latency", Role: "source_quality", Units: "ms", AxisID: "bus_ms", Source: "demo_bus_virtualization", Values: trace.busLatency},
				{ID: "trace.source_freshness", Label: "Freshness", Role: "source_quality", Units: "ms", AxisID: "bus_ms", Source: "demo_quality", Values: trace.quality},
			},
		},
		{
			ID:    "tmtc_counter_response",
			Label: "TM/TC counters",
			Axes:  []contracts.GraphYAxis{{ID: "counter", Label: "Counter", Units: "count", Scale: "linear", Min: 0, Max: 9000, Side: "left", Format: "0"}},
			Traces: []contracts.GraphTrace{
				{ID: "trace.overall_packet_counter", Label: "Overall packet counter", Role: "counter", Units: "count", AxisID: "counter", Source: "demo_bus_virtualization", Values: trace.overall},
				{ID: "trace.tm_packet_counter", Label: "TM packet counter", Role: "counter", Units: "count", AxisID: "counter", Source: "demo_bus_virtualization", Values: trace.tmCounter},
				{ID: "trace.tc_packet_counter", Label: "TC packet counter", Role: "counter", Units: "count", AxisID: "counter", Source: "demo_bus_virtualization", Values: trace.tcCounter},
				{ID: "trace.dropped_frame_count", Label: "Dropped frames", Role: "counter", Units: "count", AxisID: "counter", Source: "demo_bus_virtualization", Values: trace.dropCount},
			},
		},
		{
			ID:     "state_change_swimlane",
			Label:  "State changes and flags",
			Axes:   []contracts.GraphYAxis{stateAxis},
			Traces: stateTraces,
		},
	}
	if campaignID == "tvac_qualification" {
		pressureAxis := axes[1]
		groups = append([]contracts.CompanionGraphGroup{
			{
				ID:    "tvac_pressure_response",
				Label: "TVac pressure",
				Axes:  []contracts.GraphYAxis{pressureAxis},
				Traces: []contracts.GraphTrace{
					{ID: "trace.tvac_pressure", Label: "Pressure", Role: "actual", Units: "mbar", AxisID: "pressure_mbar", Source: "facility_pressure", Values: trace.pressureMbar},
				},
			},
			{
				ID:    "tvac_pressure_sources",
				Label: "Pump, leak, and outgassing balance",
				Axes: []contracts.GraphYAxis{
					{ID: "pressure_rate", Label: "Pressure rate", Units: "mbar/min", Scale: "log10", Min: 0.00000001, Max: 1000, Side: "left", Format: "0.000000"},
					{ID: "percent", Label: "Inventory", Units: "%", Scale: "linear", Min: 0, Max: 100, Side: "right", Format: "0"},
				},
				Traces: []contracts.GraphTrace{
					{ID: "trace.tvac_outgassing", Label: "Temperature outgassing", Role: "actual", Units: "mbar/min", AxisID: "pressure_rate", Source: "facility_pressure", Values: trace.outgasMbar},
					{ID: "trace.tvac_virtual_leak", Label: "Virtual leak", Role: "acceptance_band", Units: "mbar/min", AxisID: "pressure_rate", Source: "facility_pressure", Values: trace.virtualLeak},
					{ID: "trace.tvac_roughing_pump", Label: "Roughing pump", Role: "source_quality", Units: "mbar/min", AxisID: "pressure_rate", Source: "facility_pressure", Values: trace.roughingRate},
					{ID: "trace.tvac_turbo_pump", Label: "Turbo pump", Role: "actual", Units: "mbar/min", AxisID: "pressure_rate", Source: "facility_pressure", Values: trace.turboRate},
					{ID: "trace.tvac_pump_removal", Label: "Pump removal", Role: "source_quality", Units: "mbar/min", AxisID: "pressure_rate", Source: "facility_pressure", Values: trace.pumpRemoval},
					{ID: "trace.tvac_volatile_inventory", Label: "Volatile inventory", Role: "ghost", Units: "%", AxisID: "percent", Source: "facility_pressure", Values: trace.volatilePool},
				},
			},
			{
				ID:    "tvac_exhaust_scavenger",
				Label: "Exhaust cold scavenger",
				Axes:  []contracts.GraphYAxis{temperatureAxis, percentAxis},
				Traces: []contracts.GraphTrace{
					{ID: "trace.tvac_cryo_exhaust", Label: "Cryogenic exhaust", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_thermal", Values: trace.cryoExhaust},
					{ID: "trace.tvac_scavenged_exhaust", Label: "After water scavenger", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_infrastructure", Values: trace.scavExhaust},
					{ID: "trace.tvac_scavenger_water_return", Label: "Scavenger water return", Role: "actual", Units: "degC", AxisID: "temperature_c", Source: "facility_infrastructure", Values: trace.scavWater},
					{ID: "trace.tvac_exhaust_cold_recovery", Label: "Cold recovery", Role: "source_quality", Units: "%", AxisID: "percent", Source: "facility_infrastructure", Values: trace.coldRecovery},
				},
			},
		}, groups...)
		groups = append(groups, contracts.CompanionGraphGroup{
			ID:    "facility_temperature_safety",
			Label: "Heat-exchanger freeze margin",
			Axes:  []contracts.GraphYAxis{temperatureAxis},
			Traces: []contracts.GraphTrace{
				{ID: "trace.freeze_margin", Label: "Water scavenger freeze margin", Role: "interlock", Units: "degC", AxisID: "temperature_c", Source: "facility_infrastructure", Values: trace.freeze},
			},
		})
	}
	return groups
}

func gateFor(program *contracts.ThermalProgram, phaseID string, t time.Time) (string, float64) {
	for _, gate := range program.FunctionalGates {
		if gate.PhaseID != phaseID {
			continue
		}
		gateTime := mustTime(gate.Timestamp)
		if math.Abs(t.Sub(gateTime).Minutes()) <= 15 {
			switch gate.Gate {
			case "cold":
				return "cold", 2
			case "hot":
				return "hot", 3
			case "pre":
				return "pre", 1
			case "post":
				return "post", 4
			default:
				return gate.Gate, 1
			}
		}
	}
	return "none", 0
}

func thermalPhaseCode(phase string) float64 {
	switch phase {
	case "ambient_precheck":
		return 0
	case "ramp_cold":
		return 1
	case "cold_operational":
		return 2
	case "ramp_hot":
		return 3
	case "hot_operational":
		return 4
	case "hot_survival":
		return 5
	case "cold_survival":
		return 6
	case "ambient_postcheck_vacuum":
		return 7
	case "ambient_postcheck":
		return 8
	default:
		return 0
	}
}

func isSurvivalPhase(phase string) bool {
	return phase == "hot_survival" || phase == "cold_survival"
}

func isOperationalDwellPhase(phase string) bool {
	return phase == "hot_operational" || phase == "cold_operational"
}

func isThermalDwellPhase(phase string) bool {
	return isSurvivalPhase(phase) || isOperationalDwellPhase(phase)
}

func dwellStateFor(program *contracts.ThermalProgram, phaseID string, t time.Time) (stable bool, active bool, complete bool) {
	for _, dwell := range program.DwellWindows {
		if strings.TrimSuffix(dwell.ID, "-WINDOW") != phaseID {
			continue
		}
		start := mustTime(dwell.Start)
		end := mustTime(dwell.End)
		if !t.Before(start) {
			stable = true
		}
		if !t.Before(start) && t.Before(end) {
			active = true
		}
		if !t.Before(end) {
			complete = true
		}
		return stable, active, complete
	}
	return false, false, false
}

func pressureGateState(campaignID, phase string, pressurePa float64) bool {
	if campaignID != "tvac_qualification" {
		return true
	}
	if phase == "ambient_precheck" || phase == "ambient_postcheck_vacuum" {
		return pressurePa <= 0.05
	}
	return phase != "ambient_postcheck" && pressurePa <= 0.1
}

func dwellStateName(active, complete bool) string {
	switch {
	case complete:
		return "complete"
	case active:
		return "dwelling"
	default:
		return "waiting"
	}
}

func thermalContextDuration(program *contracts.ThermalProgram) time.Duration {
	if program == nil || len(program.Cycles) == 0 {
		return 90 * time.Minute
	}
	start := mustTime(program.Cycles[0].Start)
	end := mustTime(program.Cycles[len(program.Cycles)-1].End)
	duration := end.Sub(start)
	context := duration / 10
	if context < 90*time.Minute {
		return 90 * time.Minute
	}
	return context
}

func smoothRamp(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f * f * (3 - 2*f)
}

func acceptanceTarget(command float64, program *contracts.ThermalProgram) float64 {
	if math.Abs(command-program.HotTargetDegC) < math.Abs(command-program.ColdTargetDegC) {
		return program.HotTargetDegC
	}
	return program.ColdTargetDegC
}

func functionalHeat(active bool) float64 {
	if active {
		return 18
	}
	return 0
}

func boolValue(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func stateName(active bool, trueName, falseName string) string {
	if active {
		return trueName
	}
	return falseName
}

func progressPercent(completed, target int) float64 {
	if target <= 0 {
		return 100
	}
	return round((float64(completed) / float64(target)) * 100)
}

func progressState(completed, target int) string {
	if target <= 0 || completed >= target {
		return "complete"
	}
	if completed > 0 {
		return "in_progress"
	}
	return "pending"
}

func point(ts string, value float64) contracts.GraphPoint {
	return contracts.GraphPoint{Timestamp: ts, Value: round(value)}
}

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func noise(rng *rand.Rand, sigma float64) float64 {
	return rng.NormFloat64() * sigma
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func round(value float64) float64 {
	abs := math.Abs(value)
	switch {
	case abs == 0:
		return 0
	case abs < 0.000001:
		return math.Round(value*1e12) / 1e12
	case abs < 0.001:
		return math.Round(value*1e9) / 1e9
	case abs < 1:
		return math.Round(value*1e6) / 1e6
	case abs < 1000:
		return math.Round(value*1000) / 1000
	default:
		return math.Round(value*100) / 100
	}
}
