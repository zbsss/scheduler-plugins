package sharedev

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (sp *ShareDevPlugin) allocateNewDevice(vendor, model string) (string, error) {
	cli := sp.handle.ClientSet().AppsV1().Deployments("default")

	//TODO: should we create a Deployment or maybe a Pod would be enough?
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
		return "", fmt.Errorf("error creating deployment: %s", err.Error())
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
			return "", fmt.Errorf("timeout waiting for pod to be running")
		case event := <-watch.ResultChan():
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return "", fmt.Errorf("unexpected type: %T", event.Object)
			}

			if pod.Status.Phase == corev1.PodRunning {
				return pod.Spec.NodeName, nil
			}
		}
	}
}
