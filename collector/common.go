package collector

import (
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
)

const (
	kibi float64 = 1024
	mebi         = 1024 * kibi
	gibi         = 1024 * mebi
)

const (
	kilo float64 = 1000
	mega         = 1000 * kilo
	giga         = 1000 * mega
)

func btof(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func enumPowerState(e redfish.PowerState) float64 {
	switch e {
	case redfish.OffPowerState:
		return 0
	case redfish.OnPowerState:
		return 1
	case redfish.PoweringOnPowerState:
		return 2
	case redfish.PoweringOffPowerState:
		return 3
	default:
		return -1
	}
}

func enumHealth(e common.Health) float64 {
	switch e {
	case common.OKHealth:
		return 0
	case common.WarningHealth:
		return 1
	case common.CriticalHealth:
		return 2
	default:
		return -1
	}
}

func enumState(e common.State) float64 {
	switch e {
	case common.DisabledState:
		return 0
	case common.EnabledState:
		return 1
	case common.StandbyOfflineState:
		return 2
	case common.StandbySpareState:
		return 3
	case common.InTestState:
		return 4
	case common.StartingState:
		return 5
	case common.AbsentState:
		return 6
	case common.UnavailableOfflineState:
		return 7
	case common.DeferringState:
		return 8
	case common.QuiescedState:
		return 9
	case common.UpdatingState:
		return 10
	default:
		return -1
	}
}
