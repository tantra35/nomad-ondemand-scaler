package nodeprovider

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
)

type AnyNodeProvider struct {
	logger hclog.Logger
}

func NewAnyNodeProvider() (INodeProvider, error) {
	return &AnyNodeProvider{
		logger: hclog.L().Named("AnyNodeProvider"),
	}, nil
}

func (c *AnyNodeProvider) IsNodeExists(_nomadNode *nomad.Node) bool {
	lastTime := _nomadNode.Events[len(_nomadNode.Events)-1].Timestamp
	c.logger.Debug(fmt.Sprintf("IsNodeExists for nomad node: %s with laststevent time: %s", _nomadNode.ID, lastTime))

	return true
}

func (c *AnyNodeProvider) RemoveNode(_nomadNode []*nomad.Node) error {
	return fmt.Errorf("RemoveNode not possible for AnyNodeProvider")
}

func (c *AnyNodeProvider) UpdateNode(_ctx context.Context, _nodes []*structs.Node, _totalcount int32) error {
	return fmt.Errorf("UpdateNode not possible for AnyNodeProvider")
}
