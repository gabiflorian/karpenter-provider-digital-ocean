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

package do

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/digitalocean/godo"
)

func NewClient(token string) *godo.Client {
	return godo.NewFromToken(strings.TrimSpace(token))
}

func Paginate[T any](ctx context.Context, list func(context.Context, *godo.ListOptions) ([]T, *godo.Response, error)) ([]T, error) {
	opt := &godo.ListOptions{Page: 1, PerPage: 200}
	var all []T
	for {
		page, resp, err := list(ctx, opt)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}
		current, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, err
		}
		opt.Page = current + 1
	}
	return all, nil
}

func IsNotFound(err error) bool {
	var er *godo.ErrorResponse
	if !errors.As(err, &er) {
		return false
	}
	return er.Response != nil && er.Response.StatusCode == http.StatusNotFound
}

func IsQuotaExceeded(err error) bool {
	var er *godo.ErrorResponse
	if !errors.As(err, &er) {
		return false
	}
	msg := strings.ToLower(er.Message)
	if er.Response != nil && er.Response.StatusCode == http.StatusUnprocessableEntity {
		return strings.Contains(msg, "limit") || strings.Contains(msg, "exceed") || strings.Contains(msg, "quota")
	}
	return strings.Contains(msg, "droplet limit")
}

func clusterIDFromAPIServer(host string) string {
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.Split(host, "/")[0]
	host = strings.Split(host, ":")[0]
	const suffix = ".k8s.ondigitalocean.com"
	if strings.HasSuffix(host, suffix) {
		return strings.TrimSuffix(host, suffix)
	}
	return ""
}

func ResolveCluster(ctx context.Context, client *godo.Client, clusterID, clusterName, apiServerHost string) (*godo.KubernetesCluster, error) {
	if clusterID == "" {
		clusterID = clusterIDFromAPIServer(apiServerHost)
	}
	if clusterID != "" {
		cluster, _, err := client.Kubernetes.Get(ctx, clusterID)
		if err != nil {
			return nil, fmt.Errorf("getting cluster %s: %w", clusterID, err)
		}
		return cluster, nil
	}

	clusters, err := Paginate(ctx, client.Kubernetes.List)
	if err != nil {
		return nil, fmt.Errorf("listing clusters: %w", err)
	}

	if clusterName != "" {
		var matches []*godo.KubernetesCluster
		for _, c := range clusters {
			if c.Name == clusterName {
				matches = append(matches, c)
			}
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no DOKS cluster named %q", clusterName)
		}
		if len(matches) > 1 {
			return nil, fmt.Errorf("multiple DOKS clusters named %q", clusterName)
		}
		return matches[0], nil
	}

	if len(clusters) == 1 {
		return clusters[0], nil
	}
	if len(clusters) == 0 {
		return nil, fmt.Errorf("no DOKS clusters found for this token")
	}
	return nil, fmt.Errorf("CLUSTER_NAME or CLUSTER_ID is required when the token can see %d clusters", len(clusters))
}
