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
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/godo"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/log"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/scheduling"
	"sigs.k8s.io/karpenter/pkg/utils/resources"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/apis/v1alpha1"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/do"
)

const (
	createDeadline = 10 * time.Minute
	retryDelay     = 5 * time.Second
)

var poolNameSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

type Instance struct {
	DropletID string
	NodeID    string
	PoolID    string
	Name      string
	Type      string
	Region    string
	Created   time.Time
	State     string
	Tags      []string
}

type Provider interface {
	Create(context.Context, *v1alpha1.DONodeClass, *karpv1.NodeClaim, []*cloudprovider.InstanceType) (*Instance, error)
	Get(context.Context, string) (*Instance, error)
	List(context.Context) ([]*Instance, error)
	Delete(context.Context, string) error
}

var _ Provider = (*DefaultProvider)(nil)

type DefaultProvider struct {
	client    *godo.Client
	clusterID string
	region    string
	mu        sync.Map
}

func NewDefaultProvider(client *godo.Client, clusterID, region string) *DefaultProvider {
	return &DefaultProvider{client: client, clusterID: clusterID, region: region}
}

func (p *DefaultProvider) Create(ctx context.Context, nodeClass *v1alpha1.DONodeClass, nodeClaim *karpv1.NodeClaim, instanceTypes []*cloudprovider.InstanceType) (*Instance, error) {
	it, err := cheapestCompatible(instanceTypes, nodeClaim)
	if err != nil {
		return nil, err
	}
	nodePoolName := nodeClaim.Labels[karpv1.NodePoolLabelKey]
	if nodePoolName == "" {
		return nil, cloudprovider.NewCreateError(fmt.Errorf("nodeclaim %s missing %s label", nodeClaim.Name, karpv1.NodePoolLabelKey), "MissingNodePool", "NodeClaim is missing a NodePool label")
	}

	key := nodePoolName + "/" + it.Name
	unlock := p.lock(key)
	defer unlock()

	tags := poolTags(nodePoolName, nodeClass.Name, it.Name, nodeClass.Spec.Tags)
	pool, known, err := p.ensureCapacity(ctx, nodeClass, nodePoolName, it.Name, tags)
	if err != nil {
		return nil, err
	}
	return p.waitForNewNode(ctx, pool.ID, known, it.Name, time.Now().Add(createDeadline))
}

func (p *DefaultProvider) ensureCapacity(ctx context.Context, nodeClass *v1alpha1.DONodeClass, nodePoolName, size string, tags []string) (*godo.KubernetesNodePool, map[string]struct{}, error) {
	pool, err := p.findManagedPool(ctx, nodePoolName, size)
	if err != nil {
		return nil, nil, err
	}

	if pool == nil {
		created, _, err := p.client.Kubernetes.CreateNodePool(ctx, p.clusterID, &godo.KubernetesNodePoolCreateRequest{
			Name:  poolName(nodePoolName, size),
			Size:  size,
			Count: 1,
			Tags:  tags,
			Labels: map[string]string{
				karpv1.NodePoolLabelKey: nodePoolName,
				v1alpha1.LabelNodeClass: nodeClass.Name,
			},
		})
		if err != nil {
			return nil, nil, apiCreateError(err, "NodePoolCreationFailed", "Failed to create DOKS node pool")
		}
		log.FromContext(ctx).Info("created DOKS node pool", "poolID", created.ID, "size", size, "nodepool", nodePoolName)
		return created, dropletIDs(created), nil
	}

	known := dropletIDs(pool)
	if !needsScale(pool) {
		log.FromContext(ctx).Info("waiting for existing DOKS pool capacity",
			"poolID", pool.ID, "size", size, "nodepool", nodePoolName, "count", pool.Count, "ready", len(known))
		return pool, known, nil
	}

	count := pool.Count + 1
	updated, _, err := p.client.Kubernetes.UpdateNodePool(ctx, p.clusterID, pool.ID, &godo.KubernetesNodePoolUpdateRequest{
		Count: godo.PtrTo(count),
	})
	if err != nil {
		return nil, nil, apiCreateError(err, "NodePoolScaleFailed", "Failed to scale DOKS node pool")
	}
	if updated != nil {
		pool = updated
	}
	log.FromContext(ctx).Info("scaled DOKS node pool", "poolID", pool.ID, "size", size, "nodepool", nodePoolName, "count", count)
	return pool, known, nil
}

