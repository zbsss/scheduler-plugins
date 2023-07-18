package sharedev

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	pb "github.com/zbsss/device-manager/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const (
	Name              = "ShareDevPlugin"
	deviceManagerPort = "50051"
)

type ShareDevPlugin struct {
}

var _ framework.PreFilterPlugin = &ShareDevPlugin{}
var _ framework.FilterPlugin = &ShareDevPlugin{}

// Name returns name of the plugin.
func (ep *ShareDevPlugin) Name() string {
	return Name
}

func (e *ShareDevPlugin) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func (e *ShareDevPlugin) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	log.Println("ShareDevPlugin PreFilter is working!!")

	state.Write(ShareDevStateKey, &ShareDevState{
		FreeResourcesPerNode: map[string][]FreeResources{},
	})

	return nil, framework.NewStatus(framework.Success)
}

func (e *ShareDevPlugin) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	log.Println("ShareDevPlugin Filter is working!!")

	if pod.Labels["sharedev.vendor"] == "" || pod.Labels["sharedev.model"] == "" {
		return framework.NewStatus(framework.Unschedulable, "pod does not have sharedev.vendor or sharedev.model label")
	}

	requests, err := strconv.ParseFloat(pod.Labels["sharedev.requests"], 64)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, "label sharedev.requests is not a float")
	}

	memory, err := strconv.ParseFloat(pod.Labels["sharedev.memory"], 64)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, "label sharedev.memory is not a float")
	}

	limits, err := strconv.ParseFloat(pod.Labels["sharedev.limits"], 64)
	if err != nil {
		limits = requests
	}

	var nodeIP string
	for _, addr := range nodeInfo.Node().Status.Addresses {
		if addr.Type == "InternalIP" {
			nodeIP = addr.Address
			break
		}
	}
	log.Println("ShareDevPlugin nodeIP: ", nodeIP)
	if nodeIP == "" {
		return framework.NewStatus(framework.Unschedulable, "node does not have InternalIP")
	}

	freeResources, err := e.callDeviceManager(nodeIP, pod.Labels["sharedev.vendor"], pod.Labels["sharedev.model"], requests, limits, memory)
	if err != nil || len(freeResources) == 0 {
		return framework.NewStatus(framework.Unschedulable, err.Error())
	}

	shareDevState, err := getShareDevState(state)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, err.Error())
	}

	shareDevState.FreeResourcesPerNode[nodeIP] = freeResources

	log.Println("ShareDevPlugin freeResources: ", freeResources)

	for _, free := range freeResources {
		if requests <= free.Requests && memory <= free.Memory {
			return framework.NewStatus(framework.Success)
		}
	}

	// TODO: check if CLASSIC resources like CPU and memory are available
	// maybe use the normal Filter plugin for that?

	return framework.NewStatus(framework.Unschedulable, "no resources available")
}

func (e *ShareDevPlugin) callDeviceManager(nodeIP, vendor, model string, requests, limits, memory float64) ([]FreeResources, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("%s:%s", nodeIP, deviceManagerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	grpc := pb.NewDeviceManagerClient(conn)

	resp, err := grpc.GetAvailableDevices(ctx, &pb.GetAvailableDevicesRequest{
		Vendor: vendor,
		Model:  model,
	})
	if err != nil {
		return nil, err
	}

	freeResources := []FreeResources{}
	for _, free := range resp.Free {
		freeResources = append(freeResources, FreeResources{
			DeviceId: free.DeviceId,
			Requests: free.Requests,
			Memory:   free.Memory,
		})
	}

	return freeResources, nil
}

func New(_ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &ShareDevPlugin{}, nil
}
