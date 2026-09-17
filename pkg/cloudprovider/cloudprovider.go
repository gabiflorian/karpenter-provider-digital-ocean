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

package cloudprovider

import (
	"context"
	"fmt"
	"strings"

	"github.com/awslabs/operatorpkg/status"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/events"
	"sigs.k8s.io/karpenter/pkg/utils/resources"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/apis"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/apis/v1alpha1"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instance"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instancetype"
)

var _ cloudprovider.CloudProvider = (*CloudProvider)(nil)

type CloudProvider struct {
	kubeClient           client.Client
	recorder             events.Recorder
	instanceTypeProvider instancetype.Provider
	instanceProvider     instance.Provider
}

func New(
	instanceTypeProvider instancetype.Provider,
	instanceProvider instance.Provider,
	recorder events.Recorder,
	kubeClient client.Client,
) *CloudProvider {
	return &CloudProvider{
		instanceTypeProvider: instanceTypeProvider,
		instanceProvider:     instanceProvider,
		recorder:             recorder,
		kubeClient:           kubeClient,
	}
}

func (c *CloudProvider) Create(ctx context.Context, nodeClaim *karpv1.NodeClaim) (*karpv1.NodeClaim, error) {
	nodeClass, err := c.resolveNodeClassFromNodeClaim(ctx, nodeClaim)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, cloudprovider.NewInsufficientCapacityError(fmt.Errorf("resolving nodeclass, %w", err))
		}
		return nil, fmt.Errorf("resolving nodeclass, %w", err)
	}
	if ready := nodeClass.StatusConditions().Get(status.ConditionReady); ready != nil && ready.IsFalse() {
		return nil, cloudprovider.NewNodeClassNotReadyError(fmt.Errorf("%s", ready.Message))
	}

	instanceTypes, err := c.instanceTypeProvider.List(ctx, nodeClass)
	if err != nil {
		return nil, cloudprovider.NewCreateError(fmt.Errorf("resolving instance types, %w", err), "InstanceTypeResolutionFailed", "Error resolving instance types")
	}

	inst, err := c.instanceProvider.Create(ctx, nodeClass, nodeClaim, instanceTypes)
	if err != nil {
		return nil, err
	}
	log.FromContext(ctx).Info("created DOKS node", "dropletID", inst.DropletID, "poolID", inst.PoolID, "size", inst.Type)
	return c.instanceToNodeClaim(inst, findInstanceType(instanceTypes, inst.Type)), nil
}

func (c *CloudProvider) Delete(ctx context.Context, nodeClaim *karpv1.NodeClaim) error {
	id, err := parseDropletID(nodeClaim.Status.ProviderID)
	if err != nil {
		return cloudprovider.NewNodeClaimNotFoundError(err)
	}
	if err := c.instanceProvider.Delete(ctx, id); err != nil {
		return err
	}
	log.FromContext(ctx).Info("deleted DOKS node", "dropletID", id)
	return nil
}

func (c *CloudProvider) Get(ctx context.Context, providerID string) (*karpv1.NodeClaim, error) {
	id, err := parseDropletID(providerID)
	if err != nil {
		return nil, cloudprovider.NewNodeClaimNotFoundError(err)
	}
	inst, err := c.instanceProvider.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.instanceToNodeClaim(inst, nil), nil
}

func (c *CloudProvider) List(ctx context.Context) ([]*karpv1.NodeClaim, error) {
	instances, err := c.instanceProvider.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing instances, %w", err)
	}
	out := make([]*karpv1.NodeClaim, 0, len(instances))
	for _, inst := range instances {
		out = append(out, c.instanceToNodeClaim(inst, nil))
	}
	return out, nil
}

func (c *CloudProvider) GetInstanceTypes(ctx context.Context, nodePool *karpv1.NodePool) ([]*cloudprovider.InstanceType, error) {
	var nodeClass *v1alpha1.DONodeClass
	if nodePool != nil && nodePool.Spec.Template.Spec.NodeClassRef != nil {
		nc, err := c.resolveNodeClassFromNodePool(ctx, nodePool)
		if err != nil {
			return nil, fmt.Errorf("resolving node class, %w", err)
		}
		nodeClass = nc
	}
	return c.instanceTypeProvider.List(ctx, nodeClass)
}

func (c *CloudProvider) IsDrifted(_ context.Context, _ *karpv1.NodeClaim) (cloudprovider.DriftReason, error) {
	return "", nil
}

func (c *CloudProvider) RepairPolicies() []cloudprovider.RepairPolicy {
	return nil
}

