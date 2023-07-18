package sharedev

import (
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func (sp *ShareDevPlugin) Reserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) *framework.Status {
	log.Println("ShareDevPlugin Reserve is working!!")

	return framework.NewStatus(framework.Success)
}

func (sp *ShareDevPlugin) Unreserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) {
	log.Println("ShareDevPlugin Unreserve is working!!")
}
