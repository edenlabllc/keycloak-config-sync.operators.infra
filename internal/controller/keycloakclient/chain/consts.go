package chain

const (
	// ConditionReady indicates the overall readiness of the KeycloakClient.
	// This is the primary condition that summarizes all chain steps.
	ConditionReady = "Ready"

	// Individual chain step conditions - one per step
	ConditionClientSynced                        = "ClientSynced"                        // PutClient
	ConditionClientRolesSynced                   = "ClientRolesSynced"                   // PutClientRole
	ConditionRealmRolesSynced                    = "RealmRolesSynced"                    // PutRealmRole
	ConditionClientScopesSynced                  = "ClientScopesSynced"                  // PutClientScope
	ConditionProtocolMappersSynced               = "ProtocolMappersSynced"               // PutProtocolMappers
	ConditionServiceAccountSynced                = "ServiceAccountSynced"                // ServiceAccount
	ConditionAuthorizationScopesSynced           = "AuthorizationScopesSynced"           // ProcessScope
	ConditionAuthorizationResourcesSynced        = "AuthorizationResourcesSynced"        // ProcessResources
	ConditionAuthorizationPoliciesSynced         = "AuthorizationPoliciesSynced"         // ProcessPolicy
	ConditionAuthorizationPermissionsSynced      = "AuthorizationPermissionsSynced"      // ProcessPermissions
	ConditionAdminFineGrainedPermissionsV1Synced = "AdminFineGrainedPermissionsV1Synced" // PutAdminFineGrainedPermissions
	ConditionClientRegistrationPolicySynced      = "ClientRegistrationPolicySynced"      // PutAllowedClientScopes

	// Success reasons - one per step
	ReasonClientCreated                       = "ClientCreated"
	ReasonClientUpdated                       = "ClientUpdated"
	ReasonClientRolesSynced                   = "ClientRolesSynced"
	ReasonRealmRolesSynced                    = "RealmRolesSynced"
	ReasonClientScopesSynced                  = "ClientScopesSynced"
	ReasonProtocolMappersSynced               = "ProtocolMappersSynced"
	ReasonServiceAccountSynced                = "ServiceAccountSynced"
	ReasonAuthorizationScopesSynced           = "AuthorizationScopesSynced"
	ReasonAuthorizationResourcesSynced        = "AuthorizationResourcesSynced"
	ReasonAuthorizationPoliciesSynced         = "AuthorizationPoliciesSynced"
	ReasonAuthorizationPermissionsSynced      = "AuthorizationPermissionsSynced"
	ReasonAdminFineGrainedPermissionsV1Synced = "AdminFineGrainedPermissionsV1Synced"
	ReasonClientRegistrationPolicySynced      = "ClientRegistrationPolicySynced"
	ReasonReconciliationSucceeded             = "ReconciliationSucceeded"

	// Failure reasons - generic
	ReasonKeycloakAPIError   = "KeycloakAPIError"
	ReasonConfigurationError = "ConfigurationError"
	ReasonSecretError        = "SecretError"

	// Skipped reasons (for addOnly strategy or not configured)
	ReasonSkippedAddOnly = "SkippedAddOnly"
	ReasonNotConfigured  = "NotConfigured"
)