func (c *CloudProvider) Name() string {
	return "digitalocean"
}

func (c *CloudProvider) GetSupportedNodeClasses() []status.Object {
	return []status.Object{&v1alpha1.DONodeClass{}}
}

func (c *CloudProvider) resolveNodeClassFromNodeClaim(ctx context.Context, nodeClaim *karpv1.NodeClaim) (*v1alpha1.DONodeClass, error) {
	if nodeClaim.Spec.NodeClassRef == nil {
		return nil, errors.NewNotFound(schema.GroupResource{Group: apis.Group, Resource: "donodeclasses"}, "")
	}
	nodeClass := &v1alpha1.DONodeClass{}
	if err := c.kubeClient.Get(ctx, types.NamespacedName{Name: nodeClaim.Spec.NodeClassRef.Name}, nodeClass); err != nil {
		return nil, err
	}
	if !nodeClass.DeletionTimestamp.IsZero() {
		return nil, errors.NewNotFound(schema.GroupResource{Group: apis.Group, Resource: "donodeclasses"}, nodeClass.Name)
	}
	return nodeClass, nil
}

func (c *CloudProvider) resolveNodeClassFromNodePool(ctx context.Context, nodePool *karpv1.NodePool) (*v1alpha1.DONodeClass, error) {
	nodeClass := &v1alpha1.DONodeClass{}
	if err := c.kubeClient.Get(ctx, types.NamespacedName{Name: nodePool.Spec.Template.Spec.NodeClassRef.Name}, nodeClass); err != nil {
		return nil, err
	}
	if !nodeClass.DeletionTimestamp.IsZero() {
		return nil, errors.NewNotFound(schema.GroupResource{Group: apis.Group, Resource: "donodeclasses"}, nodeClass.Name)
	}
	return nodeClass, nil
}

func (c *CloudProvider) instanceToNodeClaim(inst *instance.Instance, instanceType *cloudprovider.InstanceType) *karpv1.NodeClaim {
	nodeClaim := &karpv1.NodeClaim{}
	labels := map[string]string{
		corev1.LabelTopologyZone:       inst.Region,
		corev1.LabelTopologyRegion:     inst.Region,
		karpv1.CapacityTypeLabelKey:    karpv1.CapacityTypeOnDemand,
		corev1.LabelInstanceTypeStable: inst.Type,
	}
	if np, ok := tagLookup(inst.Tags, v1alpha1.TagNodePool); ok {
		labels[karpv1.NodePoolLabelKey] = np
	}
	if nc, ok := tagLookup(inst.Tags, v1alpha1.TagNodeClass); ok {
		labels[v1alpha1.LabelNodeClass] = nc
	}
	if instanceType != nil {
		for key, req := range instanceType.Requirements {
			if req.Len() == 1 {
				labels[key] = req.Values()[0]
			}
		}
		nodeClaim.Status.Capacity = lo.PickBy(instanceType.Capacity, func(_ corev1.ResourceName, v resource.Quantity) bool {
			return !resources.IsZero(v)
		})
		nodeClaim.Status.Allocatable = lo.PickBy(instanceType.Allocatable(), func(_ corev1.ResourceName, v resource.Quantity) bool {
			return !resources.IsZero(v)
		})
	}
	nodeClaim.Labels = labels
	if !inst.Created.IsZero() {
		nodeClaim.CreationTimestamp = metav1.Time{Time: inst.Created}
	}
	if inst.State == "deleting" || inst.State == "draining" {
		now := metav1.Now()
		nodeClaim.DeletionTimestamp = &now
	}
	nodeClaim.Status.ProviderID = v1alpha1.ProviderPrefix + inst.DropletID
	return nodeClaim
}

func findInstanceType(instanceTypes []*cloudprovider.InstanceType, name string) *cloudprovider.InstanceType {
	it, _ := lo.Find(instanceTypes, func(i *cloudprovider.InstanceType) bool { return i.Name == name })
	return it
}

func parseDropletID(providerID string) (string, error) {
	id := strings.TrimPrefix(providerID, v1alpha1.ProviderPrefix)
	if id == "" || id == providerID {
		return "", fmt.Errorf("invalid provider id %q", providerID)
	}
	return id, nil
}

func tagLookup(tags []string, prefix string) (string, bool) {
	p := prefix + ":"
	for _, t := range tags {
		if strings.HasPrefix(t, p) {
			return strings.TrimPrefix(t, p), true
		}
	}
	return "", false
}
