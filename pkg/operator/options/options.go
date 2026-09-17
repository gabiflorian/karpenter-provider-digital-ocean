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

package options

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	coreoptions "sigs.k8s.io/karpenter/pkg/operator/options"
	"sigs.k8s.io/karpenter/pkg/utils/env"
)

func init() {
	coreoptions.Injectables = append(coreoptions.Injectables, &Options{})
}

type optionsKey struct{}

// Options are DigitalOcean-specific flags parsed into context.
type Options struct {
	ClusterName       string
	ClusterID         string
	DigitalOceanToken string
}

func (o *Options) AddFlags(fs *coreoptions.FlagSet) {
	fs.StringVar(&o.ClusterName, "cluster-name", env.WithDefaultString("CLUSTER_NAME", ""), "DOKS cluster name used to resolve the cluster UUID.")
	fs.StringVar(&o.ClusterID, "cluster-id", env.WithDefaultString("CLUSTER_ID", ""), "Optional DOKS cluster UUID override. When set, CLUSTER_NAME lookup is skipped.")
	fs.StringVar(&o.DigitalOceanToken, "digitalocean-token", env.WithDefaultString("DIGITALOCEAN_TOKEN", ""), "DigitalOcean API token (PAT).")
}

func (o *Options) Parse(fs *coreoptions.FlagSet, args ...string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		return fmt.Errorf("parsing flags, %w", err)
	}
	return nil
}

func (o *Options) ToContext(ctx context.Context) context.Context {
	return ToContext(ctx, o)
}

func ToContext(ctx context.Context, opts *Options) context.Context {
	return context.WithValue(ctx, optionsKey{}, opts)
}

func FromContext(ctx context.Context) *Options {
	retval := ctx.Value(optionsKey{})
	if retval == nil {
		return nil
	}
	return retval.(*Options)
}
