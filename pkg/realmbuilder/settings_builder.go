package realmbuilder

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"k8s.io/utils/ptr"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

// commonRealmSpec holds the normalized, API-version-agnostic fields shared by
// KeycloakRealmSpec and ClusterKeycloakRealmSpec.
type commonRealmSpec struct {
	DisplayName                  string
	DisplayHTMLName              string
	SslRequired                  string
	UseOrganizations             bool
	OrganizationsEnabled         bool
	FrontendURL                  string
	BrowserSecurityHeaders       *map[string]string
	PasswordPolicy               string // pre-formatted "type(value) and …" string, empty if none
	RequiredCredentials          []string
	TokenSettings                *common.TokenSettings
	RealmEventConfig             *common.RealmEventConfig
	Login                        *keycloakApiAlpha.RealmLogin
	Sessions                     *common.RealmSessions
	WebAuthnPolicySettings       *common.WebAuthnPolicySettings
	Oauth2DeviceSettings         *common.Oauth2DeviceSettings
	OTPPolicySettings            *common.OTPPolicySettings
	ClientSessionSettings        *common.ClientSessionSettings
	DefaultRole                  *common.RealmRoleRepresentation
	LoginTheme                   *string
	AccountTheme                 *string
	AdminTheme                   *string
	EmailTheme                   *string
	InternationalizationEnabled  *bool
	BruteForceProtected          bool
	PermanentLockout             bool
	MaxFailureWaitSeconds        int
	MinimumQuickLoginWaitSeconds int
	WaitIncrementSeconds         int
	QuickLoginCheckMilliSeconds  int64
	MaxDeltaTimeSeconds          int
	FailureFactor                int
	SupportedLocales             []string
	RegistrationFlow             string
	DirectGrantFlow              string
	ResetCredentialsFlow         string
	ClientAuthenticationFlow     string
	DockerAuthenticationFlow     string
	UserManagedAccessAllowed     bool
	Attributes                   *map[string]string
}

// ApplyRealmEventConfig sets the realm event configuration in Keycloak.
// It is a no-op if cfg is nil.
func ApplyRealmEventConfig(
	ctx context.Context,
	realmName string,
	cfg *common.RealmEventConfig,
	realmClient keycloakv2.RealmClient,
) error {
	if cfg == nil {
		return nil
	}

	rep := keycloakv2.RealmEventsConfigRepresentation{
		AdminEventsDetailsEnabled: ptr.To(cfg.AdminEventsDetailsEnabled),
		AdminEventsEnabled:        ptr.To(cfg.AdminEventsEnabled),
		EventsEnabled:             ptr.To(cfg.EventsEnabled),
		EventsExpiration:          ptr.To(int64(cfg.EventsExpiration)),
	}

	if cfg.EnabledEventTypes != nil {
		rep.EnabledEventTypes = &cfg.EnabledEventTypes
	}

	if cfg.EventsListeners != nil {
		rep.EventsListeners = &cfg.EventsListeners
	}

	if _, err := realmClient.SetRealmEventConfig(ctx, realmName, rep); err != nil {
		return fmt.Errorf("unable to set realm event config: %w", err)
	}

	return nil
}

// ApplyRealmSettings fetches the current realm from Keycloak, merges the overlay into it,
// and writes it back.
func ApplyRealmSettings(
	ctx context.Context,
	realmName string,
	overlay keycloakv2.RealmRepresentation,
	realmClient keycloakv2.RealmClient,
) error {
	current, _, err := realmClient.GetRealm(ctx, realmName)
	if err != nil {
		return fmt.Errorf("unable to get realm: %w", err)
	}

	MergeRealmRepresentation(current, &overlay)

	if _, err := realmClient.UpdateRealm(ctx, realmName, *current); err != nil {
		return fmt.Errorf("unable to update realm settings: %w", err)
	}

	return nil
}

