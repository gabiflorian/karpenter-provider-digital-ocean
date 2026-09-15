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
	"github.com/awslabs/operatorpkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionTypeReady = "Ready"
)

var nodeClassConditionTypes = status.NewReadyConditions()

// DONodeClassStatus contains the observed state of a DONodeClass.
type DONodeClassStatus struct {
	// Conditions represent the latest available observations of the object's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (in *DONodeClass) GetConditions() []status.Condition {
	conditions := make([]status.Condition, len(in.Status.Conditions))
	for i := range in.Status.Conditions {
		conditions[i] = status.Condition(in.Status.Conditions[i])
	}
	return conditions
}

func (in *DONodeClass) SetConditions(conditions []status.Condition) {
	in.Status.Conditions = make([]metav1.Condition, len(conditions))
	for i := range conditions {
		in.Status.Conditions[i] = metav1.Condition(conditions[i])
	}
}

func (in *DONodeClass) StatusConditions(opts ...status.ForOption) status.ConditionSet {
	return nodeClassConditionTypes.For(in, opts...)
}
