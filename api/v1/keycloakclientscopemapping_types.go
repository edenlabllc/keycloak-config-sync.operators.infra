package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
)

type RoleMapper struct {
	Name string `json:"name"`
}

// KeycloakClientScopeMappingSpec defines the desired state of KeycloakClientScopeMapping.
type KeycloakClientScopeMappingSpec struct {
	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// +required
	FromClient string `json:"fromClient"`

	// ClientScope of keycloak client scope.
	// +optional
	ClientScope string `json:"clientScope,omitempty"`

	// +optional
	Client string `json:"client,omitempty"`

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

// KeycloakClientScopeMappingStatus defines the observed state of KeycloakClientScopeMapping.
type KeycloakClientScopeMappingStatus struct {
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

// KeycloakClientScopeMapping is the Schema for the keycloakclientscopemapping API.
type KeycloakClientScopeMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakClientScopeMappingSpec   `json:"spec,omitempty"`
	Status KeycloakClientScopeMappingStatus `json:"status,omitempty"`
}

func (in *KeycloakClientScopeMapping) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakClientScopeMapping) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakClientScopeMapping) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakClientScopeMapping) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakClientScopeMapping) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// +kubebuilder:object:root=true

// KeycloakClientScopeMappingList contains a list of KeycloakClientScopeMapping.
type KeycloakClientScopeMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakClientScopeMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakClientScopeMapping{}, &KeycloakClientScopeMappingList{})
}
