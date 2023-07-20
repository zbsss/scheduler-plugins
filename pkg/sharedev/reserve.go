package sharedev

import (
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

func copyPod(pod *v1.Pod, hostIP, nodeName, deviceId string) *v1.Pod {
	podCopy := pod.DeepCopy()
	podCopy.ResourceVersion = ""
	podCopy.Spec.NodeName = nodeName

	// this label is used by Device Managers to query healthy pods
	// and run garbage collection to free up devices
	podCopy.Labels["sharedev"] = "true"

	for i := range podCopy.Spec.Containers {
		c := &podCopy.Spec.Containers[i]
		c.Env = append(c.Env,
			v1.EnvVar{
				Name:  "CLIENT_ID",
				Value: pod.Name,
			},
			v1.EnvVar{
				Name:  "DEVICE_ID",
				Value: deviceId,
			},
			v1.EnvVar{
				Name:  "HOST_IP",
				Value: hostIP,
			},
		)
	}

	return podCopy
}

func (sp *ShareDevPlugin) Reserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	log.Println("ShareDevPlugin Reserve is working!!")

	shareDevState, err := getShareDevState(state)
	if err != nil {
		return framework.NewStatus(framework.Unschedulable, err.Error())
	}

	nodeIP := shareDevState.NodeNameToIP[nodeName]
	_, device := getBestFit(shareDevState.PodQ, shareDevState.FreeDeviceResourcesPerNode[nodeName])

	err = reservePodQuota(nodeIP, device.DeviceId, shareDevState.PodQ)
	if err != nil {
		log.Printf("ShareDevPlugin Reserve: error reserving device: %s", err.Error())
		return framework.NewStatus(framework.Error, err.Error())
	}

	podCopy := copyPod(pod, nodeIP, nodeName, device.DeviceId)

	err = sp.handle.ClientSet().CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("ShareDevPlugin Reserve: error deleting pod %s: %s", pod.Name, err.Error())
		return framework.NewStatus(framework.Error, err.Error())
	}
	log.Printf("ShareDevPlugin [Reserve-> Delete] pod: %v/%v(%v) in node %v", pod.Namespace, pod.Name, pod.UID, pod.Spec.NodeName)

	_, err = sp.handle.ClientSet().CoreV1().Pods(pod.Namespace).Create(ctx, podCopy, metav1.CreateOptions{})
	if err != nil {
		log.Printf("ShareDevPlugin Reserve: error creating pod %s: %s", podCopy.Name, err.Error())
		return framework.NewStatus(framework.Error, err.Error())
	}

	shareDevState.ReservedDeviceId = device.DeviceId
	log.Printf("ShareDevPlugin [Reserve] New Pod %v/%v(%v) v.s. Old Pod  %v/%v(%v)", podCopy.Namespace, podCopy.Name, podCopy.UID, pod.Namespace, pod.Name, pod.UID)

	return framework.NewStatus(framework.Success)
}

func (sp *ShareDevPlugin) Unreserve(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) {
	log.Println("ShareDevPlugin Unreserve is working!!")
}