func (p *DefaultProvider) waitForNewNode(ctx context.Context, poolID string, known map[string]struct{}, size string, deadline time.Time) (*Instance, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, cloudprovider.NewCreateError(fmt.Errorf("timed out waiting for a DOKS node of type %s", size), "NodePoolProvisioning", "Timed out waiting for a DOKS node")
		}

		pool, _, err := p.client.Kubernetes.GetNodePool(ctx, p.clusterID, poolID)
		if err != nil {
			log.FromContext(ctx).Info("waiting for DOKS node", "poolID", poolID, "error", err.Error())
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
			continue
		}
		for _, n := range pool.Nodes {
			if inst := nodeToInstance(n, pool, p.region); inst != nil {
				if _, exists := known[inst.DropletID]; !exists {
					return inst, nil
				}
			}
		}
		log.FromContext(ctx).Info("waiting for DOKS node", "poolID", poolID, "size", size, "count", pool.Count, "ready", len(dropletIDs(pool)))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryDelay):
		}
	}
}

func (p *DefaultProvider) Get(ctx context.Context, dropletID string) (*Instance, error) {
	instances, err := p.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, inst := range instances {
		if inst.DropletID == dropletID {
			return inst, nil
		}
	}
	return nil, cloudprovider.NewNodeClaimNotFoundError(fmt.Errorf("droplet %s", dropletID))
}

func (p *DefaultProvider) List(ctx context.Context) ([]*Instance, error) {
	pools, err := do.Paginate(ctx, func(ctx context.Context, opt *godo.ListOptions) ([]*godo.KubernetesNodePool, *godo.Response, error) {
		return p.client.Kubernetes.ListNodePools(ctx, p.clusterID, opt)
	})
	if err != nil {
		return nil, fmt.Errorf("listing node pools: %w", err)
	}
	var out []*Instance
	for _, pool := range pools {
		if !isManagedPool(pool) {
			continue
		}
		for _, n := range pool.Nodes {
			if inst := nodeToInstance(n, pool, p.region); inst != nil {
				out = append(out, inst)
			}
		}
	}
	return out, nil
}

func (p *DefaultProvider) Delete(ctx context.Context, dropletID string) error {
	pools, err := do.Paginate(ctx, func(ctx context.Context, opt *godo.ListOptions) ([]*godo.KubernetesNodePool, *godo.Response, error) {
		return p.client.Kubernetes.ListNodePools(ctx, p.clusterID, opt)
	})
	if err != nil {
		return fmt.Errorf("listing node pools: %w", err)
	}

	var found *godo.KubernetesNodePool
	var node *godo.KubernetesNode
	for _, pool := range pools {
		if !isManagedPool(pool) {
			continue
		}
		for _, n := range pool.Nodes {
			if n != nil && n.DropletID == dropletID {
				found, node = pool, n
				break
			}
		}
		if found != nil {
			break
		}
	}
	if found == nil || node == nil {
		return cloudprovider.NewNodeClaimNotFoundError(fmt.Errorf("droplet %s", dropletID))
	}

	managedCount := 0
	for _, n := range found.Nodes {
		if hasDroplet(n) {
			managedCount++
		}
	}
	if managedCount <= 1 {
		_, err = p.client.Kubernetes.DeleteNodePool(ctx, p.clusterID, found.ID)
		if do.IsNotFound(err) {
			return cloudprovider.NewNodeClaimNotFoundError(err)
		}
		return err
	}

	_, err = p.client.Kubernetes.DeleteNode(ctx, p.clusterID, found.ID, node.ID, &godo.KubernetesNodeDeleteRequest{
		SkipDrain: false,
		Replace:   false,
	})
	if do.IsNotFound(err) {
		return cloudprovider.NewNodeClaimNotFoundError(err)
	}
	return err
}

