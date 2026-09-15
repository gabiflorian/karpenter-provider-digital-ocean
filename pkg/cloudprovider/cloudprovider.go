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
	"errors"

	"github.com/awslabs/operatorpkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/events"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/apis/v1alpha1"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instance"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instancetype"
)

var (
	errUnimplemented                             = errors.New("unimplemented")
	_                cloudprovider.CloudProvider = (*CloudProvider)(nil)
)

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

func (c *CloudProvider) Create(_ context.Context, _ *karpv1.NodeClaim) (*karpv1.NodeClaim, error) {
	return nil, cloudprovider.NewCreateError(errUnimplemented, "Unimplemented", "DigitalOcean CloudProvider Create is not implemented")
}

func (c *CloudProvider) Delete(_ context.Context, _ *karpv1.NodeClaim) error {
	return cloudprovider.NewNodeClaimNotFoundError(errUnimplemented)
}

func (c *CloudProvider) Get(_ context.Context, _ string) (*karpv1.NodeClaim, error) {
	return nil, cloudprovider.NewNodeClaimNotFoundError(errUnimplemented)
}

func (c *CloudProvider) List(_ context.Context) ([]*karpv1.NodeClaim, error) {
	return nil, nil
}

func (c *CloudProvider) GetInstanceTypes(_ context.Context, _ *karpv1.NodePool) ([]*cloudprovider.InstanceType, error) {
	return nil, nil
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
