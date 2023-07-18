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

var _ framework.FilterPlugin = &ShareDevPlugin{}

// Name returns name of the plugin.
func (ep *ShareDevPlugin) Name() string {
	return Name
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
		// TODO: verify this is actually the IP address of the Node
		if addr.Type == "InternalIP" {
			nodeIP = addr.Address
		}
	}

	log.Println("ShareDevPlugin nodeIP: ", nodeIP)

	fitsDevices, err := e.callDeviceManager(nodeIP, pod.Labels["sharedev.vendor"], pod.Labels["sharedev.model"], requests, limits, memory)
	if err != nil || len(fitsDevices) == 0 {
		return framework.NewStatus(framework.Unschedulable, err.Error())
	}

	log.Println("ShareDevPlugin fitsDevices: ", fitsDevices)

	// TODO: write fitsDevices to state
	// state.Write(...)

	// TODO: is the state kept in context of a scheduling of a single Pod?

	return framework.NewStatus(framework.Success)
}

func (e *ShareDevPlugin) callDeviceManager(nodeIP, vendor, model string, requests, limits, memory float64) ([]string, error) {
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

	fitsDevices := []string{}
	for _, free := range resp.Free {
		if memory <= free.Memory && requests <= free.Requests {
			fitsDevices = append(fitsDevices, free.DeviceId)
		}
	}

	return fitsDevices, nil
}

func New(_ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &ShareDevPlugin{}, nil
}
