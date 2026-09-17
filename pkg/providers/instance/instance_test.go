/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instance

import (
	"testing"

	"github.com/digitalocean/godo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instancetype"
)

func TestCheapestCompatibleWithHostnamePlaceholder(t *testing.T) {
	it := instancetype.NewInstanceType("s-2vcpu-4gb", godo.Size{
		Slug:        "s-2vcpu-4gb",
		Memory:      4096,
		Vcpus:       2,
		Disk:        80,
		PriceHourly: 0.036,
		Available:   true,
	}, "fra1")

	nodeClaim := &karpv1.NodeClaim{
		Spec: karpv1.NodeClaimSpec{
			Requirements: []karpv1.NodeSelectorRequirementWithMinValues{
				{Key: corev1.LabelInstanceTypeStable, Operator: corev1.NodeSelectorOpIn, Values: []string{"s-2vcpu-4gb"}},
				{Key: corev1.LabelHostname, Operator: corev1.NodeSelectorOpIn, Values: []string{"hostname-placeholder-0005"}},
				{Key: corev1.LabelArchStable, Operator: corev1.NodeSelectorOpIn, Values: []string{"amd64"}},
				{Key: corev1.LabelOSStable, Operator: corev1.NodeSelectorOpIn, Values: []string{"linux"}},
				{Key: karpv1.CapacityTypeLabelKey, Operator: corev1.NodeSelectorOpIn, Values: []string{"on-demand"}},
				{Key: corev1.LabelTopologyZone, Operator: corev1.NodeSelectorOpIn, Values: []string{"fra1"}},
				{Key: karpv1.NodePoolLabelKey, Operator: corev1.NodeSelectorOpIn, Values: []string{"default"}},
			},
			Resources: karpv1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("310m"),
					corev1.ResourceMemory: resource.MustParse("332Mi"),
					corev1.ResourcePods:   resource.MustParse("5"),
				},
			},
		},
	}
	nodeClaim.Name = "default-zhqh5"

	got, err := cheapestCompatible([]*cloudprovider.InstanceType{it}, nodeClaim)
	if err != nil {
		t.Fatalf("expected compatible size, got %v", err)
	}
	if got.Name != "s-2vcpu-4gb" {
		t.Fatalf("got %s", got.Name)
	}
}

func TestNeedsScaleWaitsWhileNodesAreProvisioning(t *testing.T) {
	pool := &godo.KubernetesNodePool{
		Count: 1,
		Nodes: []*godo.KubernetesNode{
			{ID: "pending", DropletID: ""},
			{ID: "zero", DropletID: "0"},
		},
	}
	if needsScale(pool) {
		t.Fatal("expected to wait for the in-flight node instead of scaling")
	}
	if got := len(dropletIDs(pool)); got != 0 {
		t.Fatalf("expected no ready droplets, got %d", got)
	}
}

func TestNeedsScaleWhenPoolIsFullyReady(t *testing.T) {
	pool := &godo.KubernetesNodePool{
		Count: 2,
		Nodes: []*godo.KubernetesNode{
			{ID: "a", DropletID: "111"},
			{ID: "b", DropletID: "222"},
		},
	}
	if !needsScale(pool) {
		t.Fatal("expected to scale when every desired node already has a droplet")
	}
	ids := dropletIDs(pool)
	if _, ok := ids["111"]; !ok {
		t.Fatal("missing droplet 111")
	}
	if _, ok := ids["222"]; !ok {
		t.Fatal("missing droplet 222")
	}
}

func TestPoolTaintsOmitUnregistered(t *testing.T) {
	got := poolTaints(&karpv1.NodeClaim{
		Spec: karpv1.NodeClaimSpec{
			Taints: []corev1.Taint{{Key: "dedicated", Value: "gpu", Effect: corev1.TaintEffectNoSchedule}},
		},
	})
	if len(got) != 1 || got[0].Key != "dedicated" {
		t.Fatalf("got %+v", got)
	}
	if poolTaints(nil) != nil {
		t.Fatal("expected nil taints without a nodeclaim")
	}
	if !hasUnregisteredTaint(&godo.KubernetesNodePool{Taints: []godo.Taint{{
		Key:    karpv1.UnregisteredTaintKey,
		Effect: string(corev1.TaintEffectNoExecute),
	}}}) {
		t.Fatal("expected to detect unregistered taint")
	}
}
