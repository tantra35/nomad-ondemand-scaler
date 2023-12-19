package nodeprovider

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"playrix.com/it/nomad-cluster-scalerv2/nodeprovider/karpenterprovidergrpc"
)

func TestListInstances(t *testing.T) {
	conn, err := grpc.Dial("127.0.0.1:20220", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("did not connect: %v", err)
	}

	client := karpenterprovidergrpc.NewKarpenterServiceClient(conn)
	linstancesresp, lerr := client.ListInstances(context.Background(), &karpenterprovidergrpc.ListInstancesRequest{Clustername: "test"})
	if lerr != nil {
		t.Fatalf("could not list instances: %v", lerr)
	}

	t.Logf("instances: %v", linstancesresp.Instanseids)
}
