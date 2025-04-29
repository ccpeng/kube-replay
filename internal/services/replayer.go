package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/ccpeng/kube-replay/graph/model"
	"github.com/ccpeng/kube-replay/internal/data"
	"github.com/ccpeng/kube-replay/internal/repositories"
	"github.com/ccpeng/kube-replay/internal/utils"
)

func NewReplayer(clusterName string) (Replayer, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	return &replayer{
		store: repositories.NewStore(cfg, clusterName),
	}, nil
}

type Replayer interface {
	RecordNodeSnapshot(ctx context.Context, snapshot *model.NodeSnapshotInput) error
	RecordPodSnapshots(ctx context.Context, snapshots []*model.PodSnapshotInput) error
	EventfulSnapshots(ctx context.Context, beginAt, endAt time.Time) ([]*model.TimedNodeSnapshots, error)
	IntervalSnapshots(ctx context.Context, beginAt, endAt time.Time, intervalInSec int64) ([]*model.TimedNodeSnapshots, error)
	EffectiveAtSnapshot(ctx context.Context, effectiveAt time.Time) (*model.TimedNodeSnapshots, error)
}
type replayer struct {
	store repositories.Store
}

// RecordNodeSnapshot TODO: enhance so it won't override
// RecordNodeSnapshot persists the node snapshot
func (r *replayer) RecordNodeSnapshot(ctx context.Context, snapshot *model.NodeSnapshotInput) error {
	var taints = make([]*data.Taint, 0)
	for _, taint := range snapshot.State.Taints {
		taints = append(taints, &data.Taint{
			Key:       taint.Key,
			Value:     *taint.Value,
			Effect:    taint.Effect,
			TimeAdded: *taint.TimeAdded,
		})
	}

	err := r.store.Upsert(ctx, &data.NodeMeta{
		ID:                      snapshot.ID,
		Name:                    snapshot.Name,
		ProviderID:              *snapshot.ProviderID,
		Architecture:            snapshot.Info.Architecture,
		ContainerRuntimeVersion: snapshot.Info.ContainerRuntimeVersion,
		KernelVersion:           snapshot.Info.KernelVersion,
		KubeletVersion:          snapshot.Info.KubeletVersion,
		KubeProxyVersion:        snapshot.Info.KubeProxyVersion,
		OsImage:                 snapshot.Info.OsImage,
		OperatingSystem:         *snapshot.Info.OperatingSystem,
		MachineID:               snapshot.Info.MachineID,
		SystemUUID:              snapshot.Info.SystemUUID,
		BootID:                  snapshot.Info.BootID,
		Roles:                   snapshot.Roles,
		Snapshots: data.NodeSnapshots{
			{
				ID:        snapshot.ID,
				Timestamp: snapshot.Timestamp,
				State: data.NodeState{
					Condition: data.StringToNodeCondition(snapshot.State.Status.String()),
					Capacity: data.NodeCapacity{
						Cpu:              snapshot.State.Capacity.CPU,
						Memory:           snapshot.State.Capacity.Memory,
						EphemeralStorage: snapshot.State.Capacity.EphemeralStorage,
						Pods:             *snapshot.State.Capacity.Pods,
					},
					Allocatable: data.NodeCapacity{
						Cpu:              snapshot.State.Allocatable.CPU,
						Memory:           snapshot.State.Allocatable.Memory,
						EphemeralStorage: snapshot.State.Allocatable.EphemeralStorage,
						Pods:             *snapshot.State.Allocatable.Pods,
					},
					Taints:        taints,
					Unschedulable: *snapshot.State.Unschedulable,
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return r.RecordPodSnapshots(ctx, snapshot.Pods)
}

// RecordPodSnapshots persists the pod snapshots (theoretically can be associated across different nodes)
func (r *replayer) RecordPodSnapshots(ctx context.Context, snapshots []*model.PodSnapshotInput) error {
	// map of nodeID to list of pod snapshots
	nodesPodsMap := map[string][]*model.PodSnapshotInput{}

	for _, snapshot := range snapshots {
		if len(nodesPodsMap[snapshot.NodeID]) == 0 {
			nodesPodsMap[snapshot.NodeID] = []*model.PodSnapshotInput{snapshot}
		} else {
			nodesPodsMap[snapshot.NodeID] = append(nodesPodsMap[snapshot.NodeID], snapshot)
		}
	}

	for nodeID, pods := range nodesPodsMap {
		if err := r.store.UpsertPodMetas(ctx, nodeID, utils.TransformToDataPods(pods)); err != nil {
			return err
		}
	}

	return nil
}

// EventfulSnapshots returns snapshots timestamped at every captured node or pod snapshot
func (r *replayer) EventfulSnapshots(ctx context.Context, beginAt, endAt time.Time) ([]*model.TimedNodeSnapshots, error) {
	var times []time.Time

	nodes, err := r.store.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get all nodes in cluster: %v", err)
	}

	for _, node := range nodes {
		for _, nodeSnapshots := range node.Snapshots {
			if (nodeSnapshots.Timestamp.After(beginAt) || nodeSnapshots.Timestamp.Equal(beginAt)) &&
				(nodeSnapshots.Timestamp.Before(endAt) || nodeSnapshots.Timestamp.Equal(endAt)) {
				times = append(times, nodeSnapshots.Timestamp)
			}
		}

		for _, pod := range node.Pods {
			for _, podSnapshots := range pod.Snapshots {
				if (podSnapshots.Timestamp.After(beginAt) || podSnapshots.Timestamp.Equal(beginAt)) &&
					(podSnapshots.Timestamp.Before(endAt) || podSnapshots.Timestamp.Equal(endAt)) {
					times = append(times, podSnapshots.Timestamp)
				}
			}
		}
	}

	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})

	eventfulTimedSnapshots := make([]*model.TimedNodeSnapshots, 0)
	for _, effectiveAt := range times {
		timedNodeSnapshots, err := r.EffectiveAtSnapshot(ctx, effectiveAt)
		if err != nil {
			return nil, err
		}

		eventfulTimedSnapshots = append(eventfulTimedSnapshots, timedNodeSnapshots)
	}

	return eventfulTimedSnapshots, nil
}

// IntervalSnapshots returns effective snapshots at every regular interval between beginAt and endAt
func (r *replayer) IntervalSnapshots(ctx context.Context, beginAt, endAt time.Time, intervalInSec int64) ([]*model.TimedNodeSnapshots, error) {
	var times []time.Time
	for t := beginAt; t.Before(endAt) || t.Equal(endAt); t = t.Add(time.Duration(intervalInSec) * time.Second) {
		times = append(times, t)
	}

	intervaledTimedSnapshots := make([]*model.TimedNodeSnapshots, 0)
	for _, effectiveAt := range times {
		timedNodeSnapshots, err := r.EffectiveAtSnapshot(ctx, effectiveAt)
		if err != nil {
			return nil, err
		}

		intervaledTimedSnapshots = append(intervaledTimedSnapshots, timedNodeSnapshots)
	}

	return intervaledTimedSnapshots, nil
}

// EffectiveAtSnapshot returns the snapshots that are true/effective at given timestamp
func (r *replayer) EffectiveAtSnapshot(ctx context.Context, effectiveAt time.Time) (*model.TimedNodeSnapshots, error) {
	timedSnapshots := model.TimedNodeSnapshots{
		Timestamp: effectiveAt,
	}

	var nodeSnapshots []*model.NodeSnapshot

	nodes, err := r.store.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get all nodes in cluster: %v", err)
	}

	for _, node := range nodes {
		nodeInTime := node.Snapshots.EffectiveAt(effectiveAt)
		nodeSnapshotTaints := []*model.NodeTaint{}

		for _, taint := range nodeInTime.State.Taints {
			nodeSnapshotTaints = append(nodeSnapshotTaints, &model.NodeTaint{
				Key:       taint.Key,
				Value:     &taint.Value,
				Effect:    taint.Effect,
				TimeAdded: &taint.TimeAdded,
			})
		}

		podSnapshots := []*model.PodSnapshot{}
		for _, pod := range node.Pods {
			podInTime := pod.Snapshots.EffectiveAt(effectiveAt)
			podSnapshotInitContainers := containerSnapshots(podInTime.InitContainers)
			podSnapshotEphemeralContainers := containerSnapshots(podInTime.EphemeralContainers)
			podSnapshotContainers := containerSnapshots(podInTime.Containers)

			podSnapshot := model.PodSnapshot{
				ID:                  pod.ID,
				Timestamp:           podInTime.Timestamp,
				Name:                pod.Name,
				Namespace:           &pod.Namespace,
				Status:              utils.TransformToModelPodPhase(podInTime.Status),
				StartedAt:           pod.StartedAt,
				DeletedAt:           &pod.DeletedAt,
				FinishedAt:          &pod.FinishedAt,
				DeletedBy:           &pod.DeletedBy,
				QosClass:            utils.TransformToModelPodQOSClass(pod.QOSClass),
				InitContainers:      podSnapshotInitContainers,
				Containers:          podSnapshotContainers,
				EphemeralContainers: podSnapshotEphemeralContainers,
			}
			podSnapshots = append(podSnapshots, &podSnapshot)
		}

		nodeSnapshot := &model.NodeSnapshot{
			ID:         node.ID,
			Timestamp:  nodeInTime.Timestamp,
			Name:       node.Name,
			Roles:      node.Roles,
			ProviderID: &node.ProviderID,
			Info: &model.NodeInfo{
				Architecture:            node.Architecture,
				ContainerRuntimeVersion: node.ContainerRuntimeVersion,
				KernelVersion:           node.KernelVersion,
				KubeletVersion:          node.KubeletVersion,
				KubeProxyVersion:        node.KubeProxyVersion,
				OsImage:                 node.OsImage,
				OperatingSystem:         &node.OperatingSystem,
				MachineID:               node.MachineID,
				SystemUUID:              node.SystemUUID,
				BootID:                  node.BootID,
			},
			State: &model.NodeState{
				Status: utils.TransformToModelNodeCondition(nodeInTime.State.Condition),
				Capacity: &model.NodeCapacity{
					CPU:              nodeInTime.State.Capacity.Cpu,
					Memory:           nodeInTime.State.Capacity.Memory,
					EphemeralStorage: nodeInTime.State.Capacity.EphemeralStorage,
					Pods:             &nodeInTime.State.Capacity.Pods,
				},
				Allocatable: &model.NodeCapacity{
					CPU:              nodeInTime.State.Allocatable.Cpu,
					Memory:           nodeInTime.State.Allocatable.Memory,
					EphemeralStorage: nodeInTime.State.Allocatable.EphemeralStorage,
					Pods:             &nodeInTime.State.Allocatable.Pods,
				},
				Taints:        nodeSnapshotTaints,
				Unschedulable: &nodeInTime.State.Unschedulable,
			},
			Pods: podSnapshots,
		}

		nodeSnapshots = append(nodeSnapshots, nodeSnapshot)
	}

	timedSnapshots.Nodes = nodeSnapshots
	return &timedSnapshots, nil
}

func containerSnapshots(containers []*data.ContainerSnapshot) []*model.ContainerSnapshot {
	result := []*model.ContainerSnapshot{}

	for _, container := range containers {
		result = append(result, &model.ContainerSnapshot{
			ContainerID: container.ContainerID,
			Name:        container.Name,
			Image:       container.Image,
			ImageID:     container.ImageID,
			Resources: &model.ContainerResources{
				Requests: &model.ContainerResource{
					CPU:              &container.Resources.Requests.Cpu,
					Memory:           &container.Resources.Requests.Memory,
					EphemeralStorage: &container.Resources.Requests.EphemeralStorage,
				},
				Limits: &model.ContainerResource{
					CPU:              &container.Resources.Limits.Cpu,
					Memory:           &container.Resources.Limits.Memory,
					EphemeralStorage: &container.Resources.Limits.EphemeralStorage,
				},
			},
			Ready:        container.Ready,
			RestartCount: &container.RestartCount,
			StartedAt:    container.StartedAt,
			Running:      container.Running,
			State: &model.ContainerState{
				ExitCode:   &container.State.ExitCode,
				StartedAt:  container.State.StartedAt,
				FinishedAt: &container.State.FinishedAt,
				Reason:     &container.State.Reason,
			},
			LastState: &model.ContainerLastState{
				ExitCode:   &container.LastState.ExitCode,
				StartedAt:  &container.LastState.StartedAt,
				FinishedAt: &container.LastState.FinishedAt,
				Reason:     &container.LastState.Reason,
			},
		})
	}

	return result
}
