package adapter

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
)

func (a GoCloakAdapter) SyncRealmScopeMapping(ctx context.Context,
	realmName string,
	scopeID string,
	roles []RealmRole,
) error {
	if err := a.createOrDeleteRoleScopeMapping(ctx, realmName, scopeID, roles); err != nil {
		return fmt.Errorf("error during createOrDeleteRoleScopeMapping: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) createOrDeleteRoleScopeMapping(
	ctx context.Context, realmName string, scopeID string, rols []RealmRole,
) error {
	existingRoles, err := a.GetRealmRoles(ctx, realmName)
	if err != nil {
		return fmt.Errorf("failed to get realm roles: %w", err)
	}

	currentClientRoles, err := prepareRoles(rols, existingRoles)
	if err != nil {
		return fmt.Errorf("error during createOrDeleteRoleScope: %w", err)
	}

	existingRealmRolesWithScope, err := a.GetScopeMappingRealmRoles(ctx, realmName, scopeID)
	if err != nil {
		return fmt.Errorf("failed to get realm roles by scope: %w", err)
	}

	filterRealmRolesWithScope := a.filterRoles(existingRealmRolesWithScope, false)

	if err := a.client.CreateClientScopesScopeMappingsRealmRoles(
		ctx,
		a.token.AccessToken,
		realmName,
		scopeID,
		a.needCreateRoles(currentClientRoles, filterRealmRolesWithScope),
	); err != nil {
		return fmt.Errorf("error during create client scope mapping client roles: %w", err)
	}

	needDeleteRoles := a.needDeleteRoles(currentClientRoles, filterRealmRolesWithScope)
	if len(needDeleteRoles) == 0 {
		return nil
	}

	if err := a.client.DeleteClientScopesScopeMappingsRealmRoles(
		ctx,
		a.token.AccessToken,
		realmName,
		scopeID,
		needDeleteRoles,
	); err != nil {
		return fmt.Errorf("error during delete client scope mapping client roles: %w", err)
	}

	return nil
}

func prepareRoles(currRoles []RealmRole,
	existingRealmRoles map[string]gocloak.Role,
) (map[string]gocloak.Role, error) {
	result := make(map[string]gocloak.Role)

	for _, cRole := range currRoles {
		obj, ok := checkFullRoleNameByRealmRoleMatch(cRole.Name, existingRealmRoles)

		if !ok {
			return result, fmt.Errorf("failed to find role with name %s", cRole.Name)
		}

		result[cRole.Name] = *obj
	}

	return result, nil
}

func checkFullRoleNameByRealmRoleMatch(role string, roles map[string]gocloak.Role) (*gocloak.Role, bool) {
	if len(roles) == 0 {
		return nil, false
	}

	if val, ok := roles[role]; ok {
		return &val, true
	}

	return nil, false
}

func (a GoCloakAdapter) GetScopeMappingRealmRoles(ctx context.Context,
	realm, scopeID string,
) (map[string]gocloak.Role, error) {
	roles, err := a.client.GetClientScopesScopeMappingsRealmRoles(
		ctx,
		a.token.AccessToken,
		realm,
		scopeID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get realm roles by scope: %w", err)
	}

	rolesMap := make(map[string]gocloak.Role, len(roles))

	for _, r := range roles {
		if r != nil && r.Name != nil {
			rolesMap[*r.Name] = *r
		}
	}

	return rolesMap, nil
}

func (a GoCloakAdapter) needCreateRoles(
	currentRoles, clientScopeRoles map[string]gocloak.Role,
) []gocloak.Role {
	newRoles := make([]gocloak.Role, 0)

	for cRoleName, cRole := range currentRoles {
		if _, ok := clientScopeRoles[cRoleName]; !ok {
			newRoles = append(newRoles, cRole)
		}
	}

	return newRoles
}

func (a GoCloakAdapter) needDeleteRoles(
	currentRoles, clientScopeRoles map[string]gocloak.Role,
) []gocloak.Role {
	delRoles := make([]gocloak.Role, 0)

	for cRoleName, cRoleVal := range clientScopeRoles {
		if _, ok := currentRoles[cRoleName]; !ok {
			delRoles = append(delRoles, cRoleVal)
		}
	}

	return delRoles
}

func (a GoCloakAdapter) DeleteScopeMappingRoles(ctx context.Context,
	realmName string,
	scopeID string,
	currRoles []RealmRole,
) error {
	existingRealmRolesWithScope, err := a.GetScopeMappingRealmRoles(ctx, realmName, scopeID)
	if err != nil {
		return fmt.Errorf("failed to get realm roles by scope: %w", err)
	}

	return a.client.DeleteClientScopesScopeMappingsRealmRoles(
		ctx,
		a.token.AccessToken,
		realmName,
		scopeID,
		a.findRoleMath(currRoles, a.filterRoles(existingRealmRolesWithScope, false)),
	)
}

// nolint:unused
func convertMapRoleToRoleValues(roles map[string]gocloak.Role) []gocloak.Role {
	result := make([]gocloak.Role, 0, len(roles))

	for _, role := range roles {
		if role.Name != nil {
			result = append(result, role)
		}
	}

	return result
}
