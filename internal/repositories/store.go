package repositories

import (
	"context"

	"github.com/ccpeng/kube-replay/internal/data"
)

type Store interface {
	GetAll(ctx context.Context) ([]*data.NodeMeta, error)
	Get(ctx context.Context, nodeID string) (*data.NodeMeta, error)
	Upsert(ctx context.Context, nodeMeta *data.NodeMeta) error
	UpsertNodeSnapshots(ctx context.Context, nodeID string, nodeSnapshots []*data.NodeSnapshot) error
	UpsertPodMetas(ctx context.Context, nodeID string, podMetas []*data.PodMeta) error
	UpsertPodSnapshots(ctx context.Context, nodeID string, podID string, podSnapshots []*data.PodSnapshot) error
	UpdateNodeMetaAttributes(ctx context.Context, nodeID string, updates map[string]interface{}) error
	UpdatePodMetaAttributes(ctx context.Context, nodeID, podID string, updates map[string]interface{}) error
}
