package sharedev

import (
	"context"
	"fmt"
	"time"

	pb "github.com/zbsss/device-manager/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (sp *ShareDevPlugin) getFreeResources(nodeIP string, podSpec ShareDevPodSpec) ([]FreeResources, error) {
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

	freeResources := []FreeResources{}
	for _, free := range resp.Free {
		freeResources = append(freeResources, FreeResources{
			DeviceId: free.DeviceId,
			Requests: free.Requests,
			Memory:   free.Memory,
		})
	}

	return freeResources, nil
}
