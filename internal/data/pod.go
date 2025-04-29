package data

import (
	"fmt"
	"sort"
	"time"
)

type PodMeta struct {
	ID       string    `dynamo:",hash"`                          // metadata.uuid
	TreeID   string    `index:"TreeIndex,hash"`                  // rootID
	TreePath string    `dynamo:",range" index:"TreeIndex,range"` // nodeMeta's ID
	ExpireAt time.Time `json:"-" dynamo:",unixtime"`

	Type       string //pod_meta
	Name       string
	Namespace  string
	StartedAt  time.Time `dynamo:",omitempty"`
	DeletedAt  time.Time `dynamo:",omitempty"`
	FinishedAt time.Time `dynamo:",omitempty"`
	DeletedBy  string
	QOSClass   PodQOSClass
	Snapshots  PodSnapshots `dynamo:"-"`
}

func (p *PodMeta) SetDynamoAttributes(nodeID string) {
	p.TreeID = nodeID
	p.TreePath = nodeID
	p.Type = "pod_meta"
}

type PodSnapshots []*PodSnapshot

func (p *PodSnapshots) Len() int {
	return len(*p)
}

func (p *PodSnapshots) Less(i, j int) bool {
	return (*p)[i].Timestamp.Before((*p)[j].Timestamp)
}

func (p *PodSnapshots) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

// EffectiveAt sorts all PodSnapshots in reverse and returns the first occurring snapshot that's <= the timestamp
func (p *PodSnapshots) EffectiveAt(timestamp time.Time) *PodSnapshot {
	sort.Sort(sort.Reverse(p))
	for _, snapshot := range *p {
		if snapshot.Timestamp.Before(timestamp) || snapshot.Timestamp.Equal(timestamp) {
			return snapshot
		}
	}

	return nil
}

type PodSnapshot struct {
	ID       string    `dynamo:",hash"`                          // metadata.uuid-timestamp
	TreeID   string    `index:"TreeIndex,hash"`                  // rootID
	TreePath string    `dynamo:",range" index:"TreeIndex,range"` // podMeta's path + podMeta's ID
	ExpireAt time.Time `json:"-" dynamo:",unixtime"`

	Type                string // pod_snapshot
	Timestamp           time.Time
	Status              PodPhase
	InitContainers      []*ContainerSnapshot
	EphemeralContainers []*ContainerSnapshot
	Containers          []*ContainerSnapshot
}

func (p *PodSnapshot) SetDynamoAttributes(nodeID, podID string) {
	p.ID = fmt.Sprintf("%s_%s", podID, p.Timestamp)
	p.TreeID = nodeID
	p.TreePath = fmt.Sprintf("%s#%s", nodeID, podID)
	p.Type = "pod_snapshot"
}

type ContainerSnapshot struct {
	ContainerID  string
	Name         string
	Image        string
	ImageID      string
	Ready        bool
	RestartCount int64
	StartedAt    time.Time
	Running      bool
	Resources    ContainerResources
	State        ContainerState
	LastState    ContainerState
}

type ContainerResources struct {
	Requests ContainerResource
	Limits   ContainerResource
}

type ContainerResource struct {
	Cpu              string
	Memory           string
	EphemeralStorage string
}

type ContainerState struct {
	ExitCode   int64
	StartedAt  time.Time
	FinishedAt time.Time
	Reason     string
}
