// +kubebuilder:object:generate=true
package common

// PasswordPolicy defines a single password policy rule for a realm.
type PasswordPolicy struct {
	// Type of password policy.
	Type string `json:"type"`

	// Value of password policy.
	Value string `json:"value"`
}

// TokenSettings is the configuration for tokens in the realm.
// +kubebuilder:object:generate=true
type TokenSettings struct {
	// DefaultSignatureAlgorithm specifies the default algorithm used to sign tokens for the realm
	// +optional
	// +kubebuilder:validation:Enum=ES256;ES384;ES512;EdDSA;HS256;HS384;HS512;PS256;PS384;PS512;RS256;RS384;RS512
	// +kubebuilder:default=RS256
	// +kubebuilder:example=RS256
	DefaultSignatureAlgorithm string `json:"defaultSignatureAlgorithm,omitempty"`

	// RevokeRefreshToken if enabled a refresh token can only be used up to 'refreshTokenMaxReuse' and
	// is revoked when a different token is used.
	// Otherwise, refresh tokens are not revoked when used and can be used multiple times.
	// +optional
	// +kubebuilder:default=false
	RevokeRefreshToken bool `json:"revokeRefreshToken"`

	// RefreshTokenMaxReuse specifies maximum number of times a refresh token can be reused.
	// When a different token is used, revocation is immediate.
	// +optional
	// +kubebuilder:default=0
	RefreshTokenMaxReuse int `json:"refreshTokenMaxReuse,omitempty"`

	// AccessTokenLifespan specifies max time(in seconds) before an access token is expired.
	// This value is recommended to be short relative to the SSO timeout.
	// +optional
	// +kubebuilder:default=300
	AccessTokenLifespan int `json:"accessTokenLifespan,omitempty"`

	// AccessTokenLifespanForImplicitFlow specifies max time(in seconds) before an access token is expired for implicit flow.
	// +optional
	// +kubebuilder:default=900
	AccessTokenLifespanForImplicitFlow int `json:"accessToken,omitempty"`

	// AccessCodeLifespan specifies max time(in seconds)a client has to finish the access token protocol.
	// This should normally be 1 minute.
	// +optional
	// +kubebuilder:default=60
	AccessCodeLifespan int `json:"accessCodeLifespan,omitempty"`

	// AccessCodeLifespanUserAction specifies max time(in seconds) before an action permit sent by a user (such as a forgot password e-mail) is expired.
	// This value is recommended to be short because it's expected that the user would react to self-created action quickly.
	// +optional
	// +kubebuilder:default=300
	ActionTokenGeneratedByUserLifespan int `json:"actionTokenGeneratedByUserLifespan,omitempty"`

	// ActionTokenGeneratedByAdminLifespan specifies max time(in seconds) before an action permit sent to a user by administrator is expired.
	// This value is recommended to be long to allow administrators to send e-mails for users that are currently offline.
	// The default timeout can be overridden immediately before issuing the token.
	// +optional
	// +kubebuilder:default=43200
	ActionTokenGeneratedByAdminLifespan int `json:"actionTokenGeneratedByAdminLifespan,omitempty"`
}

// UserProfileConfig defines the configuration for user profile in the realm.
type UserProfileConfig struct {
	// UnmanagedAttributePolicy are user attributes not explicitly defined in the user profile configuration.
	// Empty value means that unmanaged attributes are disabled.
	// Possible values:
	// ENABLED - unmanaged attributes are allowed.
	// ADMIN_VIEW - unmanaged attributes are read-only and only available through the administration console and API.
	// ADMIN_EDIT - unmanaged attributes can be managed only through the administration console and API.
	// +optional
	UnmanagedAttributePolicy string `json:"unmanagedAttributePolicy,omitempty"`

	// Attributes specifies the list of user profile attributes.
	Attributes []UserProfileAttribute `json:"attributes,omitempty"`

	// Groups specifies the list of user profile groups.
	Groups []UserProfileGroup `json:"groups,omitempty"`
}

type UserProfileAttribute struct {
	// Name of the user attribute, used to uniquely identify an attribute.
	// +required
	Name string `json:"name"`

	// Display name for the attribute.
	DisplayName string `json:"displayName,omitempty"`

	// Group to which the attribute belongs.
	Group string `json:"group,omitempty"`

	// Multivalued specifies if this attribute supports multiple values.
	// This setting is an indicator and does not enable any validation
	Multivalued bool `json:"multivalued,omitempty"`

	// Permissions specifies the permissions for the attribute.
	Permissions *UserProfileAttributePermissions `json:"permissions,omitempty"`

	// Required indicates that the attribute must be set by users and administrators.
	Required *UserProfileAttributeRequired `json:"required,omitempty"`

	// Selector specifies the scopes for which the attribute is available.
	Selector *UserProfileAttributeSelector `json:"selector,omitempty"`

	// Annotations specifies the annotations for the attribute.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Validations specifies the validations for the attribute.
	Validations map[string]map[string]UserProfileAttributeValidation `json:"validations,omitempty"`
}

