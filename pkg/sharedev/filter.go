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

	s, _ := getShareDevState(state)
	if s != nil {
		log.Printf("ShareDevPlugin PreFilter: State: %v", s)
		return nil, framework.NewStatus(framework.Success)
	}

	podQ, err := sp.parsePod(pod)
	if err != nil {
		log.Printf("ShareDevPlugin PreFilter: error parsing pod: %s", err.Error())
		return nil, framework.NewStatus(framework.Unschedulable, err.Error())
	}

	state.Write(ShareDevStateKey, &ShareDevState{
		FreeDeviceResourcesPerNode: map[string][]FreeDeviceResources{},
		NodeNameToIP:               map[string]string{},
		PodQ:                       *podQ,
	})

	return nil, framework.NewStatus(framework.Success)
}

func (sp *ShareDevPlugin) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	log.Println("ShareDevPlugin Filter is working!!")

	shareDevState, err := getShareDevState(state)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}

	nodeName := nodeInfo.Node().Name
	var nodeIP string
	for _, addr := range nodeInfo.Node().Status.Addresses {
		if addr.Type == "InternalIP" {
			nodeIP = addr.Address
			break
		}
	}
	if nodeIP == "" {
		return framework.NewStatus(framework.Error, "node does not have InternalIP")
	}
	log.Println("ShareDevPlugin nodeIP: ", nodeIP)

	freeResources, err := getFreeResources(nodeIP, shareDevState.PodQ)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}
	if len(freeResources) == 0 {
		return framework.NewStatus(framework.Unschedulable, "no resources available")
	}

	shareDevState.FreeDeviceResourcesPerNode[nodeInfo.Node().Name] = freeResources
	shareDevState.NodeNameToIP[nodeName] = nodeIP
	log.Println("ShareDevPlugin freeResources: ", freeResources)

	for _, free := range freeResources {
		if podFits(shareDevState.PodQ, free) {
			log.Printf("ShareDevPlugin Filter: pod %s fits device %s", pod.Name, free.DeviceId)
			return framework.NewStatus(framework.Success)
		}
	}

	// DONE: check if CLASSIC resources like CPU and memory are available, maybe use the normal Filter plugin for that?
	// Yes, the default NodeResourcesFit plugin already implements filter

	return framework.NewStatus(framework.UnschedulableAndUnresolvable, "no resources available")
}

func (sp *ShareDevPlugin) PostFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	log.Println("ShareDevPlugin PostFilter is working!!")

	shareDevState, err := getShareDevState(state)
	if err != nil {
		return nil, framework.NewStatus(framework.Error, err.Error())
	}

	vendor := shareDevState.PodQ.Vendor
	model := shareDevState.PodQ.Model

	nodeName, err := sp.allocateNewDevice(vendor, model)
	if err != nil {
		log.Printf("ShareDevPlugin PostFilter: error allocating new device: %s", err.Error())
		return nil, framework.NewStatus(framework.Error, err.Error())
	}

	return &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: nodeName}}, framework.NewStatus(framework.Success)
}

func (sp *ShareDevPlugin) parsePod(pod *v1.Pod) (*PodRequestedQuota, error) {
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

	return &PodRequestedQuota{
		PodId:    pod.Name,
		Vendor:   pod.Labels["sharedev.vendor"],
		Model:    pod.Labels["sharedev.model"],
		Requests: requests,
		Limits:   limits,
		Memory:   memory,
	}, nil
}
