package sharedev

import (
	"fmt"
	"log"

	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	ShareDevStateKey = "ShareDevStateKey"
)

type PodRequestedQuota struct {
	PodId    string
	Vendor   string
	Model    string
	Requests float64
	Limits   float64
	Memory   float64
}

type FreeDeviceResources struct {
	DeviceId string
	Requests float64
	Memory   float64
}

type ShareDevState struct {
	PodQ                       PodRequestedQuota
	FreeDeviceResourcesPerNode map[string][]FreeDeviceResources
	NodeNameToIP               map[string]string
	ReservedDeviceId           string
}

func (s *ShareDevState) Clone() framework.StateData {
	n := ShareDevState{
		FreeDeviceResourcesPerNode: make(map[string][]FreeDeviceResources),
		NodeNameToIP:               make(map[string]string),
		PodQ:                       s.PodQ,
		ReservedDeviceId:           s.ReservedDeviceId,
	}

	log.Printf("Running Clone: Before State: %v", s)

	for k, v := range s.FreeDeviceResourcesPerNode {
		arr := make([]FreeDeviceResources, len(v))
		copy(arr, v)

		n.FreeDeviceResourcesPerNode[k] = arr
	}

	for k, v := range s.NodeNameToIP {
		n.NodeNameToIP[k] = v
	}

	log.Printf("Running Clone: After State: %v", n)

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