// nolint:dupl
// TODO: need fix this fields, add option if useOrganizations == true, use this variable OrganizationsEnabled
// BuildRealmRepresentationFromV1 builds a keycloakv2.RealmRepresentation with only the
// operator-managed fields populated from a v1.KeycloakRealm spec.
func BuildRealmRepresentationFromV1(realm *keycloakApiAlpha.KeycloakRealm) keycloakv2.RealmRepresentation {
	spec := &realm.Spec

	c := commonRealmSpec{
		DisplayName:                  spec.DisplayName,
		RequiredCredentials:          spec.RequiredCredentials,
		SslRequired:                  spec.SSLRequired,
		DisplayHTMLName:              spec.DisplayHTMLName,
		FrontendURL:                  spec.FrontendURL,
		BrowserSecurityHeaders:       spec.BrowserSecurityHeaders,
		TokenSettings:                spec.TokenSettings,
		RealmEventConfig:             spec.RealmEventConfig,
		Login:                        spec.Login,
		WebAuthnPolicySettings:       spec.WebAuthnPolicySettings,
		Oauth2DeviceSettings:         spec.Oauth2DeviceSettings,
		OTPPolicySettings:            spec.OTPPolicySettings,
		ClientSessionSettings:        spec.ClientSessionSettings,
		Sessions:                     spec.Sessions,
		PasswordPolicy:               buildPasswordPolicy(spec.PasswordPolicies),
		BruteForceProtected:          spec.BruteForceProtected,
		PermanentLockout:             spec.PermanentLockout,
		MaxFailureWaitSeconds:        spec.MaxFailureWaitSeconds,
		MinimumQuickLoginWaitSeconds: spec.MinimumQuickLoginWaitSeconds,
		WaitIncrementSeconds:         spec.WaitIncrementSeconds,
		QuickLoginCheckMilliSeconds:  spec.QuickLoginCheckMilliSeconds,
		MaxDeltaTimeSeconds:          spec.MaxDeltaTimeSeconds,
		FailureFactor:                spec.FailureFactor,
		SupportedLocales:             spec.SupportedLocales,
		RegistrationFlow:             spec.RegistrationFlow,
		DirectGrantFlow:              spec.DirectGrantFlow,
		ResetCredentialsFlow:         spec.ResetCredentialsFlow,
		ClientAuthenticationFlow:     spec.ClientAuthenticationFlow,
		DockerAuthenticationFlow:     spec.DockerAuthenticationFlow,
		UserManagedAccessAllowed:     spec.UserManagedAccessAllowed,
		Attributes:                   spec.Attributes,
		DefaultRole:                  spec.DefaultRole,
	}

	if spec.UseOrganizations {
		c.UseOrganizations = spec.UseOrganizations
		c.OrganizationsEnabled = spec.OrganizationsEnabled
	}

	if spec.Themes != nil {
		c.LoginTheme = spec.Themes.LoginTheme
		c.AccountTheme = spec.Themes.AccountTheme
		c.AdminTheme = spec.Themes.AdminConsoleTheme
		c.EmailTheme = spec.Themes.EmailTheme
		c.InternationalizationEnabled = spec.Themes.InternationalizationEnabled
	}

	return buildRealmRepresentationFromCommon(c)
}

// buildPasswordPolicy formats a slice of PasswordPolicy into the Keycloak string format,
// e.g. "length(8) and upperCase(1)". Returns empty string if the slice is empty.
func buildPasswordPolicy(policies []common.PasswordPolicy) string {
	if len(policies) == 0 {
		return ""
	}

	parts := make([]string, len(policies))
	for i, p := range policies {
		parts[i] = p.Type + "(" + p.Value + ")"
	}

	return strings.Join(parts, " and ")
}

