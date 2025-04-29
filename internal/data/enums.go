package data

import "strings"

type NodeCondition int

const (
	NodeStateReady NodeCondition = iota
	NodeStateNotReady
	NodeStateUnknown
)

func (n *NodeCondition) String() string {
	switch *n {
	case NodeStateReady:
		return "Ready"
	case NodeStateNotReady:
		return "NotReady"
	default:
		return "Unknown"
	}
}

func StringToNodeCondition(s string) NodeCondition {
	conditionMap := map[string]NodeCondition{
		"ready":    NodeStateReady,
		"notready": NodeStateNotReady,
		"unknown":  NodeStateUnknown,
	}

	v, ok := conditionMap[strings.ToLower(s)]
	if !ok {
		return NodeStateUnknown
	}

	return v
}

type PodPhase int

const (
	PodPhasePending PodPhase = iota
	PodPhaseRunning
	PodPhaseSucceeded
	PodPhaseFailed
	PodPhaseUnknown
)

func (p *PodPhase) String() string {
	switch *p {
	case PodPhasePending:
		return "Pending"
	case PodPhaseRunning:
		return "Running"
	case PodPhaseSucceeded:
		return "Succeeded"
	case PodPhaseFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

func StringToPodPhase(s string) PodPhase {
	phaseMap := map[string]PodPhase{
		"pending":   PodPhasePending,
		"running":   PodPhaseRunning,
		"succeeded": PodPhaseSucceeded,
		"failed":    PodPhaseFailed,
		"unknown":   PodPhaseUnknown,
	}

	v, ok := phaseMap[strings.ToLower(s)]
	if !ok {
		return PodPhaseUnknown
	}

	return v
}

type PodQOSClass int

const (
	PodQOSClassBestEffort PodQOSClass = iota
	PodQOSClassBurstable
	PodQOSClassGuaranteed
	PodQOSClassUnknown
)

func (p *PodQOSClass) String() string {
	switch *p {
	case PodQOSClassBestEffort:
		return "BestEffort"
	case PodQOSClassBurstable:
		return "Burstable"
	case PodQOSClassGuaranteed:
		return "Guaranteed"
	default:
		return "Unknown"
	}
}

func StringToPodQOSClass(s string) PodQOSClass {
	qosClassMap := map[string]PodQOSClass{
		"besteffort": PodQOSClassBestEffort,
		"burstable":  PodQOSClassBurstable,
		"guaranteed": PodQOSClassGuaranteed,
		"unknown":    PodQOSClassUnknown,
	}

	v, ok := qosClassMap[strings.ToLower(s)]
	if !ok {
		return PodQOSClassUnknown
	}

	return v
}
