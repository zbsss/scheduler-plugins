package sharedev

import (
	"context"
	"log"
	"math"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func (sp *ShareDevPlugin) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	log.Println("before normalization: ", "scores", scores)

	// Get Min and Max Scores to normalize between framework.MaxNodeScore and framework.MinNodeScore
	minCost, maxCost := getMinMaxScores(scores)

	// If all nodes were given the minimum score, return
	if minCost == 0 && maxCost == 0 {
		return nil
	}

	var normCost float64
	for i := range scores {
		if maxCost != minCost { // If max != min
			// node_normalized_cost = MAX_SCORE * ( ( nodeScore - minCost) / (maxCost - minCost)
			// nodeScore = MAX_SCORE - node_normalized_cost
			normCost = float64(framework.MaxNodeScore) * float64(scores[i].Score-minCost) / float64(maxCost-minCost)
			scores[i].Score = framework.MaxNodeScore - int64(normCost)
		} else { // If maxCost = minCost, avoid division by 0
			normCost = float64(scores[i].Score - minCost)
			scores[i].Score = framework.MaxNodeScore - int64(normCost)
		}
	}
	log.Println("after normalization: ", "scores", scores)
	return nil
}

// MinMax : get min and max scores from NodeScoreList
func getMinMaxScores(scores framework.NodeScoreList) (int64, int64) {
	var max int64 = math.MinInt64 // Set to min value
	var min int64 = math.MaxInt64 // Set to max value

	for _, nodeScore := range scores {
		if nodeScore.Score > max {
			max = nodeScore.Score
		}
		if nodeScore.Score < min {
			min = nodeScore.Score
		}
	}
	// return min and max scores
	return min, max
}

func (sp *ShareDevPlugin) ScoreExtensions() framework.ScoreExtensions {
	return sp
}

func (sp *ShareDevPlugin) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	log.Println("ShareDevPlugin Score is working!!")

	shareDevState, err := getShareDevState(state)
	if err != nil {
		return framework.MinNodeScore, framework.NewStatus(framework.Error, err.Error())
	}

	highestScore, device := getBestFit(shareDevState.PodQ, shareDevState.FreeDeviceResourcesPerNode[nodeName])

	log.Printf("ShareDevPlugin Score: %d for node: %s, deviceId: %s", highestScore, nodeName, device.DeviceId)

	return highestScore, framework.NewStatus(framework.Success)
}

func getBestFit(pod PodRequestedQuota, freeResources []FreeDeviceResources) (int64, FreeDeviceResources) {
	var device FreeDeviceResources
	highestScore := framework.MinNodeScore

	for _, free := range freeResources {
		if podFits(pod, free) {
			score := int64((free.Requests - pod.Requests + free.Memory - pod.Memory) * 100)

			if score > highestScore {
				highestScore = score
				device = free
			}
		}
	}

	if highestScore > framework.MaxNodeScore {
		highestScore = framework.MaxNodeScore
	}

	return highestScore, device
}
