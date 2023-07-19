package sharedev

import (
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func (sp *ShareDevPlugin) Reserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) *framework.Status {
	log.Println("ShareDevPlugin Reserve is working!!")

	shareDevState, err := getShareDevState(state)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, err.Error())
	}

	_, device := getBestFit(shareDevState.PodQ, shareDevState.FreeDeviceResourcesPerNode[nodeName])

	err = reservePodQuota(shareDevState.NodeNameToIP[nodeName], device.DeviceId, shareDevState.PodQ)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, err.Error())
	}

	shareDevState.ReservedDeviceId = device.DeviceId
	log.Printf("ShareDevPlugin Reserve: reserved device: %s on node: %s", device.DeviceId, nodeName)

	return framework.NewStatus(framework.Success)
}

func (sp *ShareDevPlugin) Unreserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) {
	log.Println("ShareDevPlugin Unreserve is working!!")

	shareDevState, err := getShareDevState(state)
	if err != nil {
		log.Printf("ShareDevPlugin Unreserve: error getting ShareDevState: %s", err.Error())
	}

	err = unreservePodQuota(shareDevState.NodeNameToIP[nodeName], shareDevState.ReservedDeviceId, shareDevState.PodQ)
	if err != nil {
		log.Printf("ShareDevPlugin Unreserve: error unreserving device: %s", err.Error())
	}

	log.Printf("ShareDevPlugin Reserve: reserved device: %s on node: %s", shareDevState.ReservedDeviceId, nodeName)
	shareDevState.ReservedDeviceId = ""
}
