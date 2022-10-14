package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
	"strconv"
)

type ChassisCollector struct {
	client *gofish.APIClient
}

func (c *ChassisCollector) Collect(ch chan<- prometheus.Metric) error {
	chassiss, err := c.client.Service.Chassis()
	if err != nil {
		return fmt.Errorf("error collecting /Chassis: %s", err)
	}

	for _, chassis := range chassiss {
		c.processChassis(ch, chassis)

		thermal, err := chassis.Thermal()
		if err != nil {
			return fmt.Errorf("error collecting /Chassis/%s/Thermal: %s", chassis.ID, err)
		} else if thermal != nil {
			c.processThermal(ch, thermal, chassis.ID)

			for _, fan := range thermal.Fans {
				c.processFan(ch, fan, chassis.ID)
			}

			for _, t := range thermal.Temperatures {
				c.processTemperature(ch, t, chassis.ID)
			}
		}

		power, err := chassis.Power()
		if err != nil {
			return fmt.Errorf("error collecting /Chassis/%s/Power: %s", chassis.ID, err)
		} else if power != nil {
			for _, control := range power.PowerControl {
				c.processPowerControl(ch, control, chassis.ID)
			}

			for _, ps := range power.PowerSupplies {
				c.processPowerSupply(ch, ps, chassis.ID)
			}

			for _, voltage := range power.Voltages {
				c.processVoltage(ch, voltage, chassis.ID)
			}
		}

		adapters, err := chassis.NetworkAdapters()
		if err != nil {
			return fmt.Errorf("error collecting /Chassis/%s/NetworkAdapters: %s", chassis.ID, err)
		}

		for _, adapter := range adapters {
			c.processNetworkAdapter(ch, adapter, chassis.ID)

			ports, err := adapter.NetworkPorts()
			if err != nil {
				return fmt.Errorf("error collecting /Chassis/%s/NetworkAdapters/%s/NetworkPorts: %s", chassis.ID, adapter.ID, err)
			}

			for _, port := range ports {
				c.processNetworkPort(ch, port, chassis.ID)
			}
		}
	}

	return nil
}

