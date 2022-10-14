package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
	"net"
)

type SystemCollector struct {
	client *gofish.APIClient
}

func (c *SystemCollector) Collect(ch chan<- prometheus.Metric) error {
	systems, err := c.client.Service.Systems()
	if err != nil {
		return fmt.Errorf("error collecting /Systems: %s", err)
	}

	for _, system := range systems {
		c.processSystem(ch, system)

		ethernetInterfaces, err := system.EthernetInterfaces()
		if err != nil {
			return fmt.Errorf("error collecting /Systems/%s/EthernetInterfaces: %s", system.ID, err)
		}

		for _, intf := range ethernetInterfaces {
			c.processEthernetInterface(ch, intf, system.ID)
		}

		memories, err := system.Memory()
		if err != nil {
			return fmt.Errorf("error collecting /Systems/%s/Memory: %s", system.ID, err)
		}

		for _, memory := range memories {
			c.processMemory(ch, memory, system.ID)
		}

		networkInterfaces, err := system.NetworkInterfaces()
		if err != nil {
			return fmt.Errorf("error collecting /Systems/%s/NetworkInterfaces: %s", system.ID, err)
		}

		for _, intf := range networkInterfaces {
			c.processNetworkInterface(ch, intf, system.ID)
		}

		pcieDevices, err := system.PCIeDevices()
		if err != nil {
			return fmt.Errorf("error collecting /Systems/%s/PCIeDevices: %s", system.ID, err)
		}

		devices := make(map[string]bool)
		for _, device := range pcieDevices {
			if _, processed := devices[device.ID]; !processed {
				c.processPCIeDevice(ch, device, system.ID)
				devices[device.ID] = true
			}
		}

		processors, err := system.Processors()
		if err != nil {
			return fmt.Errorf("error collecting /Systems/%s/Processors: %s", system.ID, err)
		}

		for _, processor := range processors {
			c.processProcessor(ch, processor, system.ID)
		}

		storages, err := system.Storage()
		if err != nil {
			return fmt.Errorf("error collecting /Systems/%s/Storage: %s", system.ID, err)
		}

		for _, storage := range storages {
			c.processStorage(ch, storage, system.ID)

			for _, controller := range storage.StorageControllers {
				c.processStorageController(ch, controller, system.ID, storage.ID)
			}

			drives, err := storage.Drives()
			if err != nil {
				return fmt.Errorf("error collecting /Systems/%s/Storage/%s/Drives: %s", system.ID, storage.ID, err)
			}
			for _, drive := range drives {
				c.processDrive(ch, drive, system.ID)
			}
		}
	}

	return nil
}

