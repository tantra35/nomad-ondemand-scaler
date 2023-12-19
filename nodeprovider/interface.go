package nodeprovider

import (
	"context"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/nomad/structs"
)

type INodeProvider interface {
	IsNodeExists(_nomadNode *nomad.Node) bool
	RemoveNode(_nomadNode []*nomad.Node) error //TODO возможно добавить передачу контекста, чтобы была возможность отмены
	UpdateNode(_сtx context.Context, _nodes []*structs.Node, _totalcount int32) error
}