// TODO: need fix this fields, add option if useOrganizations == true, use this variable OrganizationsEnabled
// buildRealmRepresentationFromCommon constructs a RealmRepresentation from the
// normalized common spec. All version-specific field mapping is done by the callers.
func buildRealmRepresentationFromCommon(spec commonRealmSpec) keycloakv2.RealmRepresentation {
	rep := keycloakv2.RealmRepresentation{
		DisplayName:                  ptr.To(spec.DisplayName),
		DisplayNameHtml:              ptr.To(spec.DisplayHTMLName),
		SslRequired:                  ptr.To(spec.SslRequired),
		RequiredCredentials:          ptr.To(spec.RequiredCredentials),
		LoginTheme:                   spec.LoginTheme,
		AccountTheme:                 spec.AccountTheme,
		AdminTheme:                   spec.AdminTheme,
		EmailTheme:                   spec.EmailTheme,
		InternationalizationEnabled:  spec.InternationalizationEnabled,
		BruteForceProtected:          ptr.To(spec.BruteForceProtected),
		PermanentLockout:             ptr.To(spec.PermanentLockout),
		MaxFailureWaitSeconds:        ptr.To(int32(spec.MaxFailureWaitSeconds)),
		MinimumQuickLoginWaitSeconds: ptr.To(int32(spec.MinimumQuickLoginWaitSeconds)),
		WaitIncrementSeconds:         ptr.To(int32(spec.WaitIncrementSeconds)),
		QuickLoginCheckMilliSeconds:  ptr.To(spec.QuickLoginCheckMilliSeconds),
		MaxDeltaTimeSeconds:          ptr.To(int32(spec.MaxDeltaTimeSeconds)),
		FailureFactor:                ptr.To(int32(spec.FailureFactor)),
		SupportedLocales:             ptr.To(spec.SupportedLocales),
		RegistrationFlow:             ptr.To(spec.RegistrationFlow),
		DirectGrantFlow:              ptr.To(spec.DirectGrantFlow),
		ResetCredentialsFlow:         ptr.To(spec.ResetCredentialsFlow),
		ClientAuthenticationFlow:     ptr.To(spec.ClientAuthenticationFlow),
		DockerAuthenticationFlow:     ptr.To(spec.DockerAuthenticationFlow),
		UserManagedAccessAllowed:     ptr.To(spec.UserManagedAccessAllowed),
		Attributes:                   spec.Attributes,
	}

	if spec.UseOrganizations {
		rep.OrganizationsEnabled = ptr.To(spec.OrganizationsEnabled)
	}

	if spec.FrontendURL != "" {
		attrs := make(map[string]string)
		rep.Attributes = &attrs
		(*rep.Attributes)["frontendUrl"] = spec.FrontendURL
	}

	if spec.BrowserSecurityHeaders != nil {
		rep.BrowserSecurityHeaders = spec.BrowserSecurityHeaders
	}

	if spec.PasswordPolicy != "" {
		rep.PasswordPolicy = ptr.To(spec.PasswordPolicy)
	}

	if ts := spec.TokenSettings; ts != nil {
		rep.DefaultSignatureAlgorithm = ptr.To(ts.DefaultSignatureAlgorithm)
		rep.RevokeRefreshToken = ptr.To(ts.RevokeRefreshToken)
		rep.RefreshTokenMaxReuse = ptr.To(int32(ts.RefreshTokenMaxReuse))
		rep.AccessTokenLifespan = ptr.To(int32(ts.AccessTokenLifespan))
		rep.AccessTokenLifespanForImplicitFlow = ptr.To(int32(ts.AccessTokenLifespanForImplicitFlow))
		rep.AccessCodeLifespan = ptr.To(int32(ts.AccessCodeLifespan))
		rep.ActionTokenGeneratedByUserLifespan = ptr.To(int32(ts.ActionTokenGeneratedByUserLifespan))
		rep.ActionTokenGeneratedByAdminLifespan = ptr.To(int32(ts.ActionTokenGeneratedByAdminLifespan))
	}

	if spec.RealmEventConfig != nil && spec.RealmEventConfig.AdminEventsEnabled {
		if rep.Attributes == nil {
			attrs := make(map[string]string)
			rep.Attributes = &attrs
		}

		(*rep.Attributes)["adminEventsExpiration"] = strconv.Itoa(spec.RealmEventConfig.AdminEventsExpiration)
	}

	if l := spec.Login; l != nil {
		rep.RegistrationAllowed = ptr.To(l.UserRegistration)
		rep.ResetPasswordAllowed = ptr.To(l.ForgotPassword)
		rep.RememberMe = ptr.To(l.RememberMe)
		rep.RegistrationEmailAsUsername = ptr.To(l.EmailAsUsername)
		rep.LoginWithEmailAllowed = ptr.To(l.LoginWithEmail)
		rep.DuplicateEmailsAllowed = ptr.To(l.DuplicateEmails)
		rep.VerifyEmail = ptr.To(l.VerifyEmail)
		rep.EditUsernameAllowed = ptr.To(l.EditUsername)
	}

	if webAuth := spec.WebAuthnPolicySettings; webAuth != nil {
		rep.WebAuthnPolicyAcceptableAaguids = ptr.To(webAuth.AcceptableAaguids)
		rep.WebAuthnPolicyAttestationConveyancePreference = ptr.To(webAuth.AttestationConveyancePreference)
		rep.WebAuthnPolicyAuthenticatorAttachment = ptr.To(webAuth.AuthenticatorAttachment)
		rep.WebAuthnPolicyAvoidSameAuthenticatorRegister = ptr.To(webAuth.AvoidSameAuthenticatorRegister)
		rep.WebAuthnPolicyCreateTimeout = ptr.To(int32(webAuth.CreateTimeout))
		rep.WebAuthnPolicyPasswordlessAcceptableAaguids = ptr.To(webAuth.PasswordlessAcceptableAaguids)
		rep.WebAuthnPolicyPasswordlessAttestationConveyancePreference = ptr.To(
			webAuth.PasswordlessAttestationConveyancePreference,
		)
		rep.WebAuthnPolicyPasswordlessAuthenticatorAttachment = ptr.To(webAuth.PasswordlessAuthenticatorAttachment)
		rep.WebAuthnPolicyPasswordlessAvoidSameAuthenticatorRegister = ptr.To(
			webAuth.PasswordlessAvoidSameAuthenticatorRegister,
		)
		rep.WebAuthnPolicyPasswordlessCreateTimeout = ptr.To(int32(webAuth.PasswordlessCreateTimeout))
		rep.WebAuthnPolicyPasswordlessRequireResidentKey = ptr.To(webAuth.PasswordlessRequireResidentKey)
		rep.WebAuthnPolicyPasswordlessRpEntityName = ptr.To(webAuth.PasswordlessRpEntityName)
		rep.WebAuthnPolicyPasswordlessRpId = ptr.To(webAuth.PasswordlessRpID)
		rep.WebAuthnPolicyPasswordlessSignatureAlgorithms = ptr.To(webAuth.PasswordlessSignatureAlgorithms)
		rep.WebAuthnPolicyPasswordlessUserVerificationRequirement = ptr.To(webAuth.PasswordlessUserVerificationRequirement)
		rep.WebAuthnPolicyRequireResidentKey = ptr.To(webAuth.RequireResidentKey)
		rep.WebAuthnPolicyRpEntityName = ptr.To(webAuth.RpEntityName)
		rep.WebAuthnPolicyRpId = ptr.To(webAuth.RpID)
		rep.WebAuthnPolicySignatureAlgorithms = ptr.To(webAuth.SignatureAlgorithms)
		rep.WebAuthnPolicyUserVerificationRequirement = ptr.To(webAuth.UserVerificationRequirement)
		rep.WebAuthnPolicyExtraOrigins = ptr.To(webAuth.ExtraOrigins)
		rep.WebAuthnPolicyPasswordlessExtraOrigins = ptr.To(webAuth.PasswordlessExtraOrigins)
	}

	if oauth2Device := spec.Oauth2DeviceSettings; oauth2Device != nil {
		rep.Oauth2DevicePollingInterval = ptr.To(int32(oauth2Device.PollingInterval))
		rep.Oauth2DeviceCodeLifespan = ptr.To(int32(oauth2Device.CodeLifespan))
	}

	if otp := spec.OTPPolicySettings; otp != nil {
		rep.OtpPolicyAlgorithm = ptr.To(otp.Algorithm)
		rep.OtpPolicyCodeReusable = ptr.To(otp.CodeReusable)
		rep.OtpPolicyDigits = ptr.To(int32(otp.Digits))
		rep.OtpPolicyInitialCounter = ptr.To(int32(otp.InitialCounter))
		rep.OtpPolicyLookAheadWindow = ptr.To(int32(otp.LookAheadWindow))
		rep.OtpPolicyPeriod = ptr.To(int32(otp.Period))
		rep.OtpPolicyType = ptr.To(otp.Type)
		rep.OtpSupportedApplications = ptr.To(otp.SupportedApplications)
	}

	if clientSession := spec.ClientSessionSettings; clientSession != nil {
		rep.ClientSessionIdleTimeout = ptr.To(int32(clientSession.IdleTimeout))
		rep.ClientSessionMaxLifespan = ptr.To(int32(clientSession.MaxLifespan))
		rep.ClientOfflineSessionIdleTimeout = ptr.To(int32(clientSession.OfflineIdleTimeout))
		rep.ClientOfflineSessionMaxLifespan = ptr.To(int32(clientSession.OfflineMaxLifespan))
	}

	if defaultRole := spec.DefaultRole; defaultRole != nil {
		rep.DefaultRole = &keycloakv2.RoleRepresentation{
			Attributes:         defaultRole.Attributes,
			ClientRole:         ptr.To(defaultRole.ClientRole),
			Composite:          ptr.To(defaultRole.Composite),
			ContainerId:        ptr.To(defaultRole.ContainerId),
			Description:        ptr.To(defaultRole.Description),
			Id:                 ptr.To(defaultRole.Id),
			Name:               ptr.To(defaultRole.Name),
			ScopeParamRequired: ptr.To(defaultRole.ScopeParamRequired),
		}
	}

	setRealmRepSessionSettings(&rep, spec.Sessions)

	return rep
}

