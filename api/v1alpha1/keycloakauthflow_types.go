package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
)

// KeycloakAuthFlowSpec defines the desired state of KeycloakAuthFlow.
type KeycloakAuthFlowSpec struct {
	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// ConfigRef is reference to Keycloak Config.
	// +required
	ConfigRef common.ConfigRef `json:"configRef"`

	// Alias is display name for authentication flow.
	Alias string `json:"alias"`

	// Description is description for authentication flow.
	// +optional
	Description string `json:"description,omitempty"`

	// ProviderID for root auth flow and provider for child auth flows.
	ProviderID string `json:"providerId"`

	// TopLevel is true if this is root auth flow.
	TopLevel bool `json:"topLevel"`

	// BuiltIn is true if this is built-in auth flow.
	BuiltIn bool `json:"builtIn"`

	// AuthenticationExecutions is list of authentication executions for this auth flow.
	// +nullable
	// +optional
	AuthenticationExecutions []AuthenticationExecution `json:"authenticationExecutions,omitempty"`

	// ParentName is name of parent auth flow.
	// +optional
	ParentName string `json:"parentName,omitempty"`

	// ChildType is type for auth flow if it has a parent, available options: basic-flow, form-flow
	// +optional
	ChildType string `json:"childType,omitempty"`

	// ChildRequirement is requirement for child execution. Available options: REQUIRED, ALTERNATIVE, DISABLED, CONDITIONAL.
	// +optional
	ChildRequirement string `json:"childRequirement,omitempty"`

	// ReconciliationStrategy is a strategy to reconcile client.
	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`
}

// AuthenticationExecution defines keycloak authentication execution.
type AuthenticationExecution struct {
	// Authenticator is name of authenticator.
	// +optional
	Authenticator string `json:"authenticator,omitempty"`

	// AuthenticatorConfig is configuration for authenticator.
	// +nullable
	// +optional
	AuthenticatorConfig *AuthenticatorConfig `json:"authenticatorConfig,omitempty"`

	// AuthenticatorFlow is true if this is auth flow.
	// +optional
	AuthenticatorFlow bool `json:"authenticatorFlow,omitempty"`

	// Priority is priority for this execution. Lower values have higher priority.
	// +optional
	Priority int `json:"priority,omitempty"`

	// Requirement is requirement for this execution. Available options: REQUIRED, ALTERNATIVE, DISABLED, CONDITIONAL.
	// +optional
	Requirement string `json:"requirement,omitempty"`

	// Alias is display name for this execution.
	// +optional
	Alias string `json:"alias,omitempty"`

	// FlowV1 supported version < 23
	// +optional
	// +kubebuilder:default=false
	FlowV1 bool `json:"flowV1,omitempty"`
}

type AuthenticatorConfig struct {
	// Alias is display name for authenticator config.
	// +optional
	Alias string `json:"alias,omitempty"`

	// Config is configuration for authenticator.
	// +optional
	// +nullable.
	Config map[string]string `json:"config,omitempty"`
}

// KeycloakAuthFlowStatus defines the observed state of KeycloakAuthFlow.
type KeycloakAuthFlowStatus struct {
	// +optional
	Error string `json:"error,omitempty"`

	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// ID is the Keycloak internal ID of the auth flow.
	// +optional
	ID string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Realm",type="string",JSONPath=".spec.realmRef.name",description="Keycloak ref to realm name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Reconciliation phase"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error",description="Resource error"

// KeycloakAuthFlow is the Schema for the keycloak authentication flow API.
type KeycloakAuthFlow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakAuthFlowSpec   `json:"spec,omitempty"`
	Status KeycloakAuthFlowStatus `json:"status,omitempty"`
}

func (in *KeycloakAuthFlow) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

func (in *KeycloakAuthFlow) GetConfigRef() common.ConfigRef {
	return in.Spec.ConfigRef
}

func (in *KeycloakAuthFlow) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakAuthFlow) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakAuthFlow) GetPhase() string {
	return in.Status.Phase
}

func (in *KeycloakAuthFlow) SetPhase(value string) {
	in.Status.Phase = value
}

// +kubebuilder:object:root=true

// KeycloakAuthFlowList contains a list of KeycloakAuthFlow.
type KeycloakAuthFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakAuthFlow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakAuthFlow{}, &KeycloakAuthFlowList{})
}
