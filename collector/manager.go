package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
	"net"
)

type ManagerCollector struct {
	client *gofish.APIClient
}

func (c *ManagerCollector) Collect(ch chan<- prometheus.Metric) error {
	managers, err := c.client.Service.Managers()
	if err != nil {
		return fmt.Errorf("error collecting /Managers: %s", err)
	}

	for _, manager := range managers {
		c.processManager(ch, manager)

		ethernetInterfaces, err := manager.EthernetInterfaces()
		if err != nil {
			return fmt.Errorf("error collecting /Managers/%s/EthernetInterfaces: %s", manager.ID, err)
		}

		for _, intf := range ethernetInterfaces {
			c.processEthernetInterface(ch, intf, manager.ID)
		}
	}

	return nil
}

func (c *ManagerCollector) processManager(ch chan<- prometheus.Metric, manager *redfish.Manager) {
	constLabels := prometheus.Labels{"id": manager.ID, "name": manager.Name, "manager_id": manager.ID, "manager_type": string(manager.ManagerType)}

	commandShellDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "manager", "command_shell_status"),
		"Command shell status; 0: Disabled, 1: Enabled",
		nil, constLabels,
	)
	graphicalConsoleDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "manager", "console_graphical_status"),
		"Graphical console status; 0: Disabled, 1: Enabled",
		nil, constLabels,
	)
	serialConsoleDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "manager", "console_serial_status"),
		"Serial console status; 0: Disabled, 1: Enabled",
		nil, constLabels,
	)
	powerStateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "manager", "power_state"),
		"Manager power state; 0: Off, 1: On, 2: PoweringOn, 3: PoweringOff",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(commandShellDesc, prometheus.GaugeValue, btof(manager.CommandShell.ServiceEnabled))
	ch <- prometheus.MustNewConstMetric(graphicalConsoleDesc, prometheus.GaugeValue, btof(manager.GraphicalConsole.ServiceEnabled))
	ch <- prometheus.MustNewConstMetric(serialConsoleDesc, prometheus.GaugeValue, btof(manager.SerialConsole.ServiceEnabled))

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "manager", "health"),
		"Manager health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "manager", "state"),
		"Manager state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	if e := enumPowerState(manager.PowerState); e >= 0 {
		ch <- prometheus.MustNewConstMetric(powerStateDesc, prometheus.GaugeValue, e)
	}

	if e := enumHealth(manager.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(manager.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *ManagerCollector) processEthernetInterface(ch chan<- prometheus.Metric, intf *redfish.EthernetInterface, managerID string) {
	address, _ := net.ParseMAC(intf.MACAddress)

	constLabels := prometheus.Labels{"id": intf.ID, "name": intf.Name, "manager_id": managerID, "interface_type": string(intf.EthernetInterfaceType), "address": address.String()}

	enabledDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "manager", "ethernet_interface_status"),
		"Ethernet interface status; 0: Disabled, 1: Enabled",
		nil, constLabels,
	)
	speedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "manager", "ethernet_interface_speed_bytes"),
		"Ethernet interface speed, bytes/s",
		[]string{"duplex"}, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "ethernet_interface_health"),
		"Ethernet interface health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "ethernet_interface_state"),
		"Ethernet interface state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(enabledDesc, prometheus.GaugeValue, btof(intf.InterfaceEnabled))
	ch <- prometheus.MustNewConstMetric(speedDesc, prometheus.GaugeValue, float64(intf.SpeedMbps)*mebi/8, map[bool]string{true: "full", false: "half"}[intf.FullDuplex])

	if e := enumHealth(intf.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(intf.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}