func (c *ChassisCollector) processChassis(ch chan<- prometheus.Metric, chassis *redfish.Chassis) {
	constLabels := prometheus.Labels{"id": chassis.ID, "name": chassis.Name, "chassis_id": chassis.ID, "chassis_type": string(chassis.ChassisType)}

	intrusionSensorDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "intrusion_sensor"),
		"Intrusion sensor reading; 0: Normal, 1: HardwareIntrusion, 2: TamperingDetected",
		[]string{"sensor_number", "sensor_re_arm"}, constLabels,
	)
	powerStateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_state"),
		"Chassis power state; 0: Off, 1: On, 2: PoweringOn, 3: PoweringOff",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "health"),
		"Chassis health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "state"),
		"Chassis state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	if e := enumIntrusionSensor(chassis.PhysicalSecurity.IntrusionSensor); e >= 0 {
		ch <- prometheus.MustNewConstMetric(intrusionSensorDesc, prometheus.GaugeValue, e, strconv.Itoa(chassis.PhysicalSecurity.IntrusionSensorNumber), string(chassis.PhysicalSecurity.IntrusionSensorReArm))
	}
	if e := enumPowerState(chassis.PowerState); e >= 0 {
		ch <- prometheus.MustNewConstMetric(powerStateDesc, prometheus.GaugeValue, e)
	}

	if e := enumHealth(chassis.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(chassis.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *ChassisCollector) processThermal(ch chan<- prometheus.Metric, thermal *redfish.Thermal, chassisID string) {
	constLabels := prometheus.Labels{"id": thermal.ID, "name": thermal.Name, "chassis_id": chassisID}

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "thermal_health"),
		"Thermal health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "thermal_state"),
		"Thermal state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	if e := enumHealth(thermal.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(thermal.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *ChassisCollector) processFan(ch chan<- prometheus.Metric, fan redfish.Fan, chassisID string) {
	constLabels := prometheus.Labels{"id": fan.MemberID, "name": fan.Name, "chassis_id": chassisID, "sensor_number": strconv.Itoa(fan.SensorNumber), "physical_context": fan.PhysicalContext}

	readingRPMDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "fan_speed_rpm"),
		"Fan speed, RPM",
		nil, constLabels,
	)
	readingPercentDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "fan_speed_ratio"),
		"Fan speed, %",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "fan_health"),
		"Fan health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "fan_state"),
		"Fan state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	switch fan.ReadingUnits {
	case redfish.RPMReadingUnits:
		ch <- prometheus.MustNewConstMetric(readingRPMDesc, prometheus.GaugeValue, float64(fan.Reading))
	case redfish.PercentReadingUnits:
		ch <- prometheus.MustNewConstMetric(readingPercentDesc, prometheus.GaugeValue, float64(fan.Reading)/100)
	}

	if e := enumHealth(fan.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(fan.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *ChassisCollector) processTemperature(ch chan<- prometheus.Metric, t redfish.Temperature, chassisID string) {
	constLabels := prometheus.Labels{"id": t.MemberID, "name": t.Name, "chassis_id": chassisID, "sensor_number": strconv.Itoa(t.SensorNumber), "physical_context": t.PhysicalContext}

	readingDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "temperature_celsius"),
		"Temperature sensor reading, °C",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "temperature_health"),
		"Temperature sensor health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "temperature_state"),
		"Temperature sensor state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(readingDesc, prometheus.GaugeValue, float64(t.ReadingCelsius))

	if e := enumHealth(t.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(t.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *ChassisCollector) processPowerControl(ch chan<- prometheus.Metric, control redfish.PowerControl, chassisID string) {
	constLabels := prometheus.Labels{"id": control.MemberID, "name": control.Name, "chassis_id": chassisID}

	powerAllocatedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_control_power_allocated_watts"),
		"Power allocated to chassis resources, W",
		nil, constLabels,
	)
	powerCapacityDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_control_power_capacity_watts"),
		"Power available for allocation to chassis resources, W",
		nil, constLabels,
	)
	powerConsumedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_control_power_consumed_watts"),
		"Power consumed by the chassis resources, W",
		nil, constLabels,
	)
	powerLimitDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_control_power_limit_watts"),
		"Configured power limit for the chassis resources, W",
		[]string{"correction_interval", "action"}, constLabels,
	)
	powerRequestedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_control_power_requested_watts"),
		"Power requested by the chassis resources, W",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_control_health"),
		"Power control health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_control_state"),
		"Power control state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(powerAllocatedDesc, prometheus.GaugeValue, float64(control.PowerAllocatedWatts))
	ch <- prometheus.MustNewConstMetric(powerCapacityDesc, prometheus.GaugeValue, float64(control.PowerCapacityWatts))
	ch <- prometheus.MustNewConstMetric(powerConsumedDesc, prometheus.GaugeValue, float64(control.PowerConsumedWatts))
	ch <- prometheus.MustNewConstMetric(powerLimitDesc, prometheus.GaugeValue, float64(control.PowerLimit.LimitInWatts), strconv.FormatInt(control.PowerLimit.CorrectionInMs/1000, 10), string(control.PowerLimit.LimitException))
	ch <- prometheus.MustNewConstMetric(powerRequestedDesc, prometheus.GaugeValue, float64(control.PowerRequestedWatts))

	if e := enumHealth(control.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(control.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *ChassisCollector) processPowerSupply(ch chan<- prometheus.Metric, ps redfish.PowerSupply, chassisID string) {
	constLabels := prometheus.Labels{"id": ps.MemberID, "name": ps.Name, "chassis_id": chassisID, "power_supply_type": string(ps.PowerSupplyType)}

	efficiencyDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_supply_efficiency_ratio"),
		"Power supply measured efficiency, %",
		nil, constLabels,
	)
	inputVoltageDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_supply_input_voltage_volts"),
		"Power supply measured input voltage, V",
		[]string{"input_voltage_type"}, constLabels,
	)
	powerCapacityDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_supply_capacity_watts"),
		"Power supply maximum capacity, W",
		nil, constLabels,
	)
	powerInputDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_supply_input_power_watts"),
		"Power supply measured input power, W",
		nil, constLabels,
	)
	powerOutputDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_supply_output_power_watts"),
		"Power supply measured output power, W",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_supply_health"),
		"Power supply health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "power_supply_state"),
		"Power supply state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(efficiencyDesc, prometheus.GaugeValue, float64(ps.EfficiencyPercent)/100)
	ch <- prometheus.MustNewConstMetric(inputVoltageDesc, prometheus.GaugeValue, float64(ps.LineInputVoltage), string(ps.LineInputVoltageType))
	ch <- prometheus.MustNewConstMetric(powerCapacityDesc, prometheus.GaugeValue, float64(ps.PowerCapacityWatts))
	ch <- prometheus.MustNewConstMetric(powerInputDesc, prometheus.GaugeValue, float64(ps.PowerInputWatts))
	ch <- prometheus.MustNewConstMetric(powerOutputDesc, prometheus.GaugeValue, float64(ps.PowerOutputWatts))

	if e := enumHealth(ps.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(ps.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}

}

