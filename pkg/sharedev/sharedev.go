package sharedev

import (
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = "ShareDevPlugin"

type ShareDevPlugin struct{}

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

	return framework.NewStatus(framework.Success)
}

func New(_ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &ShareDevPlugin{}, nil
}
