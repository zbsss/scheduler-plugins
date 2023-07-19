package sharedev

import (
	"context"
	"fmt"
	"time"

	pb "github.com/zbsss/device-manager/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getFreeResources(nodeIP string, podSpec PodRequestedQuota) ([]FreeDeviceResources, error) {
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
		Vendor: podSpec.Vendor,
		Model:  podSpec.Model,
	})
	if err != nil {
		return nil, err
	}

	freeResources := []FreeDeviceResources{}
	for _, free := range resp.Free {
		freeResources = append(freeResources, FreeDeviceResources{
			DeviceId: free.DeviceId,
			Requests: free.Requests,
			Memory:   free.Memory,
		})
	}

	return freeResources, nil
}

func reservePodQuota(nodeIP, deviceId string, pod PodRequestedQuota) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("%s:%s", nodeIP, deviceManagerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	grpc := pb.NewDeviceManagerClient(conn)

	_, err = grpc.ReservePodQuota(ctx, &pb.RegisterPodQuotaRequest{
		DeviceId: deviceId,
		PodId:    pod.PodId,
		Requests: pod.Requests,
		Memory:   pod.Memory,
		Limit:    pod.Limits,
	})
	return err
}

func unreservePodQuota(nodeIP, deviceId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("%s:%s", nodeIP, deviceManagerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()

	grpc := pb.NewDeviceManagerClient(conn)

	_, err = grpc.UnreservePodQuota(ctx, &pb.RegisterPodQuotaRequest{
		DeviceId: deviceId,
		PodId:    pod.PodId,
	})
	return err
}
