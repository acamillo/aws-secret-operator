package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AWSSecretSpec defines the desired state of AWSSecret
// +k8s:openapi-gen=true
type AWSSecretSpec struct {
	SecretsManagerRef SecretsManagerRef `json:"secretsManagerRef"`
}

// SecretsManagerRef defines from which SecretsManager Secret the Kubernetes secret is built
// See https://docs.aws.amazon.com/secretsmanager/latest/userguide/terms-concepts.html for the concepts
type SecretsManagerRef struct {
	// SecretId is the SecretId a.k.a `--secret-id` of the SecretsManager secret
	SecretId string `json:"secretId"`
	// VersionId is the VersionId a.k.a `--version-id` of the SecretsManager secret
	VersionId string `json:"versionId"`
}

// AWSSecretStatus defines the observed state of AWSSecret
// +k8s:openapi-gen=true
type AWSSecretStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSSecret is the Schema for the awssecrets API
// +k8s:openapi-gen=true
type AWSSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSSecretSpec   `json:"spec"`
	Status AWSSecretStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AWSSecretList contains a list of AWSSecret
type AWSSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AWSSecret{}, &AWSSecretList{})
}
