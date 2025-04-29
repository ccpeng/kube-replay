package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/ccpeng/kube-replay/internal/services"
)

func TestReplayer_EffectiveAtSnapshot(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	replayer, err := services.NewReplayer("k8s")
	g.Expect(err).To(gomega.BeNil())

	effectiveAt, _ := time.Parse(time.RFC3339, "2025-04-27T00:00:00Z")

	timedNodeSnapshots, err := replayer.EffectiveAtSnapshot(context.Background(), effectiveAt)
	g.Expect(err).To(gomega.BeNil())

	g.Expect(timedNodeSnapshots).ToNot(gomega.BeNil())
}