type UserProfileAttributeValidation struct {
	// +optional
	StringVal string `json:"stringVal,omitempty"`

	// +optional
	// +nullable
	MapVal map[string]string `json:"mapVal,omitempty"`

	// +optional
	IntVal int `json:"intVal,omitempty"`

	// +optional
	// +nullable
	SliceVal []string `json:"sliceVal,omitempty"`
}

type UserProfileAttributePermissions struct {
	// Edit specifies who can edit the attribute.
	Edit []string `json:"edit,omitempty"`

	// View specifies who can view the attribute.
	View []string `json:"view,omitempty"`
}

// UserProfileAttributeRequired defines model for UserProfileAttributeRequired.
type UserProfileAttributeRequired struct {
	// Roles specifies the roles for whom the attribute is required.
	Roles []string `json:"roles,omitempty"`

	// Scopes specifies the scopes when the attribute is required.
	Scopes []string `json:"scopes,omitempty"`
}

// UserProfileAttributeSelector defines model for UserProfileAttributeSelector.
type UserProfileAttributeSelector struct {
	// Scopes specifies the scopes for which the attribute is available.
	Scopes []string `json:"scopes,omitempty"`
}

type UserProfileGroup struct {
	// Name is unique name of the group.
	// +required
	Name string `json:"name"`

	// Annotations specifies the annotations for the group.
	// +optional
	// nullable
	Annotations map[string]string `json:"annotations,omitempty"`

	// DisplayDescription specifies a user-friendly name for the group that should be used when rendering a group of attributes in user-facing forms.
	DisplayDescription string `json:"displayDescription,omitempty"`

	// DisplayHeader specifies a text that should be used as a header when rendering user-facing forms.
	DisplayHeader string `json:"displayHeader,omitempty"`
}

type SMTP struct {
	// Template specifies the email template configuration.
	// +required
	Template EmailTemplate `json:"template"`

	// Connection specifies the email connection configuration.
	// +required
	Connection EmailConnection `json:"connection"`
}

type EmailTemplate struct {
	// From specifies the sender email address.
	// +required
	From string `json:"from"`

	// FromDisplayName specifies the sender display for sender email address.
	// +optional
	FromDisplayName string `json:"fromDisplayName,omitempty"`

	// ReplyTo specifies the reply-to email address.
	// +optional
	ReplyTo string `json:"replyTo,omitempty"`

	// ReplyToDisplayName specifies display name for reply-to email address.
	// +optional
	ReplyToDisplayName string `json:"replyToDisplayName,omitempty"`

	// EnvelopeFrom is an email address used for bounces .
	// +optional
	EnvelopeFrom string `json:"envelopeFrom,omitempty"`
}

type EmailConnection struct {
	// Host specifies the email server host.
	// +required
	Host string `json:"host"`

	// Port specifies the email server port.
	// +optional
	// +kubebuilder:default=25
	Port int `json:"port"`

	// EnableSSL specifies if SSL is enabled.
	EnableSSL bool `json:"enableSSL,omitempty"`

	// EnableStartTLS specifies if StartTLS is enabled.
	EnableStartTLS bool `json:"enableStartTLS,omitempty"`

	// Authentication specifies the email authentication configuration.
	// +optional
	Authentication *EmailAuthentication `json:"authentication,omitempty"`
}

type EmailAuthentication struct {
	// Username specifies login username.
	// +required
	Username SourceRefOrVal `json:"username"`

	// Password specifies login password.
	// +required
	Password SourceRef `json:"password"`
}

type RealmSessions struct {
	// SSOSessionSettings defines the SSO session settings for the realm.
	// +optional
	SSOSessionSettings *RealmSSOSessionSettings `json:"ssoSessionSettings,omitempty"`

	// SSOOfflineSessionSettings defines the SSO offline session settings for the realm.
	// +optional
	SSOOfflineSessionSettings *RealmSSOOfflineSessionSettings `json:"ssoOfflineSessionSettings,omitempty"`

	// SSOLoginSettings defines the SSO login settings for the realm.
	// +optional
	SSOLoginSettings *RealmSSOLoginSettings `json:"ssoLoginSettings,omitempty"`
}

