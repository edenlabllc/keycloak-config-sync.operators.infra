package v1alpha1

const (
	ReconciliationStrategyFull    = "full"
	ReconciliationStrategyAddOnly = "addOnly"

	KeycloakClientScopeTypeDefault  = "default"
	KeycloakClientScopeTypeOptional = "optional"
	KeycloakClientScopeTypeNone     = "none"

	// ClientSecretKey is a key for client secret in secret data.
	ClientSecretKey = "clientSecret"

	// TODO: ??
	KeycloakAdminTypeUser           = "user"
	KeycloakAdminTypeServiceAccount = "serviceAccount"
)