func (p *DefaultProvider) findManagedPool(ctx context.Context, nodePoolName, size string) (*godo.KubernetesNodePool, error) {
	pools, err := do.Paginate(ctx, func(ctx context.Context, opt *godo.ListOptions) ([]*godo.KubernetesNodePool, *godo.Response, error) {
		return p.client.Kubernetes.ListNodePools(ctx, p.clusterID, opt)
	})
	if err != nil {
		return nil, err
	}
	wantNP := tagValue(v1alpha1.TagNodePool, nodePoolName)
	wantSize := tagValue(v1alpha1.TagSize, size)
	for _, pool := range pools {
		if pool.Size != size || !isManagedPool(pool) {
			continue
		}
		if lo.Contains(pool.Tags, wantNP) && lo.Contains(pool.Tags, wantSize) {
			return pool, nil
		}
	}
	return nil, nil
}

func (p *DefaultProvider) lock(key string) func() {
	mu, _ := p.mu.LoadOrStore(key, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

func cheapestCompatible(instanceTypes []*cloudprovider.InstanceType, nodeClaim *karpv1.NodeClaim) (*cloudprovider.InstanceType, error) {
	reqs := scheduling.NewNodeSelectorRequirementsWithMinValues(nodeClaim.Spec.Requirements...)
	compatible := lo.Filter(instanceTypes, func(it *cloudprovider.InstanceType, _ int) bool {
		if !reqs.IsCompatible(it.Requirements, scheduling.AllowUndefinedWellKnownLabels) {
			return false
		}
		if !resources.Fits(nodeClaim.Spec.Resources.Requests, it.Allocatable()) {
			return false
		}
		return len(it.Offerings.Compatible(reqs).Available()) > 0
	})
	if len(compatible) == 0 {
		return nil, cloudprovider.NewInsufficientCapacityError(fmt.Errorf("no compatible DOKS size for nodeclaim %s", nodeClaim.Name))
	}
	return lo.MinBy(compatible, func(a, b *cloudprovider.InstanceType) bool {
		return a.Offerings.Cheapest().Price < b.Offerings.Cheapest().Price
	}), nil
}

func isManagedPool(pool *godo.KubernetesNodePool) bool {
	return pool != nil && lo.Contains(pool.Tags, v1alpha1.TagManaged)
}

func hasDroplet(n *godo.KubernetesNode) bool {
	return n != nil && n.DropletID != "" && n.DropletID != "0"
}

func dropletIDs(pool *godo.KubernetesNodePool) map[string]struct{} {
	ids := map[string]struct{}{}
	if pool == nil {
		return ids
	}
	for _, n := range pool.Nodes {
		if hasDroplet(n) {
			ids[n.DropletID] = struct{}{}
		}
	}
	return ids
}

func needsScale(pool *godo.KubernetesNodePool) bool {
	if pool == nil {
		return false
	}
	return len(dropletIDs(pool)) >= pool.Count
}

func apiCreateError(err error, reason, message string) error {
	if do.IsQuotaExceeded(err) {
		return cloudprovider.NewInsufficientCapacityError(fmt.Errorf("%s: %w", message, err))
	}
	return cloudprovider.NewCreateError(err, reason, message)
}

func nodeToInstance(n *godo.KubernetesNode, pool *godo.KubernetesNodePool, region string) *Instance {
	if !hasDroplet(n) {
		return nil
	}
	state := ""
	if n.Status != nil {
		state = n.Status.State
	}
	return &Instance{
		DropletID: n.DropletID,
		NodeID:    n.ID,
		PoolID:    pool.ID,
		Name:      n.Name,
		Type:      pool.Size,
		Region:    region,
		Created:   n.CreatedAt,
		State:     state,
		Tags:      pool.Tags,
	}
}

func poolTags(nodePoolName, nodeClassName, size string, extra []string) []string {
	tags := []string{
		v1alpha1.TagManaged,
		tagValue(v1alpha1.TagNodePool, nodePoolName),
		tagValue(v1alpha1.TagNodeClass, nodeClassName),
		tagValue(v1alpha1.TagSize, size),
	}
	for _, t := range extra {
		t = strings.TrimSpace(t)
		if t == "" || strings.HasPrefix(t, "k8s") {
			continue
		}
		tags = append(tags, t)
	}
	return lo.Uniq(tags)
}

func tagValue(prefix, value string) string {
	return prefix + ":" + value
}

func poolName(nodePoolName, size string) string {
	name := "k-" + nodePoolName + "-" + size
	name = strings.ToLower(name)
	name = poolNameSanitizer.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if len(name) > 63 {
		name = name[:63]
		name = strings.Trim(name, "-")
	}
	return name
}
