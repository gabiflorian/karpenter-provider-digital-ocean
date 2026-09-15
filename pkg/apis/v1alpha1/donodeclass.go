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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DONodeClassSpec is the top-level specification for the DigitalOcean Karpenter provider.
// Instance type (Droplet / DOKS size slug) is selected on the Karpenter NodePool via
// node.kubernetes.io/instance-type, not on this object.
type DONodeClassSpec struct {
	// Tags applied to Karpenter-managed DOKS node pools.
	// +optional
	// +listType=set
	Tags []string `json:"tags,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""
// +kubebuilder:resource:path=donodeclasses,scope=Cluster,categories=karpenter,shortName={donc,doncs}
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
type DONodeClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DONodeClassSpec   `json:"spec,omitempty"`
	Status DONodeClassStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DONodeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DONodeClass `json:"items"`
}
