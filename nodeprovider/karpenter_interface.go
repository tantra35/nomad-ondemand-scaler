package nodeprovider

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"github.com/tantra35/nomad-ondemand-scaler/nodeprovider/karpenterprovidergrpc"
)

type K8sKapenterProviderPluginInterface interface {
	ListInstances(string) ([]string, error)
	AddInstances(context.Context, string, int, *karpenterprovidergrpc.AddInstancesSpec) ([]string, string, error)
	RemoveInstances(string, []string) error
}

type K8sKapenterProviderPlugin struct {
	plugin.Plugin

	Impl K8sKapenterProviderPluginInterface
}

func (p *K8sKapenterProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return fmt.Errorf("server not allowed here")
}

func (p *K8sKapenterProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &K8sKapenterProviderClient{
		client: karpenterprovidergrpc.NewKarpenterServiceClient(c),
	}, nil
}

type K8sKapenterProviderClient struct {
	client karpenterprovidergrpc.KarpenterServiceClient
}

func (k *K8sKapenterProviderClient) ListInstances(_poolName string) ([]string, error) {
	resp, lerr := k.client.ListInstances(context.Background(), &karpenterprovidergrpc.ListInstancesRequest{
		PoolName: _poolName,
	})
	if lerr != nil {
		return nil, lerr
	}

	return resp.Instanseids, nil
}

func (k *K8sKapenterProviderClient) AddInstances(_ctx context.Context, _poolName string, _count int, _spec *karpenterprovidergrpc.AddInstancesSpec) ([]string, string, error) {
	lresp, lerr := k.client.AddInstances(_ctx, &karpenterprovidergrpc.AddInstancesRequest{
		PoolName: _poolName,
		Count:    int32(_count),
		Spec:     _spec,
	})
	if lerr != nil {
		return nil, "", lerr
	}

	return lresp.Instanseids, lresp.Reason, nil
}

func (k *K8sKapenterProviderClient) RemoveInstances(_poolName string, _instanses []string) error {
	_, lerr := k.client.RemoveInstances(context.Background(), &karpenterprovidergrpc.DeleteInstancesRequest{
		PoolName:    _poolName,
		Instanseids: _instanses,
	})

	return lerr
}
