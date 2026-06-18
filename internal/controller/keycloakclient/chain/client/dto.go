package client

import (
	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	baseDTO "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/dto"
)

const defaultClientProtocol = "openid-connect"

func ConvertDataClientToClient(
	spec *keycloakApi.Client,
	clientSecret,
	realmName string,
	authFlowOverrides map[string]string,
) *baseDTO.Client {
	// Convert ClientRolesV2 to DTO ClientRole format
	roles := make([]baseDTO.ClientRole, 0, len(spec.ClientRolesV2))

	for _, role := range spec.ClientRolesV2 {
		if role.Name != "" {
			dtoRole := baseDTO.ClientRole{
				Name:                  role.Name,
				Description:           role.Description,
				AssociatedClientRoles: role.AssociatedClientRoles,
			}
			roles = append(roles, dtoRole)
		}
	}

	return &baseDTO.Client{
		RealmName:                          realmName,
		ClientId:                           spec.ClientId,
		ClientSecret:                       clientSecret,
		Roles:                              roles,
		PublicClient:                       spec.Public,
		DirectAccess:                       spec.DirectAccess,
		WebUrl:                             spec.WebUrl,
		AdminUrl:                           spec.AdminUrl,
		HomeUrl:                            spec.HomeUrl,
		Protocol:                           getValueOrDefault(spec.Protocol),
		Attributes:                         spec.Attributes,
		AdvancedProtocolMappers:            spec.AdvancedProtocolMappers,
		ServiceAccountEnabled:              spec.ServiceAccount != nil && spec.ServiceAccount.Enabled,
		FrontChannelLogout:                 spec.FrontChannelLogout,
		RedirectUris:                       spec.RedirectUris,
		WebOrigins:                         spec.WebOrigins,
		ImplicitFlowEnabled:                spec.ImplicitFlowEnabled,
		AuthorizationServicesEnabled:       spec.AuthorizationServicesEnabled,
		BearerOnly:                         spec.BearerOnly,
		ClientAuthenticatorType:            spec.ClientAuthenticatorType,
		ConsentRequired:                    spec.ConsentRequired,
		Description:                        spec.Description,
		Enabled:                            spec.Enabled,
		FullScopeAllowed:                   spec.FullScopeAllowed,
		Name:                               spec.Name,
		StandardFlowEnabled:                spec.StandardFlowEnabled,
		SurrogateAuthRequired:              spec.SurrogateAuthRequired,
		AuthenticationFlowBindingOverrides: authFlowOverrides,
	}
}

func getValueOrDefault(protocol *string) string {
	if protocol == nil {
		return defaultClientProtocol
	}

	return *protocol
}
