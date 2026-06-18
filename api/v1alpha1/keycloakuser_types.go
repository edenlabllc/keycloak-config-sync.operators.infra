package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
)

// KeycloakUserSpec defines the desired state of KeycloakUser.
type KeycloakUserSpec struct {
	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// ConfigRef is reference to Keycloak Config.
	// +required
	ConfigRef common.ConfigRef `json:"configRef"`

	// Username is a username in keycloak.
	Username string `json:"username"`

	// Email is a user email.
	// +optional
	Email string `json:"email,omitempty"`

	// FirstName is a user first name.
	// +optional
	FirstName string `json:"firstName,omitempty"`

	// LastName is a user last name.
	// +optional
	LastName string `json:"lastName,omitempty"`

	// Enabled is a user enabled flag.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// totp is a user flag.
	// +optional
	// +kubebuilder:default=false
	TOTP bool `json:"totp,omitempty"`

	// EmailVerified is a user email verified flag.
	// +optional
	EmailVerified bool `json:"emailVerified,omitempty"`

	// RequiredUserActions is required action when user log in, example: CONFIGURE_TOTP, UPDATE_PASSWORD, UPDATE_PROFILE, VERIFY_EMAIL.
	// +nullable
	// +optional
	RequiredUserActions []string `json:"requiredUserActions,omitempty"`

	// Roles is a list of roles assigned to user.
	// +nullable
	// +optional
	Roles []string `json:"roles,omitempty"`

	// ClientRoles is a list of client roles assigned to user.
	// +nullable
	// +optional
	ClientRoles []UserClientRole `json:"clientRoles,omitempty"`

	// Groups is a list of groups assigned to user.
	// Each entry is either a plain group name (e.g. "developers") or a slash-separated
	// path (e.g. "/developers", "/parent/child"), where each segment represents a level
	// in the group hierarchy.
	// +nullable
	// +optional
	// +kubebuilder:example={"developers","/parent/child"}
	Groups []string `json:"groups,omitempty"`

	// Attributes is a map of user attributes.
	// Deprecated: Use AttributesV2 instead.
	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// AttributesV2 is a map of service account attributes.
	// Each attribute can have multiple values.
	// +nullable
	// +optional
	AttributesV2 map[string][]string `json:"attributesV2,omitempty"`

	// ReconciliationStrategy is a strategy for reconciliation. Possible values: full, addOnly.
	// Default value: full. If set to addOnly, user will be created only if it does not exist. If user exists, it will not be updated.
	// If set to full, user will be created if it does not exist, or updated if it exists.
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`

	// Password is a user password. Allows to keep user password within Custom Resource. For security concerns, it is recommended to use PasswordSecret instead.
	// Deperecated: use PasswordSecret instead.
	// +optional
	Password string `json:"password,omitempty"`

	// KeepResource, when set to false, results in the deletion of the KeycloakRealmUser Custom Resource (CR)
	// from the cluster after the corresponding user is created in Keycloak. The user will continue to exist in Keycloak.
	// When set to true, the CR will not be deleted after processing.
	// +optional
	// +kubebuilder:default=true
	KeepResource bool `json:"keepResource"`

	// PasswordSecret defines Kubernetes secret Name and Key, which holds User secret.
	// +nullable
	// +optional
	PasswordSecret *PasswordSecret `json:"passwordSecret,omitempty"`

	// IdentityProviders is a list of identity providers aliases linked to the user.
	// +nullable
	// +optional
	IdentityProviders *[]string `json:"identityProviders,omitempty"`
}

// PasswordSecret defines struct which contains reference to secret name and key.
type PasswordSecret struct {
	// Name is the name of the secret.
	Name string `json:"name"`

	// Key is the key in the secret.
	Key string `json:"key"`

	// Temporary indicates whether the password is temporary.
	// +optional
	// +kubebuilder:default=false
	Temporary bool `json:"temporary"`
}

// KeycloakUserStatus defines the observed state of KeycloakUser.
type KeycloakUserStatus struct {
	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// Conditions represent the latest available observations of an object's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncedPasswordSecretVersion stores the ResourceVersion of the password secret
	// that was last successfully synced to Keycloak. Used to detect secret changes.
	// +optional
	LastSyncedPasswordSecretVersion string `json:"lastSyncedPasswordSecretVersion,omitempty"`

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

// KeycloakUser is the Schema for the keycloak user API.
type KeycloakUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakUserSpec   `json:"spec,omitempty"`
	Status KeycloakUserStatus `json:"status,omitempty"`
}

func (in *KeycloakUser) GetReconciliationStrategy() string {
	if in.Spec.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.Spec.ReconciliationStrategy
}

func (in *KeycloakUser) IsReconciliationStrategyAddOnly() bool {
	return in.GetReconciliationStrategy() == ReconciliationStrategyAddOnly
}

func (in *KeycloakUser) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakUser) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakUser) GetPhase() string {
	return in.Status.Phase
}

func (in *KeycloakUser) SetPhase(value string) {
	in.Status.Phase = value
}

func (in *KeycloakUser) GetError() string {
	return in.Status.Error
}

func (in *KeycloakUser) SetError(err error) {
	if err != nil {
		in.Status.Error = err.Error()
	}

	in.Status.Error = ""
}

func (in *KeycloakUser) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

func (in *KeycloakUser) GetConfigRef() common.ConfigRef {
	return in.Spec.ConfigRef
}

// +kubebuilder:object:root=true

// KeycloakUserList contains a list of KeycloakUser.
type KeycloakUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakUser{}, &KeycloakUserList{})
}
