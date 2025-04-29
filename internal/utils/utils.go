package utils

import (
	"github.com/ccpeng/kube-replay/graph/model"
	"github.com/ccpeng/kube-replay/internal/data"
)

func TransformToModelNodeCondition(condition data.NodeCondition) model.NodeCondition {
	switch condition {
	case data.NodeStateReady:
		return model.NodeConditionReady
	case data.NodeStateNotReady:
		return model.NodeConditionNotReady
	default:
		return model.NodeConditionUnknown
	}
}

func TransformToModelPodPhase(phase data.PodPhase) model.PodPhase {
	switch phase {
	case data.PodPhaseRunning:
		return model.PodPhaseRunning
	case data.PodPhasePending:
		return model.PodPhasePending
	case data.PodPhaseFailed:
		return model.PodPhaseFailed
	default:
		return model.PodPhaseUnknown
	}
}

func TransformToModelPodQOSClass(class data.PodQOSClass) model.PodQOSClass {
	switch class {
	case data.PodQOSClassBestEffort:
		return model.PodQOSClassBestEffort
	case data.PodQOSClassBurstable:
		return model.PodQOSClassBurstable
	case data.PodQOSClassGuaranteed:
		return model.PodQOSClassGuaranteed
	default:
		return model.PodQOSClassUnknown
	}
}

func TransformToDataPods(untransformed []*model.PodSnapshotInput) []*data.PodMeta {
	transformed := make([]*data.PodMeta, len(untransformed))

	for i, pod := range untransformed {
		transformed[i] = &data.PodMeta{
			ID:         pod.ID,
			Name:       pod.Name,
			Namespace:  *pod.Namespace,
			StartedAt:  pod.StartedAt,
			DeletedAt:  *pod.DeletedAt,
			FinishedAt: *pod.FinishedAt,
			DeletedBy:  *pod.DeletedBy,
			QOSClass:   data.StringToPodQOSClass(pod.QosClass.String()),
			Snapshots: data.PodSnapshots{
				{
					Timestamp:           pod.Timestamp,
					Status:              data.StringToPodPhase(pod.Status.String()),
					InitContainers:      transformToDataContainers(pod.InitContainers),
					EphemeralContainers: transformToDataContainers(pod.EphemeralContainers),
					Containers:          transformToDataContainers(pod.Containers),
				},
			},
		}
	}

	return transformed
}

func transformToDataContainers(untransformed []*model.ContainerSnapshotInput) []*data.ContainerSnapshot {
	transformed := make([]*data.ContainerSnapshot, len(untransformed))

	for i, container := range untransformed {
		transformed[i] = &data.ContainerSnapshot{
			ContainerID:  container.ContainerID,
			Name:         container.Name,
			Image:        container.Image,
			ImageID:      container.ImageID,
			Ready:        container.Ready,
			RestartCount: *container.RestartCount,
			StartedAt:    container.StartedAt,
			Running:      container.Running,
			Resources: data.ContainerResources{
				Requests: data.ContainerResource{
					Cpu:              *container.Resources.Requests.CPU,
					Memory:           *container.Resources.Requests.Memory,
					EphemeralStorage: *container.Resources.Requests.EphemeralStorage,
				},
				Limits: data.ContainerResource{
					Cpu:              *container.Resources.Limits.CPU,
					Memory:           *container.Resources.Limits.Memory,
					EphemeralStorage: *container.Resources.Limits.EphemeralStorage,
				},
			},
			State: data.ContainerState{
				ExitCode:   *container.State.ExitCode,
				StartedAt:  container.State.StartedAt,
				FinishedAt: *container.State.FinishedAt,
				Reason:     *container.State.Reason,
			},
			LastState: data.ContainerState{
				ExitCode:   *container.LastState.ExitCode,
				StartedAt:  container.LastState.StartedAt,
				FinishedAt: *container.LastState.FinishedAt,
				Reason:     *container.LastState.Reason,
			},
		}
	}

	return transformed
}
