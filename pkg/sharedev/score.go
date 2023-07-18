package sharedev

import (
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func (sp *ShareDevPlugin) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

func (sp *ShareDevPlugin) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	log.Println("ShareDevPlugin Score is working!!")

	score := framework.MinNodeScore

	requestsScaleFactor := 100.0
	memoryScaleFactor := 100.0

	shareDevState, err := getShareDevState(state)
	if err != nil {
		return score, framework.NewStatus(framework.Error, err.Error())
	}

	log.Printf("ShareDevPlugin shareDevState: %v", shareDevState)
	log.Printf("ShareDevPlugin FreeResourcesPerNode: %v", shareDevState.FreeResourcesPerNode)
	log.Println("ShareDevPlugin score free resources: ", shareDevState.FreeResourcesPerNode[nodeName])

	for _, free := range shareDevState.FreeResourcesPerNode[nodeName] {
		var nscore int64 = 0

		log.Printf("ShareDevPlugin NScore: %d for node: %s", nscore, nodeName)

		if podFits(shareDevState.PodSpec, free) {
			nscore = int64(
				(free.Requests-shareDevState.PodSpec.Requests)*requestsScaleFactor +
					(free.Memory-shareDevState.PodSpec.Memory)*memoryScaleFactor,
			)

			log.Printf("ShareDevPlugin NScore: %d for node: %s", nscore, nodeName)
		}

		if nscore > score {
			score = nscore
		}
	}

	if score > framework.MaxNodeScore {
		score = framework.MaxNodeScore
	}

	log.Printf("ShareDevPlugin Score: %d for node: %s", score, nodeName)

	return score, framework.NewStatus(framework.Success)
}
