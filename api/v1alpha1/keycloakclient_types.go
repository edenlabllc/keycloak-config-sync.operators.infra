package v1alpha1

import (
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Client struct {
	// ClientId is a unique keycloak client ID referenced in URI and tokens.
	ClientId string `json:"clientId"`

	// Secret is kubernetes secret name where the client's secret will be stored.
	// Secret should have the following format: $secretName:secretKey.
	// If not specified, a client secret will be generated and stored in a secret with the name keycloak-client-{metadata.name}-secret.
	// If keycloak client is public, secret property will be ignored.
	// +optional
	// +kubebuilder:example="$keycloak-secret:client_secret"
	Secret string `json:"secret,omitempty"`

	// RealmRoles is a list of realm roles assigned to client.
	// +nullable
	// +optional
	RealmRoles *[]RealmRole `json:"realmRoles,omitempty"`

	// Public is a flag to set client as public.
	// +optional
	Public bool `json:"public,omitempty"`

	// WebUrl is a client web url.
	// +optional
	WebUrl string `json:"webUrl,omitempty"`

	// AdminUrl is client admin url.
	// If empty - WebUrl will be used.
	// +optional
	AdminUrl string `json:"adminUrl,omitempty"`

	// HomeUrl is a client home url.
	// +optional
	HomeUrl string `json:"homeUrl,omitempty"`

	// Protocol is a client protocol.
	// +nullable
	// +optional
	Protocol *string `json:"protocol,omitempty"`

	// Attributes is a map of client attributes.
	// +nullable
	// +optional
	// +kubebuilder:default={"post.logout.redirect.uris": "+"}
	Attributes map[string]string `json:"attributes,omitempty"`

	// DirectAccess is a flag to set client as direct access.
	// +optional
	DirectAccess bool `json:"directAccess,omitempty"`

	// AdvancedProtocolMappers is a flag to enable advanced protocol mappers.
	// +optional
	AdvancedProtocolMappers bool `json:"advancedProtocolMappers,omitempty"`

	// ClientRoles is a list of client roles names assigned to client.
	// Deprecated: Use ClientRolesV2 instead.
	// +nullable
	// +optional
	ClientRoles []string `json:"clientRoles,omitempty"`

	// ClientRolesV2 is a list of client roles assigned to client.
	// +nullable
	// +optional
	ClientRolesV2 []ClientRole `json:"clientRolesV2,omitempty"`

	// ProtocolMappers is a list of protocol mappers assigned to client.
	// +nullable
	// +optional
	ProtocolMappers *[]ProtocolMapper `json:"protocolMappers,omitempty"`

	// ServiceAccount is a service account configuration.
	// +nullable
	// +optional
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`

	// FrontChannelLogout is a flag to enable front channel logout.
	// +optional
	FrontChannelLogout bool `json:"frontChannelLogout,omitempty"`

	// ReconciliationStrategy is a strategy to reconcile client.
	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`

	// DefaultClientScopes is a list of default client scopes assigned to client.
	// +nullable
	// +optional
	DefaultClientScopes []string `json:"defaultClientScopes,omitempty"`

	// OptionalClientScopes is a list of optional client scopes assigned to client.
	// +nullable
	// +optional
	OptionalClientScopes []string `json:"optionalClientScopes,omitempty"`

	// RedirectUris is a list of valid URI pattern a browser can redirect to after a successful login.
	// Simple wildcards are allowed such as 'https://example.com/*'.
	// Relative path can be specified too, such as /my/relative/path/*. Relative paths are relative to the client root URL.
	// If not specified, spec.webUrl + "/*" will be used.
	// +nullable
	// +optional
	// +kubebuilder:example={"https://example.com/*", "/my/relative/path/*"}
	RedirectUris []string `json:"redirectUris,omitempty"`

	// WebOrigins is a list of allowed CORS origins.
	// To permit all origins of Valid Redirect URIs, add '+'. This does not include the '*' wildcard though.
	// To permit all origins, explicitly add '*'.
	// If not specified, the value from `WebUrl` is used
	// +nullable
	// +optional
	// +kubebuilder:example={"https://example.com/*"}
	WebOrigins []string `json:"webOrigins,omitempty"`

	// ImplicitFlowEnabled is a flag to enable support for OpenID Connect redirect based authentication without authorization code.
	// +optional
	ImplicitFlowEnabled bool `json:"implicitFlowEnabled,omitempty"`

	// AuthorizationServicesEnabled enable/disable fine-grained authorization support for a client.
	// +optional
	AuthorizationServicesEnabled bool `json:"authorizationServicesEnabled,omitempty"`

	// AdminFineGrainedPermissionsEnabled enable/disable fine-grained admin permissions for a client.
	// Feature flag admin-fine-grained-authz:v1 should be enabled in Keycloak server.
	// Important: FGAP:V1 Keycloak feature remains in preview and may be deprecated and removed in a future releases.
	// +optional
	AdminFineGrainedPermissionsEnabled bool `json:"adminFineGrainedPermissionsEnabled,omitempty"`

	// BearerOnly is a flag to enable bearer-only.
	// +optional
	BearerOnly bool `json:"bearerOnly,omitempty"`

	// ClientAuthenticatorType is a client authenticator type.
	// +optional
	// +kubebuilder:default="client-secret"
	ClientAuthenticatorType string `json:"clientAuthenticatorType,omitempty"`

	// ConsentRequired is a flag to enable consent.
	// +optional
	ConsentRequired bool `json:"consentRequired,omitempty"`

	// Description is a client description.
	// +optional
	Description string `json:"description,omitempty"`

	// Enabled is a flag to enable client.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// FullScopeAllowed is a flag to enable full scope.
	// +optional
	// +kubebuilder:default=true
	FullScopeAllowed bool `json:"fullScopeAllowed"`

	// Name is a client name.
	// +optional
	Name string `json:"name,omitempty"`

	// StandardFlowEnabled is a flag to enable standard flow.
	// +optional
	// +kubebuilder:default=true
	StandardFlowEnabled bool `json:"standardFlowEnabled"`

	// SurrogateAuthRequired is a flag to enable surrogate auth.
	SurrogateAuthRequired bool `json:"surrogateAuthRequired,omitempty"`

	// Authorization is a client authorization configuration.
	// +nullable
	// +optional
	Authorization *Authorization `json:"authorization,omitempty"`

	// Permission is a client permissions configuration
	// +nullable
	// +optional
	Permission *AdminFineGrainedPermission `json:"permission,omitempty"`

	// AuthenticationFlowBindingOverrides client auth flow overrides
	// +optional
	AuthenticationFlowBindingOverrides *AuthenticationFlowBindingOverrides `json:"authenticationFlowBindingOverrides,omitempty"`
}

func (in *Client) GetReconciliationStrategy() string {
	if in.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.ReconciliationStrategy
}

type ClientScope struct {
	// Name of keycloak client scope.
	Name string `json:"name"`

	// Protocol is SSO protocol configuration which is being supplied by this client scope.
	Protocol string `json:"protocol"`

	// Description is a description of client scope.
	// +optional
	Description string `json:"description,omitempty"`

	// Attributes is a map of client scope attributes.
	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// Default is a flag to set client scope as default.
	// Deprecated: Use Type: default instead.
	// +optional
	Default bool `json:"default,omitempty"`

	// Type of the client scope.
	// If set to "default", the client scope is assigned to all clients by default.
	// If set to "optional", the client scope can be assigned to clients on demand.
	// If set to "none", the client scope is not assigned to any clients by default.
	// +kubebuilder:validation:Enum=default;optional;none
	// +kubebuilder:default=none
	// +optional
	Type string `json:"type"`

	// ProtocolMappers is a list of protocol mappers assigned to client scope.
	// +nullable
	// +optional
	ProtocolMappers []ProtocolMapper `json:"protocolMappers,omitempty"`

	// ReconciliationStrategy is a strategy to reconcile client.
	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`
}

func (in *ClientScope) GetReconciliationStrategy() string {
	if in.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.ReconciliationStrategy
}

// GetType returns the type of the client scope.
// For backward compatibility, if Default is set to true, it returns "default".
func (in *ClientScope) GetType() string {
	if in.Default {
		return KeycloakClientScopeTypeDefault
	}

	return in.Type
}

func (in *ClientScope) IsTypeDefault() bool {
	return in.GetType() == KeycloakClientScopeTypeDefault
}

func (in *ClientScope) IsTypeOptional() bool {
	return in.GetType() == KeycloakClientScopeTypeOptional
}

func (in *ClientScope) IsTypeNone() bool {
	return in.GetType() == KeycloakClientScopeTypeNone
}

type ServiceAccount struct {
	// Enabled is a flag to enable service account.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// RealmRoles is a list of realm roles assigned to service account.
	// +nullable
	// +optional
	RealmRoles []string `json:"realmRoles"`

	// ClientRoles is a list of client roles assigned to service account.
	// +nullable
	// +optional
	ClientRoles []UserClientRole `json:"clientRoles,omitempty"`

	// Attributes is a map of service account attributes.
	// Deprecated: Use AttributesV2 instead.
	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// AttributesV2 is a map of service account attributes.
	// Each attribute can have multiple values.
	// +nullable
	// +optional
	AttributesV2 map[string][]string `json:"attributesV2,omitempty"`

	// Groups is a list of groups assigned to service account
	// +nullable
	// +optional
	Groups []string `json:"groups,omitempty"`
}

type UserClientRole struct {
	// ClientID is a client ID.
	ClientID string `json:"clientId"`

	// Roles is a list of client roles names assigned to user.
	// +nullable
	// +optional
	Roles []string `json:"roles,omitempty"`
}

type ClientRole struct {
	// Name is a client role name.
	// +required
	Name string `json:"name,omitempty"`

	// Description is a client role description.
	// +optional
	Description string `json:"description,omitempty"`

	// AssociatedClientRoles is a list of client roles names associated with the current role.
	// These roles won't be created automatically, user should specify them separately in clientRolesV2.
	// +nullable
	// +optional
	AssociatedClientRoles []string `json:"associatedClientRoles,omitempty"`
}

type ProtocolMapper struct {
	// Name is a protocol mapper name.
	// +optional
	Name string `json:"name,omitempty"`

	// Protocol is a protocol name.
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// ProtocolMapper is a protocol mapper name.
	// +optional
	ProtocolMapper string `json:"protocolMapper,omitempty"`

	// Config is a map of protocol mapper configuration.
	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type RealmRole struct {
	// Name is a realm role name.
	// +optional
	Name string `json:"name,omitempty"`

	// Composite is a realm composite role name.
	Composite string `json:"composite"`
}

type Authorization struct {
	Scopes []string `json:"scopes,omitempty"`

	Policies []Policy `json:"policies,omitempty"`

	Permissions []Permission `json:"permissions,omitempty"`

	Resources []Resource `json:"resources,omitempty"`
}

type AuthenticationFlowBindingOverrides struct {
	Browser     string `json:"browser,omitempty"`
	DirectGrant string `json:"directGrant,omitempty"`
}

type AdminFineGrainedPermission struct {
	// ScopePermissions mapping of scope and the policies attached
	// +optional
	ScopePermissions []ScopePermissions `json:"scopePermissions,omitempty"`
}

type ScopePermissions struct {
	Name     string   `json:"name"`
	Policies []string `json:"policies,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeycloakClientSpec defines the desired state of KeycloakClient
type KeycloakClientSpec struct {
	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// ConfigRef is reference to Keycloak Config.
	// +required
	ConfigRef common.ConfigRef `json:"configRef"`

	// Client is a list of client.
	// +required
	Client []Client `json:"client"`

	// ClientScope is a list of client scopes.
	// +nullable
	// +optional
	ClientScope *[]ClientScope `json:"clientScope,omitempty"`
}

// KeycloakClientStatus defines the observed state of KeycloakClient.
type KeycloakClientStatus struct {
	// +optional
	Error string `json:"error,omitempty"`

	// +optional
	Phase string `json:"phase,omitempty"`

	// +optional
	ClientIDs map[string]string `json:"clientIDs,omitempty"`

	// +optional
	ClientScopeIDs map[string]string `json:"clientScopeIDs,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// Conditions represent the latest available observations of an object's state.
	// +optional
	// +nullable
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (kcss *KeycloakClientStatus) GetClientIDByName(name string) string {
	if val, ok := kcss.ClientIDs[name]; ok {
		return val
	}

	return ""
}

func (kcss *KeycloakClientStatus) PutClientIDByName(name, id string) {
	if _, ok := kcss.ClientIDs[name]; ok {
		kcss.ClientIDs[name] = id
	} else {
		if kcss.ClientIDs == nil {
			kcss.ClientIDs = make(map[string]string)
		}

		kcss.ClientIDs[name] = id
	}
}

func (kcss *KeycloakClientStatus) GetClientScopeIDByName(name string) string {
	if val, ok := kcss.ClientScopeIDs[name]; ok {
		return val
	}

	return ""
}

func (kcss *KeycloakClientStatus) PutClientScopeIDByName(name, id string) {
	if _, ok := kcss.ClientScopeIDs[name]; ok {
		kcss.ClientScopeIDs[name] = id
	} else {
		if kcss.ClientScopeIDs == nil {
			kcss.ClientScopeIDs = make(map[string]string)
		}

		kcss.ClientScopeIDs[name] = id
	}
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Realm",type="string",JSONPath=".spec.realmRef.name",description="Keycloak ref to realm name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Reconciliation phase"
// +kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.error",description="Resource error"

// KeycloakClient is the Schema for the keycloakclient API
type KeycloakClient struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of KeycloakClient
	// +required
	Spec KeycloakClientSpec `json:"spec"`

	// status defines the observed state of KeycloakClient
	// +optional
	Status KeycloakClientStatus `json:"status,omitzero"`
}

func (in *KeycloakClient) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakClient) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakClient) GetPhase() string {
	return in.Status.Phase
}

func (in *KeycloakClient) SetPhase(value string) {
	in.Status.Phase = value
}

func (in *KeycloakClient) GetError() string {
	return in.Status.Error
}

func (in *KeycloakClient) SetError(err error) {
	if err != nil {
		in.Status.Error = err.Error()
	}

	in.Status.Error = ""
}

func (in *KeycloakClient) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

func (in *KeycloakClient) GetConfigRef() common.ConfigRef {
	return in.Spec.ConfigRef
}

// +kubebuilder:object:root=true

// KeycloakClientList contains a list of KeycloakClient
type KeycloakClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []KeycloakClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakClient{}, &KeycloakClientList{})
}
