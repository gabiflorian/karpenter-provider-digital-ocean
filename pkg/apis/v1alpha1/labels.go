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

package v1alpha1

import (
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/apis"
)

func init() {
	karpv1.RestrictedLabelDomains = karpv1.RestrictedLabelDomains.Insert(RestrictedLabelDomains...)
}

const (
	// ProviderPrefix is the DigitalOcean CCM provider ID scheme: digitalocean://<droplet_id>.
	ProviderPrefix = "digitalocean://"

	TerminationFinalizer = apis.Group + "/termination"
	LabelNodeClass       = apis.Group + "/donodeclass"

	// DigitalOcean tags cannot contain '/'. Karpenter-managed DOKS pools use these.
	TagManaged   = "karpenter:managed"
	TagNodePool  = "karpenter:nodepool"
	TagNodeClass = "karpenter:nodeclass"
	TagSize      = "karpenter:size"
)

var (
	RestrictedLabelDomains = []string{
		apis.Group,
	}
)
