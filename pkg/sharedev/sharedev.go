package sharedev

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const (
	Name              = "ShareDevPlugin"
	deviceManagerPort = "50051"
)

type ShareDevPlugin struct {
	handle framework.Handle
}

var _ framework.PreFilterPlugin = &ShareDevPlugin{}
var _ framework.FilterPlugin = &ShareDevPlugin{}
var _ framework.ScorePlugin = &ShareDevPlugin{}
var _ framework.ReservePlugin = &ShareDevPlugin{}

// Name returns name of the plugin.
func (sp *ShareDevPlugin) Name() string {
	return Name
}

func podFits(pod PodRequestedQuota, freeResources FreeDeviceResources) bool {
	return pod.Requests <= freeResources.Requests && pod.Memory <= freeResources.Memory
}

func New(_ runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	return &ShareDevPlugin{
		handle: handle,
	}, nil
}
