package data

import (
	"fmt"
	"sort"
	"time"
)

type NodeMeta struct {
	ID       string    `dynamo:",hash"`                          // metadata.uuid
	TreeID   string    `index:"TreeIndex,hash"`                  // same as ID
	TreePath string    `dynamo:",range" index:"TreeIndex,range"` // root
	ExpireAt time.Time `json:"-" dynamo:",unixtime"`

	Type                    string // node_meta
	Name                    string
	ProviderID              string
	Architecture            string
	ContainerRuntimeVersion string
	KernelVersion           string
	KubeletVersion          string
	KubeProxyVersion        string
	OsImage                 string
	OperatingSystem         string
	MachineID               string
	SystemUUID              string
	BootID                  string
	Roles                   []string
	Snapshots               NodeSnapshots `dynamo:"-"`
	Pods                    []*PodMeta    `dynamo:"-"`
}

func (n *NodeMeta) SetDynamoAttributes() {
	n.TreeID = n.ID
	n.TreePath = "root"
	n.Type = "node_meta"
}

type NodeSnapshots []*NodeSnapshot

func (n *NodeSnapshots) Len() int {
	return len(*n)
}

func (n *NodeSnapshots) Less(i, j int) bool {
	return (*n)[i].Timestamp.Before((*n)[j].Timestamp)
}

func (n *NodeSnapshots) Swap(i, j int) {
	(*n)[i], (*n)[j] = (*n)[j], (*n)[i]
}

// EffectiveAt sorts all NodeSnapshots in reverse and returns the first occurring snapshot that's <= the timestamp
func (n *NodeSnapshots) EffectiveAt(timestamp time.Time) *NodeSnapshot {
	sort.Sort(sort.Reverse(n))
	for _, snapshot := range *n {
		if snapshot.Timestamp.Before(timestamp) || snapshot.Timestamp.Equal(timestamp) {
			return snapshot
		}
	}

	return nil
}

type NodeSnapshot struct {
	ID       string    `dynamo:",hash"`                          // metadata.uuid-timestamp
	TreeID   string    `index:"TreeIndex,hash"`                  // rootID
	TreePath string    `dynamo:",range" index:"TreeIndex,range"` // nodeMeta's ID
	ExpireAt time.Time `json:"-" dynamo:",unixtime"`

	Type      string // node_snapshot
	Timestamp time.Time
	State     NodeState
}

func (n *NodeSnapshot) SetDynamoAttributes(nodeID string) {
	n.ID = fmt.Sprintf("%s_%s", nodeID, n.Timestamp)
	n.TreeID = nodeID
	n.TreePath = nodeID
	n.Type = "node_snapshot"
}

type NodeState struct {
	Condition     NodeCondition
	Capacity      NodeCapacity
	Allocatable   NodeCapacity
	Taints        []*Taint
	Unschedulable bool
}

type NodeCapacity struct {
	Cpu              string
	Memory           string
	EphemeralStorage string
	Pods             int64
}
type Taint struct {
	Key       string
	Value     string
	Effect    string
	TimeAdded time.Time
}
