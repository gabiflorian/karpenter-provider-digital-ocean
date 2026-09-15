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
	"errors"

	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/apis/v1alpha1"
)

var errUnimplemented = errors.New("unimplemented")

// Instance is a DigitalOcean worker that Karpenter manages (a DOKS node / Droplet).
type Instance struct {
	ID   string
	Name string
	Type string
}

type Provider interface {
	Create(context.Context, *v1alpha1.DONodeClass, *karpv1.NodeClaim) (*Instance, error)
	Get(context.Context, string) (*Instance, error)
	List(context.Context) ([]*Instance, error)
	Delete(context.Context, string) error
}

var _ Provider = (*DefaultProvider)(nil)

type DefaultProvider struct{}

func NewDefaultProvider() *DefaultProvider {
	return &DefaultProvider{}
}

func (p *DefaultProvider) Create(context.Context, *v1alpha1.DONodeClass, *karpv1.NodeClaim) (*Instance, error) {
	return nil, errUnimplemented
}

func (p *DefaultProvider) Get(context.Context, string) (*Instance, error) {
	return nil, errUnimplemented
}

func (p *DefaultProvider) List(context.Context) ([]*Instance, error) {
	return nil, nil
}

func (p *DefaultProvider) Delete(context.Context, string) error {
	return errUnimplemented
}
