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
	"fmt"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/karpenter/pkg/operator"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/do"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/operator/options"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instance"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instancetype"
)

type Operator struct {
	*operator.Operator
	InstanceProvider     instance.Provider
	InstanceTypeProvider instancetype.Provider
	ClusterID            string
	ClusterName          string
	Region               string
}

func NewOperator(ctx context.Context, op *operator.Operator) (context.Context, *Operator) {
	opts := options.FromContext(ctx)
	if opts == nil || opts.DigitalOceanToken == "" {
		log.FromContext(ctx).Error(fmt.Errorf("DIGITALOCEAN_TOKEN is required"), "missing DigitalOcean token")
		os.Exit(1)
	}

	client := do.NewClient(opts.DigitalOceanToken)
	cluster, err := do.ResolveCluster(ctx, client, opts.ClusterID, opts.ClusterName, op.GetConfig().Host)
	if err != nil {
		log.FromContext(ctx).Error(err, "resolving DOKS cluster")
		os.Exit(1)
	}

	log.FromContext(ctx).Info("discovered DOKS cluster",
		"id", cluster.ID,
		"name", cluster.Name,
		"region", cluster.RegionSlug,
	)

	instanceTypeProvider := instancetype.NewDefaultProvider(client, cluster.RegionSlug)
	if err := instanceTypeProvider.Refresh(ctx); err != nil {
		log.FromContext(ctx).Error(err, "discovering DOKS instance types")
		os.Exit(1)
	}

	return ctx, &Operator{
		Operator:             op,
		InstanceProvider:     instance.NewDefaultProvider(client, cluster.ID, cluster.RegionSlug),
		InstanceTypeProvider: instanceTypeProvider,
		ClusterID:            cluster.ID,
		ClusterName:          cluster.Name,
		Region:               cluster.RegionSlug,
	}
}