// MergeRealmRepresentation copies only the operator-managed fields from overlay onto base,
// merging map fields key-by-key to preserve live Keycloak values the operator doesn't manage.
func MergeRealmRepresentation(base, overlay *keycloakv2.RealmRepresentation) {
	mergeRealmAppearance(base, overlay)
	mergeRealmTokenSettings(base, overlay)
	mergeRealmLoginSettings(base, overlay)
	mergeRealmSessionSettings(base, overlay)
	mergeRealmWebAuthnPolicySettings(base, overlay)
	mergeRealmOTPPolicySettings(base, overlay)
	mergeRealmClientSessionSettings(base, overlay)
	mergeRealmMaps(base, overlay)
}

// nolint:staticcheck
func mergeRealmAppearance(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.DisplayName, &overlay.DisplayName)
	mergePtr(&base.DisplayNameHtml, &overlay.DisplayNameHtml)
	mergePtr(&base.OrganizationsEnabled, &overlay.OrganizationsEnabled)
	mergePtr(&base.LoginTheme, &overlay.LoginTheme)
	mergePtr(&base.AccountTheme, &overlay.AccountTheme)
	mergePtr(&base.AdminTheme, &overlay.AdminTheme)
	mergePtr(&base.EmailTheme, &overlay.EmailTheme)
	mergePtr(&base.InternationalizationEnabled, &overlay.InternationalizationEnabled)
	mergePtr(&base.PasswordPolicy, &overlay.PasswordPolicy)
	mergePtr(&base.SslRequired, &overlay.SslRequired)
	mergePtr(&base.BruteForceProtected, &overlay.BruteForceProtected)
	mergePtr(&base.PermanentLockout, &overlay.PermanentLockout)
	mergePtr(&base.MaxFailureWaitSeconds, &overlay.MaxFailureWaitSeconds)
	mergePtr(&base.MinimumQuickLoginWaitSeconds, &overlay.MinimumQuickLoginWaitSeconds)
	mergePtr(&base.WaitIncrementSeconds, &overlay.WaitIncrementSeconds)
	mergePtr(&base.QuickLoginCheckMilliSeconds, &overlay.QuickLoginCheckMilliSeconds)
	mergePtr(&base.MaxDeltaTimeSeconds, &overlay.MaxDeltaTimeSeconds)
	mergePtr(&base.FailureFactor, &overlay.FailureFactor)
	mergePtr(&base.SupportedLocales, &overlay.SupportedLocales)
	mergePtr(&base.RegistrationFlow, &overlay.RegistrationFlow)
	mergePtr(&base.DirectGrantFlow, &overlay.DirectGrantFlow)
	mergePtr(&base.ResetCredentialsFlow, &overlay.ResetCredentialsFlow)
	mergePtr(&base.ClientAuthenticationFlow, &overlay.ClientAuthenticationFlow)
	mergePtr(&base.DockerAuthenticationFlow, &overlay.DockerAuthenticationFlow)
	mergePtr(&base.UserManagedAccessAllowed, &overlay.UserManagedAccessAllowed)
	mergePtr(&base.DefaultRole, &overlay.DefaultRole)
	mergePtr(&base.RequiredCredentials, &overlay.RequiredCredentials)
}

