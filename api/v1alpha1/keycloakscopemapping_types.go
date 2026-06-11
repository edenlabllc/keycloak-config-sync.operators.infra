package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
)

type RoleMapper struct {
	Name string `json:"name"`
}

// KeycloakScopeMappingSpec defines the desired state of KeycloakClientScopeMapping.
type KeycloakScopeMappingSpec struct {
	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// ConfigRef is reference to Keycloak Config.
	// +required
	ConfigRef common.ConfigRef `json:"configRef"`

	// ClientScope of keycloak client scope.
	// +optional
	ClientScope string `json:"clientScope,omitempty"`

	// +optional
	Client string `json:"client,omitempty"`

	// +optional
	FromClient string `json:"fromClient,omitempty"`

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
	FailureCount int64 `json:"failureCount,omitempty"`

	// +optional
	Error string `json:"error,omitempty"`

	// +optional
	Phase string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Realm",type="string",JSONPath=".spec.realmRef.name",description="Keycloak ref to realm name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Reconciliation phase"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error",description="Resource error"

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

func (in *KeycloakScopeMapping) GetPhase() string {
	return in.Status.Phase
}

func (in *KeycloakScopeMapping) SetPhase(value string) {
	in.Status.Phase = value
}

func (in *KeycloakScopeMapping) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

func (in *KeycloakScopeMapping) GetConfigRef() common.ConfigRef {
	return in.Spec.ConfigRef
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
