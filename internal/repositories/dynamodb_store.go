package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/guregu/dynamo/v2"

	"github.com/ccpeng/kube-replay/internal/data"
)

const ttl = time.Hour * 24 * 90 // 90 days TTL

type treeStore struct {
	table dynamo.Table
}

func (t *treeStore) GetAll(ctx context.Context) ([]*data.NodeMeta, error) {
	var items []dynamo.Item
	err := t.table.Scan().Filter("TreePath = ?", "root").All(ctx, &items)
	if err != nil {
		return nil, err
	}

	var nodeMetas []*data.NodeMeta
	for _, item := range items {
		v, ok := item["ID"].(*types.AttributeValueMemberS)
		if !ok {
			return nil, errors.New("failed to cast item ID attribute to string")
		}

		nodeMeta, err := t.Get(ctx, v.Value)
		if err != nil {
			return nil, err
		}

		nodeMetas = append(nodeMetas, nodeMeta)
	}

	return nodeMetas, nil
}

func (t *treeStore) Get(ctx context.Context, nodeID string) (*data.NodeMeta, error) {
	var items []dynamo.Item
	err := t.table.Get("TreeID", nodeID).Index("TreeIndex").All(ctx, &items)
	if err != nil {
		return nil, err
	}

	var nodeMeta *data.NodeMeta
	var nodeSnapshots []*data.NodeSnapshot
	var podMetas []*data.PodMeta
	var podSnapshots []*data.PodSnapshot

	for _, item := range items {
		v, ok := item["Type"].(*types.AttributeValueMemberS)
		if !ok {
			return nil, errors.New("failed to cast item Type attribute to string")
		}
		switch v.Value {
		case "node_meta":
			err := dynamo.UnmarshalItem(item, &nodeMeta)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal node_meta: %w", err)
			}
		case "node_snapshot":
			var nodeSnapshot data.NodeSnapshot
			err := dynamo.UnmarshalItem(item, &nodeSnapshot)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal node_snapshot: %w", err)
			}

			nodeSnapshots = append(nodeSnapshots, &nodeSnapshot)
		case "pod_meta":
			var podMeta data.PodMeta
			err := dynamo.UnmarshalItem(item, &podMeta)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal pod_meta: %w", err)
			}

			podMetas = append(podMetas, &podMeta)
		case "pod_snapshot":
			var podSnapshot data.PodSnapshot
			err := dynamo.UnmarshalItem(item, &podSnapshot)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal pod_snapshot: %w", err)
			}

			podSnapshots = append(podSnapshots, &podSnapshot)
		}
	}

	if nodeMeta == nil {
		return nil, errors.New("failed to find items to build node meta aka. tree root")
	}

	nodeMeta.Snapshots = nodeSnapshots

	for _, podSnapshot := range podSnapshots {
		for i, podMeta := range podMetas {
			idParts := strings.Split(podSnapshot.ID, "_")
			if idParts[0] == podMeta.ID {
				podMetas[i].Snapshots = append(podMetas[i].Snapshots, podSnapshot)
			}
		}
	}

	nodeMeta.Pods = podMetas

	return nodeMeta, nil
}

func (t *treeStore) Upsert(ctx context.Context, nodeMeta *data.NodeMeta) error {
	if len(nodeMeta.ID) == 0 {
		return errors.New("unable to upsert NodeMeta vertex since node ID is empty")
	}
	nodeMeta.ExpireAt = time.Now().Add(ttl)

	nodeMeta.SetDynamoAttributes()
	err := t.table.Put(nodeMeta).Run(ctx)
	if err != nil {
		return fmt.Errorf("error when upserting NodeMeta vertex: %w", err)
	}

	err = t.UpsertNodeSnapshots(ctx, nodeMeta.ID, nodeMeta.Snapshots)
	if err != nil {
		return err
	}

	err = t.UpsertPodMetas(ctx, nodeMeta.ID, nodeMeta.Pods)
	if err != nil {
		return err
	}

	return nil
}