func (c *SystemCollector) processSystem(ch chan<- prometheus.Metric, system *redfish.ComputerSystem) {
	constLabels := prometheus.Labels{"id": system.ID, "name": system.Name, "system_id": system.ID, "system_type": string(system.SystemType)}

	powerStateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "power_state"),
		"System power state; 0: Off, 1: On, 2: PoweringOn, 3: PoweringOff",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "health"),
		"System health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "state"),
		"System state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	if e := enumPowerState(system.PowerState); e >= 0 {
		ch <- prometheus.MustNewConstMetric(powerStateDesc, prometheus.GaugeValue, e)
	}

	if e := enumHealth(system.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(system.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *SystemCollector) processEthernetInterface(ch chan<- prometheus.Metric, intf *redfish.EthernetInterface, systemID string) {
	address, _ := net.ParseMAC(intf.MACAddress)

	constLabels := prometheus.Labels{"id": intf.ID, "name": intf.Name, "system_id": systemID, "interface_type": string(intf.EthernetInterfaceType), "address": address.String()}

	enabledDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "ethernet_interface_status"),
		"Ethernet interface status; 0: Disabled, 1: Enabled",
		nil, constLabels,
	)
	speedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "ethernet_interface_speed_bytes"),
		"Ethernet interface speed, bytes/s",
		[]string{"duplex"}, constLabels,
	)
	linkStatusDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "ethernet_interface_link_status"),
		"Ethernet interface link status; 0: LinkDown, 1: LinkUp, 2: NoLink",
		nil, constLabels,
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

	if e := enumInterfaceLinkStatus(intf.LinkStatus); e >= 0 {
		ch <- prometheus.MustNewConstMetric(linkStatusDesc, prometheus.GaugeValue, e)
	}

	if e := enumHealth(intf.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(intf.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *SystemCollector) processMemory(ch chan<- prometheus.Metric, memory *redfish.Memory, systemID string) {
	constLabels := prometheus.Labels{"id": memory.ID, "name": memory.Name, "system_id": systemID, "memory_type": string(memory.MemoryType)}

	cacheSizeDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "memory_cache_size_bytes"),
		"Memory cache size, bytes",
		nil, constLabels,
	)
	capacityDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "memory_capacity_bytes"),
		"Memory capacity, bytes",
		nil, constLabels,
	)
	nonVolatileSizeDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "memory_non_volatile_size_desc"),
		"Memory non-volatile size, bytes",
		nil, constLabels,
	)
	operatingSpeedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "memory_operating_speed_hertz"),
		"Memory operating speed, Hz",
		nil, constLabels,
	)
	volatileSizeDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "memory_volatile_size_desc"),
		"Memory volatile size, bytes",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "memory_health"),
		"Memory health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "memory_state"),
		"Memory state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(cacheSizeDesc, prometheus.GaugeValue, float64(memory.CacheSizeMiB)*mebi)
	ch <- prometheus.MustNewConstMetric(capacityDesc, prometheus.GaugeValue, float64(memory.CapacityMiB)*mebi)
	ch <- prometheus.MustNewConstMetric(nonVolatileSizeDesc, prometheus.GaugeValue, float64(memory.NonVolatileSizeMiB)*mebi)
	ch <- prometheus.MustNewConstMetric(operatingSpeedDesc, prometheus.GaugeValue, float64(memory.OperatingSpeedMhz)*mega)
	ch <- prometheus.MustNewConstMetric(volatileSizeDesc, prometheus.GaugeValue, float64(memory.VolatileSizeMiB)*mebi)

	if e := enumHealth(memory.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(memory.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *SystemCollector) processNetworkInterface(ch chan<- prometheus.Metric, intf *redfish.NetworkInterface, systemID string) {
	constLabels := prometheus.Labels{"id": intf.ID, "name": intf.Name, "system_id": systemID}

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "network_interface_health"),
		"Network interface health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "network_interface_state"),
		"Network interface state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	if e := enumHealth(intf.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(intf.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *SystemCollector) processPCIeDevice(ch chan<- prometheus.Metric, device *redfish.PCIeDevice, systemID string) {
	constLabels := prometheus.Labels{"id": device.ID, "name": device.Name, "system_id": systemID, "device_type": string(device.DeviceType)}

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "pcie_device_health"),
		"PCIe device health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "pcie_device_state"),
		"PCIe device state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	if e := enumHealth(device.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(device.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *SystemCollector) processProcessor(ch chan<- prometheus.Metric, processor *redfish.Processor, systemID string) {
	constLabels := prometheus.Labels{"id": processor.ID, "name": processor.Name, "system_id": systemID, "processor_type": string(processor.ProcessorType)}

	maxSpeedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "processor_speed_max_hertz"),
		"Maximum processor speed, Hz",
		nil, constLabels,
	)
	maxTDPDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "processor_tdp_max_watts"),
		"Maximum processor TDP, W",
		nil, constLabels,
	)
	tdpDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "processor_tdp_current_wats"),
		"Current processor TDP, W",
		nil, constLabels,
	)
	coresDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "processor_cores"),
		"Total processor cores",
		nil, constLabels,
	)
	enabledCoresDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "processor_cores_enabled"),
		"Enabled processor cores",
		nil, constLabels,
	)
	threadsDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "processor_threads"),
		"Processor threads",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "processor_health"),
		"Processor health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "processor_state"),
		"Processor state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(maxSpeedDesc, prometheus.GaugeValue, float64(processor.MaxSpeedMHz)*mega)
	ch <- prometheus.MustNewConstMetric(maxTDPDesc, prometheus.GaugeValue, float64(processor.MaxTDPWatts))
	ch <- prometheus.MustNewConstMetric(tdpDesc, prometheus.GaugeValue, float64(processor.TDPWatts))
	ch <- prometheus.MustNewConstMetric(coresDesc, prometheus.GaugeValue, float64(processor.TotalCores))
	ch <- prometheus.MustNewConstMetric(enabledCoresDesc, prometheus.GaugeValue, float64(processor.TotalEnabledCores))
	ch <- prometheus.MustNewConstMetric(threadsDesc, prometheus.GaugeValue, float64(processor.TotalThreads))

	if e := enumHealth(processor.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(processor.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *SystemCollector) processStorage(ch chan<- prometheus.Metric, storage *redfish.Storage, systemID string) {
	constLabels := prometheus.Labels{"id": storage.ID, "name": storage.Name, "system_id": systemID, "storage_id": storage.ID}

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_health"),
		"Storage health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_state"),
		"Storage state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	if e := enumHealth(storage.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(storage.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *SystemCollector) processStorageController(ch chan<- prometheus.Metric, controller redfish.StorageController, systemID string, storageID string) {
	constLabels := prometheus.Labels{"id": controller.MemberID, "name": controller.Name, "system_id": systemID, "storage_id": storageID}

	speedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_controller_speed_bytes"),
		"Storage controller speed, bytes/s",
		nil, constLabels,
	)
	persistentCacheSizeDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_controller_cache_size_persistent_bytes"),
		"Persistent cache size, bytes",
		nil, constLabels,
	)
	cacheSizeDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_controller_cache_size_bytes"),
		"Total cache size, bytes",
		nil, constLabels,
	)

	cacheHealthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_controller_health"),
		"Storage controller cache health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	cacheStateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_controller_state"),
		"Storage controller cache state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)
	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_controller_health"),
		"Storage controller health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "storage_controller_state"),
		"Storage controller state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(speedDesc, prometheus.GaugeValue, float64(controller.SpeedGbps)*giga/8)
	ch <- prometheus.MustNewConstMetric(persistentCacheSizeDesc, prometheus.GaugeValue, float64(controller.CacheSummary.PersistentCacheSizeMiB)*mebi)
	ch <- prometheus.MustNewConstMetric(cacheSizeDesc, prometheus.GaugeValue, float64(controller.CacheSummary.TotalCacheSizeMiB)*mebi)

	if e := enumHealth(controller.CacheSummary.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(cacheHealthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(controller.CacheSummary.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(cacheStateDesc, prometheus.GaugeValue, e)
	}
	if e := enumHealth(controller.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(controller.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func (c *SystemCollector) processDrive(ch chan<- prometheus.Metric, drive *redfish.Drive, systemID string) {
	constLabels := prometheus.Labels{"id": drive.ID, "name": drive.Name, "system_id": systemID, "drive_type": string(drive.MediaType)}

	capableSpeedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_speed_capable_bytes"),
		"Fastest capable drive speed, bytes/s",
		nil, constLabels,
	)
	capacityDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_capacity_bytes"),
		"Drive raw capacity, bytes",
		nil, constLabels,
	)
	failurePredictedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_predicted_failure"),
		"Drive failure predicted; 0: NoFailure, 1: Failure",
		nil, constLabels,
	)
	negotiatedSpeedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_speed_negotiated_bytes"),
		"Actual drive speed, bytes/s",
		nil, constLabels,
	)
	writeCacheDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_write_cache_status"),
		"Drive write cache status; 0: Disabled, 1: Enabled",
		nil, constLabels,
	)
	encryptionDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_encryption_status"),
		"Drive encryption status; 0: Unencrypted, 1: Unlocked, 2: Locked, 3: Foreign",
		[]string{"encryption_ability"}, constLabels,
	)
	hotspareDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_hotspare_type"),
		"Drive hotspare type; 0: None, 1: Global, 2: Chassis, 3: Dedicated",
		[]string{"hotspare_replacement_mode"}, constLabels,
	)
	rotationSpeedDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_rotation_speed_rpm"),
		"Drive rotation speed, RPM",
		nil, constLabels,
	)
	lifeLeftDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_predicted_media_life_left_ratio"),
		"Drive media life left, %",
		nil, constLabels,
	)
	statusDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_status"),
		"Drive status; 0: Fail, 1: OK, 2: Rebuild, 3: PredictiveFailureAnalysis, 4: Hotspare, 5: InACriticalArray, 6: InAFailedArray",
		nil, constLabels,
	)

	healthDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_health"),
		" health; 0: OK, 1: Warning, 2: Critical",
		nil, constLabels,
	)
	stateDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "system", "drive_state"),
		" state; 0: Disabled, 1: Enabled, 2: StandbyOffline, 3: StandbySpare, 4: InTest, 5: Starting, 6: Absent, 7: UnavailableOffline, 8: Deferring, 9: Quiesced, 10: Updating",
		nil, constLabels,
	)

	ch <- prometheus.MustNewConstMetric(capableSpeedDesc, prometheus.GaugeValue, float64(drive.CapableSpeedGbs)*giga/8)
	ch <- prometheus.MustNewConstMetric(capacityDesc, prometheus.GaugeValue, float64(drive.CapacityBytes))
	ch <- prometheus.MustNewConstMetric(failurePredictedDesc, prometheus.GaugeValue, btof(drive.FailurePredicted))
	ch <- prometheus.MustNewConstMetric(negotiatedSpeedDesc, prometheus.GaugeValue, float64(drive.NegotiatedSpeedGbs)*giga/8)
	ch <- prometheus.MustNewConstMetric(writeCacheDesc, prometheus.GaugeValue, btof(drive.WriteCacheEnabled))

	switch drive.MediaType {
	case redfish.HDDMediaType, redfish.SMRMediaType:
		ch <- prometheus.MustNewConstMetric(rotationSpeedDesc, prometheus.GaugeValue, float64(drive.RotationSpeedRPM))
	case redfish.SSDMediaType:
		ch <- prometheus.MustNewConstMetric(lifeLeftDesc, prometheus.GaugeValue, float64(drive.PredictedMediaLifeLeftPercent)/100)
	}

	if e := enumEncryptionStatus(drive.EncryptionStatus); e >= 0 {
		ch <- prometheus.MustNewConstMetric(encryptionDesc, prometheus.GaugeValue, e, string(drive.EncryptionAbility))
	}
	if e := enumHotspareType(drive.HotspareType); e >= 0 {
		ch <- prometheus.MustNewConstMetric(hotspareDesc, prometheus.GaugeValue, e, string(drive.HotspareReplacementMode))
	}
	if e := enumDriveStatusIndicator(drive.StatusIndicator); e >= 0 {
		ch <- prometheus.MustNewConstMetric(statusDesc, prometheus.GaugeValue, e)
	}

	if e := enumHealth(drive.Status.Health); e >= 0 {
		ch <- prometheus.MustNewConstMetric(healthDesc, prometheus.GaugeValue, e)
	}
	if e := enumState(drive.Status.State); e >= 0 {
		ch <- prometheus.MustNewConstMetric(stateDesc, prometheus.GaugeValue, e)
	}
}

