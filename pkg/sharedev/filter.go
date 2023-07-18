package sharedev

import (
	"context"
	"fmt"
	"log"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func (sp *ShareDevPlugin) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (sp *ShareDevPlugin) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	log.Println("ShareDevPlugin Hello world")
	log.Println("ShareDevPlugin PreFilter is working!!")

	podSpec, err := sp.parsePod(pod)
	if err != nil {
		return nil, framework.NewStatus(framework.Unschedulable, err.Error())
	}

	state.Write(ShareDevStateKey, &ShareDevState{
		FreeResourcesPerNode: map[string][]FreeResources{},
		PodSpec:              *podSpec,
	})

	return nil, framework.NewStatus(framework.Success)
}

func (sp *ShareDevPlugin) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	log.Println("ShareDevPlugin Filter is working!!")

	shareDevState, err := getShareDevState(state)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, err.Error())
	}

	var nodeIP string
	for _, addr := range nodeInfo.Node().Status.Addresses {
		if addr.Type == "InternalIP" {
			nodeIP = addr.Address
			break
		}
	}
	if nodeIP == "" {
		return framework.NewStatus(framework.Unschedulable, "node does not have InternalIP")
	}
	log.Println("ShareDevPlugin nodeIP: ", nodeIP)

	freeResources, err := sp.getFreeResources(nodeIP, shareDevState.PodSpec)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, err.Error())
	}
	if len(freeResources) == 0 {
		return framework.NewStatus(framework.Unschedulable, "no resources available")
	}

	shareDevState.FreeResourcesPerNode[nodeInfo.Node().Name] = freeResources

	log.Println("ShareDevPlugin freeResources: ", freeResources)

	for _, free := range freeResources {
		if podFits(shareDevState.PodSpec, free) {
			return framework.NewStatus(framework.Success)
		}
	}

	// DONE: check if CLASSIC resources like CPU and memory are available, maybe use the normal Filter plugin for that?
	// Yes, the default NodeResourcesFit plugin already implements filter

	return framework.NewStatus(framework.Unschedulable, "no resources available")
}

func (sp *ShareDevPlugin) parsePod(pod *v1.Pod) (*ShareDevPodSpec, error) {
	if pod.Labels["sharedev.vendor"] == "" || pod.Labels["sharedev.model"] == "" {
		return nil, fmt.Errorf("pod does not have sharedev.vendor or sharedev.model label")
	}

	requests, err := strconv.ParseFloat(pod.Labels["sharedev.requests"], 64)
	if err != nil {
		return nil, fmt.Errorf("label sharedev.requests is not a float")
	}

	memory, err := strconv.ParseFloat(pod.Labels["sharedev.memory"], 64)
	if err != nil {
		return nil, fmt.Errorf("label sharedev.memory is not a float")
	}

	limits, err := strconv.ParseFloat(pod.Labels["sharedev.limits"], 64)
	if err != nil {
		limits = requests
	}

	return &ShareDevPodSpec{
		Vendor:   pod.Labels["sharedev.vendor"],
		Model:    pod.Labels["sharedev.model"],
		Requests: requests,
		Limits:   limits,
		Memory:   memory,
	}, nil
}