func (t *treeStore) UpsertNodeSnapshots(ctx context.Context, nodeID string, nodeSnapshots []*data.NodeSnapshot) error {
	batchSize := len(nodeSnapshots)
	items := make([]interface{}, batchSize)

	for i, nodeSnapshot := range nodeSnapshots {
		if nodeSnapshot.Timestamp.IsZero() {
			return errors.New("unable to upsert NodeSnapshot vertex since timestamp is zero")
		}
		nodeSnapshot.ExpireAt = time.Now().Add(ttl)

		nodeSnapshot.SetDynamoAttributes(nodeID)
		items[i] = nodeSnapshot
	}

	if len(items) > 0 {
		wc, err := t.table.Batch().Write().Put(items...).Run(ctx)
		if err != nil {
			return fmt.Errorf("error when upserting NodeSnapshot vertexes: %w", err)
		}

		if wc != batchSize {
			return fmt.Errorf("while upsetting node snapshots: persisted count (%v) != length of snapshots (%v)", wc, batchSize)
		}
	}

	return nil
}

func (t *treeStore) UpsertPodMetas(ctx context.Context, nodeID string, podMetas []*data.PodMeta) error {
	batchSize := len(podMetas)
	items := make([]interface{}, batchSize)

	for i, podMeta := range podMetas {
		if len(podMeta.ID) == 0 {
			return errors.New("unable to upsert PodMetas vertex since pod meta ID is empty")
		}
		podMeta.ExpireAt = time.Now().Add(ttl)

		podMeta.SetDynamoAttributes(nodeID)
		items[i] = podMeta

		err := t.UpsertPodSnapshots(ctx, nodeID, podMeta.ID, podMeta.Snapshots)
		if err != nil {
			return err
		}
	}

	if len(items) > 0 {
		wc, err := t.table.Batch().Write().Put(items...).Run(ctx)
		if err != nil {
			return fmt.Errorf("error when upserting PodMeta vertexes: %w", err)
		}

		if wc != batchSize {
			return fmt.Errorf("while upsetting pod meta vertexes: persisted count (%v) != length of pod metas (%v)", wc, batchSize)
		}
	}

	return nil
}

func (t *treeStore) UpsertPodSnapshots(ctx context.Context, nodeID string, podID string, podSnapshots []*data.PodSnapshot) error {
	batchSize := len(podSnapshots)
	items := make([]interface{}, batchSize)

	for i, podSnapshot := range podSnapshots {
		if podSnapshot.Timestamp.IsZero() {
			return errors.New("unable to upsert PodSnapshot vertex since timestamp is zero")
		}
		podSnapshot.ExpireAt = time.Now().Add(ttl)

		podSnapshot.SetDynamoAttributes(nodeID, podID)
		items[i] = podSnapshot
	}

	if len(items) > 0 {
		wc, err := t.table.Batch().Write().Put(items...).Run(ctx)
		if err != nil {
			return fmt.Errorf("error when upserting PodSnapshot vertexes: %w", err)
		}

		if wc != batchSize {
			return fmt.Errorf("while upsetting pod snapshot vertexes: persisted count (%v) != length of pod snapshots (%v)", wc, batchSize)
		}
	}

	return nil
}

func (t *treeStore) UpdateNodeMetaAttributes(ctx context.Context, nodeID string, updates map[string]interface{}) error {
	update := t.table.Update("ID", nodeID).Range("TreePath", "root")

	for k, v := range updates {
		update = update.Set(k, v)
	}

	return update.Run(ctx)
}

func (t *treeStore) UpdatePodMetaAttributes(ctx context.Context, nodeID, podID string, updates map[string]interface{}) error {
	update := t.table.Update("ID", podID).Range("TreePath", nodeID)

	for k, v := range updates {
		update = update.Set(k, v)
	}

	return update.Run(ctx)
}

func NewStore(cfg aws.Config, table string) Store {
	db := dynamo.New(cfg)

	return &treeStore{
		table: db.Table(table),
	}
}