func mergeRealmTokenSettings(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.DefaultSignatureAlgorithm, &overlay.DefaultSignatureAlgorithm)
	mergePtr(&base.RevokeRefreshToken, &overlay.RevokeRefreshToken)
	mergePtr(&base.RefreshTokenMaxReuse, &overlay.RefreshTokenMaxReuse)
	mergePtr(&base.AccessTokenLifespan, &overlay.AccessTokenLifespan)
	mergePtr(&base.AccessTokenLifespanForImplicitFlow, &overlay.AccessTokenLifespanForImplicitFlow)
	mergePtr(&base.AccessCodeLifespan, &overlay.AccessCodeLifespan)
	mergePtr(&base.ActionTokenGeneratedByUserLifespan, &overlay.ActionTokenGeneratedByUserLifespan)
	mergePtr(&base.ActionTokenGeneratedByAdminLifespan, &overlay.ActionTokenGeneratedByAdminLifespan)
}

func mergeRealmWebAuthnPolicySettings(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.WebAuthnPolicyAcceptableAaguids, &overlay.WebAuthnPolicyAcceptableAaguids)
	mergePtr(&base.WebAuthnPolicyAttestationConveyancePreference, &overlay.WebAuthnPolicyAttestationConveyancePreference)
	mergePtr(&base.WebAuthnPolicyAuthenticatorAttachment, &overlay.WebAuthnPolicyAuthenticatorAttachment)
	mergePtr(&base.WebAuthnPolicyAvoidSameAuthenticatorRegister, &overlay.WebAuthnPolicyAvoidSameAuthenticatorRegister)
	mergePtr(&base.WebAuthnPolicyCreateTimeout, &overlay.WebAuthnPolicyCreateTimeout)
	mergePtr(&base.WebAuthnPolicyPasswordlessAcceptableAaguids, &overlay.WebAuthnPolicyPasswordlessAcceptableAaguids)
	mergePtr(&base.WebAuthnPolicyPasswordlessAttestationConveyancePreference,
		&overlay.WebAuthnPolicyPasswordlessAttestationConveyancePreference)
	mergePtr(&base.WebAuthnPolicyPasswordlessAuthenticatorAttachment,
		&overlay.WebAuthnPolicyPasswordlessAuthenticatorAttachment)
	mergePtr(&base.WebAuthnPolicyPasswordlessAvoidSameAuthenticatorRegister,
		&overlay.WebAuthnPolicyPasswordlessAvoidSameAuthenticatorRegister)
	mergePtr(&base.WebAuthnPolicyPasswordlessCreateTimeout, &overlay.WebAuthnPolicyPasswordlessCreateTimeout)
	mergePtr(&base.WebAuthnPolicyPasswordlessRequireResidentKey, &overlay.WebAuthnPolicyPasswordlessRequireResidentKey)
	mergePtr(&base.WebAuthnPolicyPasswordlessRpEntityName, &overlay.WebAuthnPolicyPasswordlessRpEntityName)
	mergePtr(&base.WebAuthnPolicyPasswordlessRpId, &overlay.WebAuthnPolicyPasswordlessRpId)
	mergePtr(&base.WebAuthnPolicyPasswordlessSignatureAlgorithms, &overlay.WebAuthnPolicyPasswordlessSignatureAlgorithms)
	mergePtr(&base.WebAuthnPolicyPasswordlessUserVerificationRequirement,
		&overlay.WebAuthnPolicyPasswordlessUserVerificationRequirement)
	mergePtr(&base.WebAuthnPolicyRequireResidentKey, &overlay.WebAuthnPolicyRequireResidentKey)
	mergePtr(&base.WebAuthnPolicyRpEntityName, &overlay.WebAuthnPolicyRpEntityName)
	mergePtr(&base.WebAuthnPolicyRpId, &overlay.WebAuthnPolicyRpId)
	mergePtr(&base.WebAuthnPolicySignatureAlgorithms, &overlay.WebAuthnPolicySignatureAlgorithms)
	mergePtr(&base.WebAuthnPolicyUserVerificationRequirement, &overlay.WebAuthnPolicyUserVerificationRequirement)
}

