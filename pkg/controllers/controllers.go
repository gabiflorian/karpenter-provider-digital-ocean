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

package controllers

import (
	"context"

	"github.com/awslabs/operatorpkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	"sigs.k8s.io/karpenter/pkg/events"

	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instance"
	"github.com/digitalocean/karpenter-provider-digital-ocean/pkg/providers/instancetype"
)

func NewControllers(
	_ context.Context,
	_ manager.Manager,
	_ events.Recorder,
	_ cloudprovider.CloudProvider,
	_ instance.Provider,
	_ instancetype.Provider,
) []controller.Controller {
	return nil
}