func (c *ChassisCollector) processVoltage(ch chan<- prometheus.Metric, voltage redfish.Voltage, chassisID string) {
	constLabels := prometheus.Labels{"id": voltage.MemberID, "name": voltage.Name, "chassis_id": chassisID, "sensor_number": strconv.Itoa(voltage.SensorNumber), "physical_context": voltage.PhysicalContext}

	readingDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "voltage_reading_volts"),
		"Voltage sensor reading, V",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "voltage_health"),
		"Voltage sensor health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "voltage_state"),
		"Voltage sensor state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(readingDesc, prometheus.GaugeValue, float64(voltage.ReadingVolts))

	if e := enumHealth(voltage.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(voltage.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *ChassisCollector) processNetworkAdapter(ch chan<- prometheus.Metric, adapter *redfish.NetworkAdapter, chassisID string) {
	constLabels := prometheus.Labels{"id": adapter.ID, "name": adapter.Name, "chassis_id": chassisID}

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "network_adapter_health"),
		"Network adapter health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "network_adapter_state"),
		"Network adapter state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	if e := enumHealth(adapter.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(adapter.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *ChassisCollector) processNetworkPort(ch chan<- prometheus.Metric, port *redfish.NetworkPort, chassisID string) {
	constLabels := prometheus.Labels{"id": port.ID, "name": port.Name, "chassis_id": chassisID, "link_type": string(port.ActiveLinkTechnology)}

	speedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "network_port_speed_bytes"),
		"Network port speed, bytes/s",
		nil, constLabels,
	)
	statusDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "network_port_status"),
		"Network port status; 0: Down, 1: Up",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "network_port_health"),
		"Network port health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "chassis", "network_port_state"),
		"Network port state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(speedDesc, prometheus.GaugeValue, float64(port.CurrentLinkSpeedMbps)*mega/8)

	if e := enumPortLinkStatus(port.LinkStatus); e >= 0 {
		ch <- prometheus.MustNewConstMetric(statusDesc, prometheus.GaugeValue, e)
	}

	if e := enumHealth(port.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(port.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func enumIntrusionSensor(e redfish.IntrusionSensor) float64 {
	switch e {
	case redfish.NormalIntrusionSensor:
		return 0
	case redfish.HardwareIntrusionIntrusionSensor:
		return 1
	case redfish.TamperingDetectedIntrusionSensor:
		return 2
	default:
		return -1
	}
}

func enumPortLinkStatus(e redfish.PortLinkStatus) float64 {
	switch e {
	case redfish.DownPortLinkStatus:
		return 0
	case redfish.UpPortLinkStatus:
		return 1
	default:
		return -1
	}
}
