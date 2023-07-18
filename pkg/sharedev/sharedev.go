package sharedev

import (
	"context"
	"log"

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
var _ framework.ScorePlugin = &ShareDevPlugin{}
var _ framework.ReservePlugin = &ShareDevPlugin{}
var _ framework.PreBindPlugin = &ShareDevPlugin{}

func (sp *ShareDevPlugin) PreBind(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) *framework.Status {
	log.Println("ShareDevPlugin PreBind is working!!")

	return framework.NewStatus(framework.Success)
}

// Name returns name of the plugin.
func (sp *ShareDevPlugin) Name() string {
	return Name
}

func podFits(pod ShareDevPodSpec, freeResources FreeResources) bool {
	return pod.Requests <= freeResources.Requests && pod.Memory <= freeResources.Memory
}

func New(_ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &ShareDevPlugin{}, nil
}