func mergeRealmLoginSettings(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.RegistrationAllowed, &overlay.RegistrationAllowed)
	mergePtr(&base.ResetPasswordAllowed, &overlay.ResetPasswordAllowed)
	mergePtr(&base.RememberMe, &overlay.RememberMe)
	mergePtr(&base.RegistrationEmailAsUsername, &overlay.RegistrationEmailAsUsername)
	mergePtr(&base.LoginWithEmailAllowed, &overlay.LoginWithEmailAllowed)
	mergePtr(&base.DuplicateEmailsAllowed, &overlay.DuplicateEmailsAllowed)
	mergePtr(&base.VerifyEmail, &overlay.VerifyEmail)
	mergePtr(&base.EditUsernameAllowed, &overlay.EditUsernameAllowed)
}

func mergeRealmSessionSettings(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.SsoSessionIdleTimeout, &overlay.SsoSessionIdleTimeout)
	mergePtr(&base.SsoSessionMaxLifespan, &overlay.SsoSessionMaxLifespan)
	mergePtr(&base.SsoSessionIdleTimeoutRememberMe, &overlay.SsoSessionIdleTimeoutRememberMe)
	mergePtr(&base.SsoSessionMaxLifespanRememberMe, &overlay.SsoSessionMaxLifespanRememberMe)
	mergePtr(&base.OfflineSessionIdleTimeout, &overlay.OfflineSessionIdleTimeout)
	mergePtr(&base.OfflineSessionMaxLifespanEnabled, &overlay.OfflineSessionMaxLifespanEnabled)
	mergePtr(&base.OfflineSessionMaxLifespan, &overlay.OfflineSessionMaxLifespan)
	mergePtr(&base.AccessCodeLifespanLogin, &overlay.AccessCodeLifespanLogin)
	mergePtr(&base.AccessCodeLifespanUserAction, &overlay.AccessCodeLifespanUserAction)
}

