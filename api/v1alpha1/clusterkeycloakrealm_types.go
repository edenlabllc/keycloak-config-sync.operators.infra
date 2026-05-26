package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
	v1 "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1"
)

// ClusterKeycloakRealmSpec defines the desired state of ClusterKeycloakRealm.
type ClusterKeycloakRealmSpec struct {
	// ClusterKeycloakRef is a name of the ClusterKeycloak instance that owns the realm.
	// +required
	ClusterKeycloakRef string `json:"clusterKeycloakRef"`

	// RealmName specifies the name of the realm.
	RealmName string `json:"realmName"`

	// ReconciliationStrategy is a strategy to reconcile client.
	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`

	// FrontendURL Set the frontend URL for the realm.
	// Use in combination with the default hostname provider to override the base URL for frontend requests for a specific realm.
	// +optional
	FrontendURL string `json:"frontendUrl,omitempty"`

	// RealmEventConfig is the configuration for events in the realm.
	// +nullable
	// +optional
	RealmEventConfig *common.RealmEventConfig `json:"realmEventConfig,omitempty"`

	// Themes is a map of themes to apply to the realm.
	// +nullable
	// +optional
	Themes *ClusterRealmThemes `json:"themes,omitempty"`

	// Localization is the configuration for localization in the realm.
	// +nullable
	// +optional
	Localization *RealmLocalization `json:"localization,omitempty"`

	// BrowserSecurityHeaders is a map of security headers to apply to HTTP responses from the realm's browser clients.
	// +nullable
	// +optional
	BrowserSecurityHeaders *map[string]string `json:"browserSecurityHeaders,omitempty"`

	// PasswordPolicies is a list of password policies to apply to the realm.
	// +nullable
	// +optional
	PasswordPolicies []common.PasswordPolicy `json:"passwordPolicy,omitempty"`

	// TokenSettings is the configuration for tokens in the realm.
	// +nullable
	// +optional
	TokenSettings *common.TokenSettings `json:"tokenSettings,omitempty"`

	// AuthenticationFlow is the configuration for authentication flows in the realm.
	// +nullable
	// +optional
	AuthenticationFlow *AuthenticationFlow `json:"authenticationFlows,omitempty"`

	// DisplayHTMLName name to render in the UI.
	// +optional
	DisplayHTMLName string `json:"displayHtmlName,omitempty"`

	// DisplayName is the display name of the realm.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// SSLRequired is the ssl required of the realm.
	// +optional
	// +kubebuilder:default=external
	SSLRequired string `json:"sslRequired,omitempty"`

	// BruteForceProtected is the brute force protected of the realm.
	// +optional
	// +kubebuilder:default=false
	BruteForceProtected bool `json:"bruteForceProtected,omitempty"`

	// +optional
	// +kubebuilder:default=false
	PermanentLockout bool `json:"permanentLockout,omitempty"`

	// +optional
	// +kubebuilder:default=900
	MaxFailureWaitSeconds int `json:"maxFailureWaitSeconds,omitempty"`

	// +optional
	// +kubebuilder:default=60
	MinimumQuickLoginWaitSeconds int `json:"minimumQuickLoginWaitSeconds,omitempty"`

	// +optional
	// +kubebuilder:default=60
	WaitIncrementSeconds int `json:"waitIncrementSeconds,omitempty"`

	// +optional
	// +kubebuilder:default=1000
	QuickLoginCheckMilliSeconds int64 `json:"quickLoginCheckMilliSeconds,omitempty"`

	// +optional
	// +kubebuilder:default=43200
	MaxDeltaTimeSeconds int `json:"maxDeltaTimeSeconds,omitempty"`

	// +optional
	// +kubebuilder:default=30
	FailureFactor int `json:"failureFactor,omitempty"`

	// +optional
	// +nullable
	SupportedLocales []string `json:"supportedLocales,omitempty"`

	// +optional
	// +kubebuilder:default=registration
	RegistrationFlow string `json:"registrationFlow,omitempty"`

	// +optional
	// +kubebuilder:default=direct grant
	DirectGrantFlow string `json:"directGrantFlow,omitempty"`

	// +optional
	// +kubebuilder:default=reset credentials
	ResetCredentialsFlow string `json:"resetCredentialsFlow,omitempty"`

	// +optional
	// +kubebuilder:default=clients
	ClientAuthenticationFlow string `json:"clientAuthenticationFlow,omitempty"`

	// +optional
	// +kubebuilder:default=docker auth
	DockerAuthenticationFlow string `json:"dockerAuthenticationFlow,omitempty"`

	// +optional
	// +kubebuilder:default=false
	UserManagedAccessAllowed bool `json:"userManagedAccessAllowed,omitempty"`

	// +optional
	// +nullable
	Attributes *map[string]string `json:"attributes,omitempty"`

	// +optional
	// +kubebuilder:default=false
	UseOrganizations bool `json:"useOrganizations,omitempty"`

	// OrganizationsEnabled enables Keycloak Organizations feature for this realm.
	// When enabled, this realm can support Organization resources for multi-tenant scenarios,
	// identity provider groupings, and domain-based user routing.
	// +optional
	// +kubebuilder:default=false
	OrganizationsEnabled bool `json:"organizationsEnabled,omitempty"`

	// UserProfileConfig is the configuration for user profiles in the realm.
	// +nullable
	// +optional
	UserProfileConfig *common.UserProfileConfig `json:"userProfileConfig,omitempty"`

	// Smtp is the configuration for email in the realm.
	// +nullable
	// +optional
	Smtp *common.SMTP `json:"smtp,omitempty"`

	// Login settings for the realm.
	// +nullable
	// +optional
	Login *v1.RealmLogin `json:"login,omitempty"`

	// Sessions defines the session settings for the realm.
	// +optional
	Sessions *common.RealmSessions `json:"sessions,omitempty"`

	// RequiredCredentials for the realm.
	// +nullable
	// +optional
	RequiredCredentials []string `json:"requiredCredentials,omitempty"`

	// WebAuthnPolicySettings is the webAuthPolicy for realm.
	// +nullable
	// +optional
	WebAuthnPolicySettings *common.WebAuthnPolicySettings `json:"webAuthnPolicySettings,omitempty"`

	// Oauth2DeviceSettings is the Oauth2Device for realm.
	// +nullable
	// +optional
	Oauth2DeviceSettings *common.Oauth2DeviceSettings `json:"oauth2DeviceSettings,omitempty"`

	// OTPPolicySettings is the otp policy for realm.
	// +nullable
	// +optional
	OTPPolicySettings *common.OTPPolicySettings `json:"otpPolicySettings,omitempty"`

	// ClientSessionSettings is the client session for realm.
	// +nullable
	// +optional
	ClientSessionSettings *common.ClientSessionSettings `json:"clientSessionSettings,omitempty"`

	// DefaultRole is the default role for realm.
	// +nullable
	// +optional
	DefaultRole *common.RealmRoleRepresentation `json:"defaultRole,omitempty"`
}

type AuthenticationFlow struct {
	// BrowserFlow specifies the authentication flow to use for the realm's browser clients.
	// +optional
	// +kubebuilder:example="browser"
	BrowserFlow string `json:"browserFlow,omitempty"`
}

type ClusterRealmThemes struct {
	// LoginTheme specifies the login theme to use for the realm.
	// +nullable
	// +optional
	LoginTheme *string `json:"loginTheme"`

	// AccountTheme specifies the account theme to use for the realm.
	// +nullable
	// +optional
	AccountTheme *string `json:"accountTheme"`

	// AdminConsoleTheme specifies the admin console theme to use for the realm.
	// +nullable
	// +optional
	AdminConsoleTheme *string `json:"adminConsoleTheme"`

	// EmailTheme specifies the email theme to use for the realm.
	// +nullable
	// +optional
	EmailTheme *string `json:"emailTheme"`
}

type RealmLocalization struct {
	// InternationalizationEnabled indicates whether to enable internationalization.
	// +nullable
	// +optional
	InternationalizationEnabled *bool `json:"internationalizationEnabled,omitempty"`

	// SupportedLocales lists locale tags offered to users (BCP 47).
	// +optional
	SupportedLocales []string `json:"supportedLocales,omitempty"`

	// DefaultLocale is the realm default locale tag.
	// +optional
	DefaultLocale *string `json:"defaultLocale,omitempty"`

	// LocalizationTexts maps locale code to message key → translated text (Keycloak realm export field `localizationTexts`).
	// +optional
	LocalizationTexts map[string]map[string]string `json:"localizationTexts,omitempty"`
}

// ClusterKeycloakRealmStatus defines the observed state of ClusterKeycloakRealm.
type ClusterKeycloakRealmStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// +optional
	Value string `json:"value,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Available",type="boolean",JSONPath=".status.available",description="Keycloak realm is available"
// +kubebuilder:printcolumn:name="Realm",type="boolean",JSONPath=".spec.realmName",description="Keycloak realm name"
// +kubebuilder:printcolumn:name="Cluster-Keycloak",type="boolean",JSONPath=".spec.clusterKeycloakRef",description="ClusterKeycloak instance name"

// ClusterKeycloakRealm is the Schema for the clusterkeycloakrealms API.
type ClusterKeycloakRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterKeycloakRealmSpec   `json:"spec,omitempty"`
	Status ClusterKeycloakRealmStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterKeycloakRealmList contains a list of ClusterKeycloakRealm.
type ClusterKeycloakRealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterKeycloakRealm `json:"items"`
}

func (r *ClusterKeycloakRealm) GetKeycloakRef() common.KeycloakRef {
	return common.KeycloakRef{
		Kind: ClusterKeycloakKind,
		Name: r.Spec.ClusterKeycloakRef,
	}
}

func (in *ClusterKeycloakRealm) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *ClusterKeycloakRealm) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func init() {
	SchemeBuilder.Register(&ClusterKeycloakRealm{}, &ClusterKeycloakRealmList{})
}
