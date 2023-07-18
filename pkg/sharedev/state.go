package sharedev

import (
	"fmt"

	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	ShareDevStateKey = "ShareDevStateKey"
)

type FreeResources struct {
	DeviceId string
	Requests float64
	Memory   float64
}

type ShareDevState struct {
	FreeResourcesPerNode map[string][]FreeResources
}

func (s *ShareDevState) Clone() framework.StateData {
	n := ShareDevState{
		FreeResourcesPerNode: make(map[string][]FreeResources),
	}

	for k, v := range s.FreeResourcesPerNode {
		arr := make([]FreeResources, len(v))
		copy(arr, v)

		n.FreeResourcesPerNode[k] = arr
	}

	return &n
}

func getShareDevState(state *framework.CycleState) (*ShareDevState, error) {
	s, err := state.Read(ShareDevStateKey)
	if err != nil {
		// ShareDevState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("error reading %q from cycleState: %w", ShareDevStateKey, err)
	}

	sd, ok := s.(*ShareDevState)
	if !ok {
		return nil, fmt.Errorf("%+v  convert to ShareDevState error", s)
	}
	return sd, nil
}
