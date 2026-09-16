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

package instancetype

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/digitalocean/godo"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/apis/v1alpha1"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/do"
)

type Provider interface {
	List(context.Context, *v1alpha1.DONodeClass) ([]*cloudprovider.InstanceType, error)
}

var _ Provider = (*DefaultProvider)(nil)

type DefaultProvider struct {
	client *godo.Client
	region string

	mu    sync.RWMutex
	types []*cloudprovider.InstanceType
}

func NewDefaultProvider(client *godo.Client, region string) *DefaultProvider {
	return &DefaultProvider{client: client, region: region}
}

func (p *DefaultProvider) Refresh(ctx context.Context) error {
	opts, _, err := p.client.Kubernetes.GetOptions(ctx)
	if err != nil {
		return fmt.Errorf("getting kubernetes options: %w", err)
	}
	sizes, err := do.Paginate(ctx, p.client.Sizes.List)
	if err != nil {
		return fmt.Errorf("listing droplet sizes: %w", err)
	}
	sizeBySlug := lo.KeyBy(sizes, func(s godo.Size) string { return s.Slug })

	var types []*cloudprovider.InstanceType
	skipped := 0
	for _, ks := range opts.Sizes {
		if ks == nil || ks.Slug == "" {
			continue
		}
		sz, ok := sizeBySlug[ks.Slug]
		if !ok {
			skipped++
			continue
		}
		if !sz.Available {
			continue
		}
		if len(sz.Regions) > 0 && !lo.Contains(sz.Regions, p.region) {
			continue
		}
		types = append(types, NewInstanceType(ks.Slug, sz, p.region))
	}

	p.mu.Lock()
	p.types = types
	p.mu.Unlock()

	log.FromContext(ctx).Info("discovered DOKS instance types", "count", len(types), "skipped", skipped, "region", p.region)
	return nil
}

func (p *DefaultProvider) List(_ context.Context, _ *v1alpha1.DONodeClass) ([]*cloudprovider.InstanceType, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.types) == 0 {
		return nil, fmt.Errorf("no instance types discovered")
	}
	out := make([]*cloudprovider.InstanceType, len(p.types))
	copy(out, p.types)
	return out, nil
}

func NewInstanceType(slug string, size godo.Size, region string) *cloudprovider.InstanceType {
	return &cloudprovider.InstanceType{
		Name: slug,
		Requirements: scheduling.NewRequirements(
			scheduling.NewRequirement(corev1.LabelInstanceTypeStable, corev1.NodeSelectorOpIn, slug),
			scheduling.NewRequirement(corev1.LabelArchStable, corev1.NodeSelectorOpIn, karpv1.ArchitectureAmd64),
			scheduling.NewRequirement(corev1.LabelOSStable, corev1.NodeSelectorOpIn, string(corev1.Linux)),
			scheduling.NewRequirement(corev1.LabelTopologyRegion, corev1.NodeSelectorOpIn, region),
			scheduling.NewRequirement(corev1.LabelTopologyZone, corev1.NodeSelectorOpIn, region),
			scheduling.NewRequirement(karpv1.CapacityTypeLabelKey, corev1.NodeSelectorOpIn, karpv1.CapacityTypeOnDemand),
		),
		Offerings: cloudprovider.Offerings{
			{
				Requirements: scheduling.NewRequirements(
					scheduling.NewRequirement(karpv1.CapacityTypeLabelKey, corev1.NodeSelectorOpIn, karpv1.CapacityTypeOnDemand),
					scheduling.NewRequirement(corev1.LabelTopologyZone, corev1.NodeSelectorOpIn, region),
				),
				Price:     size.PriceHourly,
				Available: true,
			},
		},
		Capacity: corev1.ResourceList{
			corev1.ResourceCPU:              resource.MustParse(strconv.Itoa(size.Vcpus)),
			corev1.ResourceMemory:           resource.MustParse(fmt.Sprintf("%dMi", size.Memory)),
			corev1.ResourcePods:             resource.MustParse("110"),
			corev1.ResourceEphemeralStorage: resource.MustParse(fmt.Sprintf("%dGi", size.Disk)),
		},
		Overhead: &cloudprovider.InstanceTypeOverhead{
			KubeReserved: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("80m"),
				corev1.ResourceMemory:           resource.MustParse("256Mi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
		},
	}
}
