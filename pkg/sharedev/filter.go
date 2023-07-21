package sharedev

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return nil, framework.NewStatus(framework.Unschedulable, err.Error())
	}

	state.Write(ShareDevStateKey, &ShareDevState{
		FreeDeviceResourcesPerNode: map[string][]FreeDeviceResources{},
		NodeNameToIP:               map[string]string{},
		PodQ:                       *podQ,
	})

	return nil, framework.NewStatus(framework.Success)
}

func (sp *ShareDevPlugin) allocateNewDevice(vendor, model string) (string, error) {
	cli := sp.handle.ClientSet().AppsV1().Deployments("default")

	deployName := "allocator" + "-" + vendor + "-" + model + fmt.Sprint(time.Now().Unix())
	var replicas int32 = 1

	deviceName := vendor + "/" + model
	deviceQuantity, _ := resource.ParseQuantity("1")

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: deployName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas, // One replica
			Selector: &metav1.LabelSelector{ // The deployment will manage pods with these labels
				MatchLabels: map[string]string{
					"app": deployName,
				},
			},
			Template: corev1.PodTemplateSpec{ // The pods that will be created
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deployName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "allocator",
						Image: "docker.io/zbsss/device-allocator:latest",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceName(deviceName): deviceQuantity,
							},
						},
						Env: []corev1.EnvVar{
							{
								Name: "HOST_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "status.hostIP",
									},
								},
							},
						},
					}},
				},
			},
		},
	}

	_, err := cli.Create(context.Background(), deployment, metav1.CreateOptions{})
	if err != nil {
		log.Printf("ShareDevPlugin allocateNewDevice: error creating deployment: %s", err.Error())
		return "", err
	}

	timeout := time.After(60 * time.Second)
	watch, _ := sp.handle.ClientSet().CoreV1().Pods("default").Watch(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s", deployName),
		},
	)
	defer watch.Stop()

	for {
		select {
		case <-timeout:
			log.Printf("ShareDevPlugin allocateNewDevice: timeout waiting for pod to be running")
			return "", fmt.Errorf("timeout waiting for pod to be running")
		case event := <-watch.ResultChan():
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return "", fmt.Errorf("unexpected type: %T", event.Object)
			}

			if pod.Status.Phase == corev1.PodRunning {
				log.Printf("ShareDevPlugin allocateNewDevice: pod %s is running on node %s", pod.Name, pod.Spec.NodeName)
				return pod.Spec.NodeName, nil
			}
		}
	}
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

	// TODO: should I store it in state?

	return &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: nodeName}}, framework.NewStatus(framework.Success)

	// nodeInfos, err := sp.handle.ClientSet().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	// if err != nil {
	// 	log.Printf("ShareDevPlugin PostFilter: error listing nodes: %s", err.Error())
	// 	return nil, framework.NewStatus(framework.Error, fmt.Sprintf("ShareDevPlugin PostFilter: error listing nodes: %s", err.Error()))
	// }

	// zeroDevices, _ := resource.ParseQuantity("0")

	// TODO: instead of doing this look and loading all nodes
	// just call allocateNewDevice but without setting the node name
	// this will cause the the Pod to be allocated on a node with a free device
	// or if there are no free devices, it will fail
	// if it fails that means there are no real devices available in the cluster
	// so we cannot schedule the current pod

	// thats why there was the error UnexpectedAdmissionError
	// because I manually set the nodeName to the first node
	// after a while it ran out of devices
	// but I still kept trying to schedule more pods there with the same device

	// for _, nodeInfo := range nodeInfos.Items {
	// 	nodeName := nodeInfo.Name

	// 	// TODO: better check if node is control plane
	// 	if _, ok := filteredNodeStatusMap[nodeName]; !ok || nodeName == "kind-control-plane" {
	// 		continue
	// 	}

	// 	if nodeInfo.Status.Allocatable[v1.ResourceName(vendor+"/"+model)] == zeroDevices {
	// 		continue
	// 	}

	// 	// schedule an Allocator Pod that will allocate a real device
	// 	// and register it with the Device Manager
	// 	log.Printf("ShareDevPlugin PostFilter: allocating new device on node %s", nodeInfo.Name)
	// 	err = sp.allocateNewDevice(vendor, model)
	// 	if err != nil {
	// 		log.Printf("ShareDevPlugin PostFilter: error allocating new device: %s", err.Error())
	// 		continue
	// 	}

	// 	// TODO: wait for Pod to be running
	// 	time.Sleep(30 * time.Second)

	// 	var nodeIP string
	// 	for _, addr := range nodeInfo.Status.Addresses {
	// 		if addr.Type == "InternalIP" {
	// 			nodeIP = addr.Address
	// 			break
	// 		}
	// 	}
	// 	if nodeIP == "" {
	// 		log.Printf("ShareDevPlugin PostFilter: node %s does not have InternalIP", nodeName)
	// 		continue
	// 	}
	// 	log.Println("ShareDevPlugin PostFilter nodeIP: ", nodeIP)

	// 	freeResources, err := getFreeResources(nodeIP, shareDevState.PodQ)
	// 	if err != nil {
	// 		log.Printf("ShareDevPlugin PostFilter: error getting free resources: %s", err.Error())
	// 		continue
	// 	}
	// 	if len(freeResources) == 0 {
	// 		log.Printf("ShareDevPlugin PostFilter: no free resources")
	// 		continue
	// 	}

	// 	shareDevState.FreeDeviceResourcesPerNode[nodeName] = freeResources
	// 	shareDevState.NodeNameToIP[nodeName] = nodeIP

	// 	log.Printf("ShareDevPlugin node %s %s freeResources: %v", nodeName, nodeIP, freeResources)

	// 	for _, free := range freeResources {
	// 		if podFits(shareDevState.PodQ, free) {
	// 			log.Printf("ShareDevPlugin Filter: pod %s fits device %s", pod.Name, free.DeviceId)

	// 			log.Printf("Filter State: %v", shareDevState)

	// 			return &framework.PostFilterResult{NominatingInfo: &framework.NominatingInfo{NominatedNodeName: nodeName}}, framework.NewStatus(framework.Success)
	// 		}
	// 	}
	// }

	// return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, "no resources available")
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
