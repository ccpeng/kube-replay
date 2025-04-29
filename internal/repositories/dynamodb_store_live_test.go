package repositories_test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/onsi/gomega"

	"github.com/ccpeng/kube-replay/internal/data"
	"github.com/ccpeng/kube-replay/internal/repositories"
)

func TestDynamodbStore(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	k8sStore := repositories.NewStore(cfg, "k8s")

	nodeSnap, _ := time.Parse(time.RFC3339, "2025-04-21T23:22:32Z")
	podStart, _ := time.Parse(time.RFC3339, "2025-04-21T21:31:15Z")
	podSnap, _ := time.Parse(time.RFC3339, "2025-04-21T23:22:32Z")

	tree := data.NodeMeta{
		ID:                      "eb2d7ef7-99f9-41f4-8d91-658582413cf2",
		Name:                    "ip-10-59-178-234.us-west-2.compute.internal",
		ProviderID:              "aws:///us-west-2b/i-0c0cfe71229fd536a",
		ContainerRuntimeVersion: "containerd://1.7.22",
		KernelVersion:           "5.10.234-225.910.amzn2.x86_64",
		KubeProxyVersion:        "v1.30.4-eks-a737599",
		OsImage:                 "Amazon Linux 2",
		OperatingSystem:         "linux",
		MachineID:               "0eeaa0c87c6044b88b30fe9c7f16d8f5",
		SystemUUID:              "ec23a7a9-bab4-6908-b845-ee154aa73c4c",
		BootID:                  "55f9af62-b345-42bb-a5bc-3e8884512bd2",
		Roles:                   []string{"node"},
		Snapshots: data.NodeSnapshots{
			{
				Timestamp: nodeSnap,
				State: data.NodeState{
					Condition: data.NodeStateReady,
					Capacity: data.NodeCapacity{
						Cpu:              "16",
						Memory:           "64445364Ki",
						EphemeralStorage: "268423148Ki",
						Pods:             160,
					},
					Allocatable: data.NodeCapacity{
						Cpu:              "15890m",
						Memory:           "60017588Ki",
						EphemeralStorage: "260048296346",
						Pods:             160,
					},
					Taints: []*data.Taint{
						{
							Key:    "karpenter.sh/disrupted",
							Effect: "NoScheudule",
						},
					},
				},
			},
		},
		Pods: []*data.PodMeta{
			{
				ID:        "3acc87f7-0a55-4c75-b550-c62605f97b4e",
				Name:      "t4i-asset-data-producer-rollout-6f8d876774-qlpk4",
				Namespace: "t4i-wft-t4iassetdataproducer-e2e",
				StartedAt: podStart,
				QOSClass:  data.PodQOSClassBurstable,
				Snapshots: []*data.PodSnapshot{
					{
						Timestamp:           podSnap,
						Status:              0,
						InitContainers:      nil,
						EphemeralContainers: nil,
						Containers: []*data.ContainerSnapshot{
							{
								ContainerID:  "",
								Name:         "istio-proxy",
								Image:        "docker.com/strategic/services/service-mesh/service/proxyv2:master-1.23.3-71221d0",
								ImageID:      "",
								Ready:        false,
								RestartCount: 0,
								StartedAt:    time.Now(),
								Running:      false,
								Resources: data.ContainerResources{
									Requests: data.ContainerResource{
										Cpu:              "500m",
										Memory:           "234042164",
										EphemeralStorage: "1Gi",
									},
									Limits: data.ContainerResource{
										Cpu:              "1",
										Memory:           "351063246",
										EphemeralStorage: "8Gi",
									},
								},
								State: data.ContainerState{
									StartedAt: time.Now(),
								},
								LastState: data.ContainerState{},
							},
							{
								ContainerID:  "containerd://2d28cb2414eb13f6b69b900598e34428c8e818903369acf2cb1baaf877fe1f2a",
								Name:         "app",
								Image:        "docker.com/t4i-wft/t4i-asset-data-producer/service/t4i-asset-data-producer:PR-6-7bdcbf8",
								ImageID:      "docker.com/t4i-wft/t4i-asset-data-producer/service/t4i-asset-data-producer@sha256:19fa4d9c7bae87c05ddee0f9cfb2499091a28ff6aa0449086f1394301347962b",
								Ready:        false,
								RestartCount: 0,
								StartedAt:    time.Now(),
								Running:      false,
								Resources: data.ContainerResources{
									Requests: data.ContainerResource{
										Cpu:              "810m",
										Memory:           "1038683533",
										EphemeralStorage: "1Gi",
									},
									Limits: data.ContainerResource{
										Cpu:              "1620m",
										Memory:           "1038683533",
										EphemeralStorage: "8Gi",
									},
								},
								State: data.ContainerState{
									StartedAt:  time.Now(),
									FinishedAt: time.Time{},
									Reason:     "",
								},
								LastState: data.ContainerState{},
							},
						},
					},
				},
			},
		},
	}
	err = k8sStore.Upsert(context.Background(), &tree)
	g.Expect(err).Should(gomega.BeNil())

	nodeUpdates := map[string]interface{}{
		"Architecture": "amd64",
	}
	err = k8sStore.UpdateNodeMetaAttributes(context.Background(), "eb2d7ef7-99f9-41f4-8d91-658582413cf2", nodeUpdates)
	g.Expect(err).Should(gomega.BeNil())

	podUpdates := map[string]interface{}{
		"DeletedBy": "JohnSmith",
	}
	err = k8sStore.UpdatePodMetaAttributes(context.Background(), "eb2d7ef7-99f9-41f4-8d91-658582413cf2", "3acc87f7-0a55-4c75-b550-c62605f97b4e", podUpdates)
	g.Expect(err).Should(gomega.BeNil())

	nodeMeta, err := k8sStore.Get(context.Background(), "eb2d7ef7-99f9-41f4-8d91-658582413cf2")
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(nodeMeta.ID).Should(gomega.Equal("eb2d7ef7-99f9-41f4-8d91-658582413cf2"))
	g.Expect(nodeMeta.Architecture).Should(gomega.Equal("amd64"))
	g.Expect(len(nodeMeta.Snapshots)).Should(gomega.BeEquivalentTo(1))
	g.Expect(nodeMeta.Snapshots[0].State.Capacity.Cpu).Should(gomega.Equal("16"))
	g.Expect(len(nodeMeta.Pods)).Should(gomega.BeEquivalentTo(1))
	g.Expect(nodeMeta.Pods[0].DeletedBy).Should(gomega.Equal("JohnSmith"))
	g.Expect(len(nodeMeta.Pods[0].Snapshots)).Should(gomega.BeEquivalentTo(1))
	g.Expect(len(nodeMeta.Pods[0].Snapshots[0].Containers)).Should(gomega.BeEquivalentTo(2))

	nodeMetas, err := k8sStore.GetAll(context.Background())
	g.Expect(err).Should(gomega.BeNil())
	g.Expect(len(nodeMetas)).Should(gomega.BeEquivalentTo(1))
	nodeMeta1 := nodeMetas[0]
	g.Expect(nodeMeta1).Should(gomega.Equal(nodeMeta))
}