func mergeRealmOTPPolicySettings(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.OtpPolicyAlgorithm, &overlay.OtpPolicyAlgorithm)
	mergePtr(&base.OtpPolicyCodeReusable, &overlay.OtpPolicyCodeReusable)
	mergePtr(&base.OtpPolicyDigits, &overlay.OtpPolicyDigits)
	mergePtr(&base.OtpPolicyInitialCounter, &overlay.OtpPolicyInitialCounter)
	mergePtr(&base.OtpPolicyLookAheadWindow, &overlay.OtpPolicyLookAheadWindow)
	mergePtr(&base.OtpPolicyPeriod, &overlay.OtpPolicyPeriod)
	mergePtr(&base.OtpPolicyType, &overlay.OtpPolicyType)
	mergePtr(&base.OtpSupportedApplications, &overlay.OtpSupportedApplications)
}

func mergeRealmClientSessionSettings(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.ClientSessionIdleTimeout, &overlay.ClientSessionIdleTimeout)
	mergePtr(&base.ClientSessionMaxLifespan, &overlay.ClientSessionMaxLifespan)
	mergePtr(&base.ClientOfflineSessionIdleTimeout, &overlay.ClientOfflineSessionIdleTimeout)
	mergePtr(&base.ClientOfflineSessionMaxLifespan, &overlay.ClientOfflineSessionMaxLifespan)
}

func setRealmRepSessionSettings(rep *keycloakv2.RealmRepresentation, sessions *common.RealmSessions) {
	if sessions == nil {
		return
	}

	if s := sessions.SSOSessionSettings; s != nil {
		rep.SsoSessionIdleTimeout = ptr.To(int32(s.IdleTimeout))
		rep.SsoSessionMaxLifespan = ptr.To(int32(s.MaxLifespan))
		rep.SsoSessionIdleTimeoutRememberMe = ptr.To(int32(s.IdleTimeoutRememberMe))
		rep.SsoSessionMaxLifespanRememberMe = ptr.To(int32(s.MaxLifespanRememberMe))
	}

	if s := sessions.SSOOfflineSessionSettings; s != nil {
		rep.OfflineSessionIdleTimeout = ptr.To(int32(s.IdleTimeout))
		rep.OfflineSessionMaxLifespanEnabled = ptr.To(s.MaxLifespanEnabled)
		rep.OfflineSessionMaxLifespan = ptr.To(int32(s.MaxLifespan))
	}

	if s := sessions.SSOLoginSettings; s != nil {
		rep.AccessCodeLifespanLogin = ptr.To(int32(s.AccessCodeLifespanLogin))
		rep.AccessCodeLifespanUserAction = ptr.To(int32(s.AccessCodeLifespanUserAction))
	}
}

// mergePtr copies *overlay into *base only when *overlay is non-nil.
func mergePtr[T any](base, overlay **T) {
	if *overlay != nil {
		*base = *overlay
	}
}

func mergeRealmMaps(base, overlay *keycloakv2.RealmRepresentation) {
	// BrowserSecurityHeaders: merge keys into base map
	if overlay.BrowserSecurityHeaders != nil {
		if base.BrowserSecurityHeaders == nil {
			m := make(map[string]string)
			base.BrowserSecurityHeaders = &m
		}

		maps.Copy(*base.BrowserSecurityHeaders, *overlay.BrowserSecurityHeaders)
	}

	// Attributes: merge keys into base map
	if overlay.Attributes != nil {
		if base.Attributes == nil {
			attrs := make(map[string]string)
			base.Attributes = &attrs
		}

		maps.Copy(*base.Attributes, *overlay.Attributes)
	}
}