// RealmSSOSessionSettings defines the SSO session settings for the realm.
type RealmSSOSessionSettings struct {
	// IdleTimeout represents the time a session is allowed to be idle before it expires.
	// Tokens and browser sessions are invalidated when a session is expired.
	// +optional
	// +kubebuilder:default=1800
	IdleTimeout int `json:"idleTimeout,omitempty"`

	// MaxLifespan represents the max time before a session is expired.
	// Tokens and browser sessions are invalidated when a session is expired.
	// +optional
	// +kubebuilder:default=36000
	MaxLifespan int `json:"maxLifespan,omitempty"`

	// IdleTimeoutRememberMe represents the time a session is allowed to be idle before it expires.
	// Tokens and browser sessions are invalidated when a session is expired.
	// If not set it uses the standard ssoSessionIdle value.
	// +optional
	// +kubebuilder:default=0
	IdleTimeoutRememberMe int `json:"idleTimeoutRememberMe,omitempty"`

	// MaxLifespanRememberMe represents the max time before a session is expired when a user has set the remember me option.
	// Tokens and browser sessions are invalidated when a session is expired.
	// If not set it uses the standard ssoSessionMax value.
	// +optional
	// +kubebuilder:default=0
	MaxLifespanRememberMe int `json:"maxLifespanRememberMe,omitempty"`
}

// RealmSSOOfflineSessionSettings defines the SSO offline session settings for the realm.
type RealmSSOOfflineSessionSettings struct {
	// IdleTimeout represents the time an offline session is allowed to be idle before it expires.
	// You need to use offline token to refresh at least once within this period; otherwise offline session will expire.
	// +optional
	// +kubebuilder:default=2592000
	IdleTimeout int `json:"idleTimeout,omitempty"`

	// MaxLifespanEnabled enables the offline session maximum lifetime.
	// +optional
	// +kubebuilder:default=false
	MaxLifespanEnabled bool `json:"maxLifespanEnabled,omitempty"`

	// MaxLifespan represents the max time before an offline session is expired regardless of activity.
	// +optional
	// +kubebuilder:default=5184000
	MaxLifespan int `json:"maxLifespan,omitempty"`
}

// RealmEventConfig is the configuration for events in the realm.
type RealmEventConfig struct {
	// AdminEventsDetailsEnabled indicates whether to enable detailed admin events.
	// +optional
	AdminEventsDetailsEnabled bool `json:"adminEventsDetailsEnabled,omitempty"`

	// AdminEventsEnabled indicates whether to enable admin events.
	// +optional
	AdminEventsEnabled bool `json:"adminEventsEnabled,omitempty"`

	// AdminEventsExpiration sets the expiration for events in seconds.
	// Expired events are periodically deleted from the database.
	// +optional
	AdminEventsExpiration int `json:"adminEventsExpiration,omitempty"`

	// EnabledEventTypes is a list of event types to enable.
	// +optional
	// +nullable
	EnabledEventTypes []string `json:"enabledEventTypes,omitempty"`

	// EventsEnabled indicates whether to enable events.
	// +optional
	EventsEnabled bool `json:"eventsEnabled,omitempty"`

	// EventsExpiration is the number of seconds after which events expire.
	// +optional
	EventsExpiration int `json:"eventsExpiration,omitempty"`

	// EventsListeners is a list of event listeners to enable.
	// +optional
	// +nullable
	EventsListeners []string `json:"eventsListeners,omitempty"`
}

// RealmSSOLoginSettings defines the SSO login settings for the realm.
type RealmSSOLoginSettings struct {
	// AccessCodeLifespanLogin represents the max time a user has to complete a login. This is recommended to be relatively long, such as 30 minutes or more.
	// +optional
	// +kubebuilder:default=1800
	AccessCodeLifespanLogin int `json:"accessCodeLifespanLogin,omitempty"`

	// AccessCodeLifespanUserAction represents the max time a user has to complete login related actions like update password or configure totp. This is recommended to be relatively long, such as 5 minutes or more.
	// +optional
	// +kubebuilder:default=300
	AccessCodeLifespanUserAction int `json:"accessCodeLifespanUserAction,omitempty"`
}

