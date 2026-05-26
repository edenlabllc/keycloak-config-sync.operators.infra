package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
)

// KeycloakScopeMappingSpec defines the desired state of KeycloakClientScopeMapping.
type KeycloakScopeMappingSpec struct {
	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// ClientScope of keycloak client scope.
	// +required
	ClientScope string `json:"clientScope"`

	// Description is a description of client scope.
	// +optional
	Description string `json:"description,omitempty"`

	// Roles
	Roles []RoleMapper `json:"roles"`

	// ReconciliationStrategy is a strategy to reconcile client.
	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`
}

// KeycloakScopeMappingStatus defines the observed state of KeycloakClientScopeMapping.
type KeycloakScopeMappingStatus struct {
	// +optional
	ID string `json:"id,omitempty"`

	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconciliation status"

// KeycloakScopeMapping is the Schema for the keycloakscopemapping API.
type KeycloakScopeMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakScopeMappingSpec   `json:"spec,omitempty"`
	Status KeycloakScopeMappingStatus `json:"status,omitempty"`
}

func (in *KeycloakScopeMapping) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakScopeMapping) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakScopeMapping) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakScopeMapping) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakScopeMapping) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// +kubebuilder:object:root=true

// KeycloakScopeMappingList contains a list of KeycloakScopeMapping.
type KeycloakScopeMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakScopeMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakScopeMapping{}, &KeycloakScopeMappingList{})
}
