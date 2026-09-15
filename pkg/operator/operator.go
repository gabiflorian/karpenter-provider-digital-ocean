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

package operator

import (
	"context"

	"sigs.k8s.io/karpenter/pkg/operator"

	_ "github.com/digitalocean/karpenter-provider-digital-ocean/pkg/operator/options" // register Injectable flags
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instance"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instancetype"
)

// Operator wraps Karpenter core with DigitalOcean stub providers.
type Operator struct {
	*operator.Operator
	InstanceProvider     instance.Provider
	InstanceTypeProvider instancetype.Provider
}

func NewOperator(ctx context.Context, op *operator.Operator) (context.Context, *Operator) {
	return ctx, &Operator{
		Operator:             op,
		InstanceProvider:     instance.NewDefaultProvider(),
		InstanceTypeProvider: instancetype.NewDefaultProvider(),
	}
}