// WebAuthnPolicySettings defines the WebAuthPolicy settings for the realm.
type WebAuthnPolicySettings struct {
	// RpEntityName rp entity name
	// +optional
	RpEntityName string `json:"rpEntityName,omitempty"`

	// SignatureAlgorithms is a list of signature algorithms.
	// +optional
	// +nullable
	SignatureAlgorithms []string `json:"signatureAlgorithms,omitempty"`

	// +optional
	RpID string `json:"rpId,omitempty"`

	// +optional
	AttestationConveyancePreference string `json:"attestationConveyancePreference,omitempty"`

	// +optional
	AuthenticatorAttachment string `json:"authenticatorAttachment,omitempty"`

	// +optional
	RequireResidentKey string `json:"requireResidentKey,omitempty"`

	// +optional
	UserVerificationRequirement string `json:"userVerificationRequirement,omitempty"`

	// +optional
	CreateTimeout int `json:"createTimeout,omitempty"`

	// +optional
	AvoidSameAuthenticatorRegister bool `json:"avoidSameAuthenticatorRegister,omitempty"`

	// +optional
	// +nullable
	AcceptableAaguids []string `json:"acceptableAaguids,omitempty"`

	// +optional
	PasswordlessRpEntityName string `json:"passwordlessRpEntityName,omitempty"`

	// +optional
	// +nullable
	PasswordlessSignatureAlgorithms []string `json:"passwordlessSignatureAlgorithms,omitempty"`

	// +optional
	PasswordlessRpID string `json:"passwordlessRpId,omitempty"`

	// +optional
	PasswordlessAttestationConveyancePreference string `json:"passwordlessAttestationConveyancePreference,omitempty"`

	// +optional
	PasswordlessAuthenticatorAttachment string `json:"passwordlessAuthenticatorAttachment,omitempty"`

	// +optional
	PasswordlessRequireResidentKey string `json:"passwordlessRequireResidentKey,omitempty"`

	// +optional
	PasswordlessUserVerificationRequirement string `json:"passwordlessUserVerificationRequirement,omitempty"`

	// +optional
	PasswordlessCreateTimeout int `json:"passwordlessCreateTimeout,omitempty"`

	// +optional
	PasswordlessAvoidSameAuthenticatorRegister bool `json:"passwordlessAvoidSameAuthenticatorRegister,omitempty"`

	// +optional
	// +nullable
	PasswordlessAcceptableAaguids []string `json:"passwordlessAcceptableAaguids,omitempty"`

	// +optional
	// +nullable
	ExtraOrigins []string `json:"extraOrigins,omitempty"`

	// +optional
	// +nullable
	PasswordlessExtraOrigins []string `json:"passwordlessExtraOrigins,omitempty"`
}

// Oauth2DeviceSettings defines the Oauth2Device settings for the realm.
type Oauth2DeviceSettings struct {
	// +optional
	// +kubebuilder:default=600
	CodeLifespan int `json:"codeLifespan,omitempty"`

	// +optional
	// +kubebuilder:default=5
	PollingInterval int `json:"pollingInterval,omitempty"`
}

type OTPPolicySettings struct {
	// +optional
	// +kubebuilder:default=HmacSHA1
	Algorithm string `json:"algorithm,omitempty"`

	// +optional
	// +kubebuilder:default=false
	CodeReusable bool `json:"codeReusable,omitempty"`

	// +optional
	// +kubebuilder:default=6
	Digits int `json:"digits,omitempty"`

	// +optional
	// +kubebuilder:default=0
	InitialCounter int `json:"initialCounter,omitempty"`

	// +optional
	// +kubebuilder:default=1
	LookAheadWindow int `json:"lookAheadWindow,omitempty"`

	// +optional
	// +kubebuilder:default=30
	Period int `json:"period,omitempty"`

	// +optional
	// +kubebuilder:default=totp
	Type string `json:"type,omitempty"`

	// +optional
	// +nullable
	SupportedApplications []string `json:"supportedApplications,omitempty"`
}

type ClientSessionSettings struct {
	// +optional
	// +kubebuilder:default=0
	IdleTimeout int `json:"idleTimeout,omitempty"`

	// +optional
	// +kubebuilder:default=0
	MaxLifespan int `json:"maxLifespan,omitempty"`

	// +optional
	// +kubebuilder:default=0
	OfflineIdleTimeout int `json:"offlineIdleTimeout,omitempty"`

	// +optional
	// +kubebuilder:default=0
	OfflineMaxLifespan int `json:"offlineMaxLifespan,omitempty"`
}

type RealmRoleRepresentation struct {
	// +optional
	// +nullable
	Attributes *map[string][]string `json:"attributes,omitempty"`

	// +optional
	ClientRole bool `json:"clientRole,omitempty"`

	// +optional
	Composite bool `json:"composite,omitempty"`

	// +optional
	Composites *Composites `json:"composites,omitempty"`

	// +optional
	ContainerId string `json:"containerId,omitempty"`

	// +optional
	Description string `json:"description,omitempty"`

	// +optional
	Id string `json:"id,omitempty"`

	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	ScopeParamRequired bool `json:"scopeParamRequired,omitempty"`
}

type Composites struct {
	// +optional
	// +nullable
	Application *map[string][]string `json:"application,omitempty"`

	// +optional
	// +nullable
	Client *map[string][]string `json:"client,omitempty"`

	// +optional
	// +nullable
	Realm []string `json:"realm,omitempty"`
}