func enumInterfaceLinkStatus(e redfish.LinkStatus) float64 {
	switch e {
	case redfish.LinkDownLinkStatus:
		return 0
	case redfish.LinkUpLinkStatus:
		return 1
	case redfish.NoLinkLinkStatus:
		return 2
	default:
		return -1
	}
}

func enumEncryptionStatus(e redfish.EncryptionStatus) float64 {
	switch e {
	case redfish.UnecryptedEncryptionStatus, redfish.UnencryptedEncryptionStatus:
		return 0
	case redfish.UnlockedEncryptionStatus:
		return 1
	case redfish.LockedEncryptionStatus:
		return 2
	case redfish.ForeignEncryptionStatus:
		return 3
	default:
		return -1
	}
}

func enumHotspareType(e redfish.HotspareType) float64 {
	switch e {
	case redfish.NoneHotspareType:
		return 0
	case redfish.GlobalHotspareType:
		return 1
	case redfish.ChassisHotspareType:
		return 2
	case redfish.DedicatedHotspareType:
		return 3
	default:
		return -1
	}
}

func enumDriveStatusIndicator(e redfish.StatusIndicator) float64 {
	switch e {
	case redfish.FailStatusIndicator:
		return 0
	case redfish.OKStatusIndicator:
		return 1
	case redfish.RebuildStatusIndicator:
		return 2
	case redfish.PredictiveFailureAnalysisStatusIndicator:
		return 3
	case redfish.HotspareStatusIndicator:
		return 4
	case redfish.InACriticalArrayStatusIndicator:
		return 5
	case redfish.InAFailedArrayStatusIndicator:
		return 6
	default:
		return -1
	}
}
