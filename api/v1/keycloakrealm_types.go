package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
)

// KeycloakRealmSpec defines the desired state of KeycloakRealm.
type KeycloakRealmSpec struct {
	// RealmName specifies the name of the realm.
	// +required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	RealmName string `json:"realmName"`

	// ReconciliationStrategy is a strategy to reconcile client.
	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`

	// KeycloakRef is reference to Keycloak custom resource.
	// +required
	KeycloakRef common.KeycloakRef `json:"keycloakRef,omitempty"`

	// Users is a list of users to create in the realm.
	// +nullable
	// +optional
	Users []User `json:"users,omitempty"`

	// BrowserFlow specifies the authentication flow to use for the realm's browser clients.
	// +nullable
	// +optional
	BrowserFlow *string `json:"browserFlow,omitempty"`

	// Themes is a map of themes to apply to the realm.
	// +nullable
	// +optional
	Themes *RealmThemes `json:"themes,omitempty"`

	// Localization configures supported/default locales and custom message bundles (realm export field `localizationTexts`).
	// +nullable
	// +optional
	Localization *RealmLocalization `json:"localization,omitempty"`

	// BrowserSecurityHeaders is a map of security headers to apply to HTTP responses from the realm's browser clients.
	// +nullable
	// +optional
	BrowserSecurityHeaders *map[string]string `json:"browserSecurityHeaders,omitempty"`

	// ID is the ID of the realm.
	// +nullable
	// +optional
	ID *string `json:"id,omitempty"`

	// RealmEventConfig is the configuration for events in the realm.
	// +nullable
	// +optional
	RealmEventConfig *common.RealmEventConfig `json:"realmEventConfig,omitempty"`

	// PasswordPolicies is a list of password policies to apply to the realm.
	// +nullable
	// +optional
	PasswordPolicies []common.PasswordPolicy `json:"passwordPolicy,omitempty"`

	// DisplayHTMLName name to render in the UI
	// +optional
	DisplayHTMLName string `json:"displayHtmlName,omitempty"`

	// FrontendURL Set the frontend URL for the realm. Use in combination with the default hostname provider to override the base URL for frontend requests for a specific realm.
	// +optional
	FrontendURL string `json:"frontendUrl,omitempty"`

	// TokenSettings is the configuration for tokens in the realm.
	// +nullable
	// +optional
	TokenSettings *common.TokenSettings `json:"tokenSettings,omitempty"`

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
	// Attributes and groups will be added to the current realm configuration.
	// Deletion of attributes and groups is not supported.
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
	Login *RealmLogin `json:"login,omitempty"`

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

// RealmLocalization configures realm locales and custom translations (Keycloak Admin API / realm export).
type RealmLocalization struct {
	// InternationalizationEnabled enables the realm internationalization feature.
	// +nullable
	// +optional
	InternationalizationEnabled *bool `json:"internationalizationEnabled,omitempty"`

	// SupportedLocales lists locale tags offered to users (BCP 47).
	// +optional
	SupportedLocales []string `json:"supportedLocales,omitempty"`

	// DefaultLocale is the realm default locale tag.
	// +optional
	DefaultLocale *string `json:"defaultLocale,omitempty"`

	// LocalizationTexts maps locale code to message key → translated text (same shape as Keycloak `localizationTexts` in a realm export).
	// +optional
	LocalizationTexts map[string]map[string]string `json:"localizationTexts,omitempty"`
}

type User struct {
	// Username of keycloak user.
	Username string `json:"username"`

	// RealmRoles is a list of roles attached to keycloak user.
	RealmRoles []string `json:"realmRoles,omitempty"`
}

type RealmThemes struct {
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

	// InternationalizationEnabled indicates whether to enable internationalization.
	// +nullable
	// +optional
	InternationalizationEnabled *bool `json:"internationalizationEnabled"`
}

func (in *KeycloakRealm) GetKeycloakRef() common.KeycloakRef {
	return in.Spec.KeycloakRef
}

type SSORealmMapper struct {
	// IdentityProviderMapper specifies the identity provider mapper to use.
	// +optional
	IdentityProviderMapper string `json:"identityProviderMapper,omitempty"`

	// Name specifies the name of the SSO realm mapper.
	// +optional
	Name string `json:"name,omitempty"`

	// Config is a map of configuration options for the SSO realm mapper.
	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// RealmLogin defines the login settings for the realm.
type RealmLogin struct {
	// UserRegistration enables/disables the registration page. A link for registration will show on the login page too.
	// +optional
	// +kubebuilder:default=false
	UserRegistration bool `json:"userRegistration"`

	// ForgotPassword shows a link on the login page for users who have forgotten their credentials.
	// +optional
	// +kubebuilder:default=false
	ForgotPassword bool `json:"forgotPassword"`

	// RememberMe shows checkbox on the login page to allow the user to remain logged in between browser restarts until the session expires.
	// +optional
	// +kubebuilder:default=false
	RememberMe bool `json:"rememberMe"`

	// EmailAsUsername allows users to set email as username.
	// +optional
	// +kubebuilder:default=false
	EmailAsUsername bool `json:"emailAsUsername"`

	// LoginWithEmail allows users to log in with their email address.
	// +optional
	// +kubebuilder:default=true
	LoginWithEmail bool `json:"loginWithEmail"`

	// DuplicateEmails allows multiple users to have the same email address.
	// +optional
	// +kubebuilder:default=false
	DuplicateEmails bool `json:"duplicateEmails"`

	// VerifyEmail requires user to verify their email address after initial login or after address changes are submitted.
	// +optional
	// +kubebuilder:default=false
	VerifyEmail bool `json:"verifyEmail"`

	// EditUsername allows to edit username.
	// +optional
	// +kubebuilder:default=false
	EditUsername bool `json:"editUsername"`
}

// KeycloakRealmStatus defines the observed state of KeycloakRealm.
type KeycloakRealmStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// +optional
	Value string `json:"value,omitempty"`
}

func (in *KeycloakRealm) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealm) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Available",type="boolean",JSONPath=".status.available",description="Is the resource available"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconciliation status"
// +kubebuilder:printcolumn:name="Realm",type="boolean",JSONPath=".spec.realmName",description="Keycloak realm name"
// +kubebuilder:printcolumn:name="Keycloak",type="boolean",JSONPath=".spec.keycloakRef",description="Keycloak instance name"

// KeycloakRealm is the Schema for the keycloak realms API.
type KeycloakRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmSpec   `json:"spec,omitempty"`
	Status KeycloakRealmStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeycloakRealmList contains a list of KeycloakRealm.
type KeycloakRealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealm{}, &KeycloakRealmList{})
}
