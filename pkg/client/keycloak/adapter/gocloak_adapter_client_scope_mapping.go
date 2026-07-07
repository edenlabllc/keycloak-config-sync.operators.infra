package adapter

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
)

type RealmRole struct {
	Name string `json:"name"`
}

type ScopeMappingOptions struct {
	Client      string
	ClientScope string
}

func (smo *ScopeMappingOptions) isOnlyClientScope() bool {
	return smo.ClientScope != "" && smo.Client == ""
}

func (smo *ScopeMappingOptions) isOnlyClient() bool {
	return smo.ClientScope == "" && smo.Client != ""
}

func (smo *ScopeMappingOptions) checkRequired() bool {
	return smo.ClientScope == "" && smo.Client == ""
}

func (a GoCloakAdapter) SyncRealmClientScopeMapping(ctx context.Context,
	fromClientID string,
	realmName string,
	rols []RealmRole,
	opts ScopeMappingOptions,
) error {
	if opts.checkRequired() {
		return fmt.Errorf("must provide either ClientScope or FromClient")
	}

	if opts.isOnlyClientScope() {
		if err := a.createOrDeleteRoleScope(ctx, realmName, fromClientID, rols, opts); err != nil {
			return fmt.Errorf("error during createOrDeleteRoleScope: %w", err)
		}
	}

	if opts.isOnlyClient() {
		if err := a.createOrDeleteClientRoleToClient(ctx, realmName, fromClientID, rols, opts); err != nil {
			return fmt.Errorf("error during createOrDeleteRoleScope: %w", err)
		}
	}

	return nil
}

func (a GoCloakAdapter) createOrDeleteRoleScope(
	ctx context.Context, realmName string, fromClientID string, rols []RealmRole, opts ScopeMappingOptions,
) error {
	existingClientRolesWithScope := make(map[string]gocloak.Role, 0)

	clientScope, err := a.GetClientScope(ctx, opts.ClientScope, realmName)
	if err != nil && !IsErrNotFound(err) {
		return fmt.Errorf("unable to get client scope: %w", err)
	}

	if err == nil {
		existingClientRolesWithScope, err = a.GetScopeMappingRealmRoles(ctx, realmName, clientScope.ID)
		if err != nil {
			return fmt.Errorf("failed to get realm roles by scope: %w", err)
		}
	}

	existingRoles, err := a.getExistingClientRoles(ctx, realmName, fromClientID)
	if err != nil {
		return fmt.Errorf("failed to get client roles: %w", err)
	}

	currentClientRoles, err := prepareRoles(rols, existingRoles)
	if err != nil {
		return fmt.Errorf("error during createOrDeleteRoleScope: %w", err)
	}

	filterClientRolesWithScope := a.filterRoles(existingClientRolesWithScope, true)

	if err := a.client.CreateClientScopesScopeMappingsRealmRoles(
		ctx,
		a.token.AccessToken,
		realmName,
		clientScope.ID,
		a.needCreateRoles(currentClientRoles, filterClientRolesWithScope),
	); err != nil {
		return fmt.Errorf("error during create client scope mapping client roles: %w", err)
	}

	needDeleteRoles := a.needDeleteRoles(currentClientRoles, filterClientRolesWithScope)
	if len(needDeleteRoles) == 0 {
		return nil
	}

	if err := a.client.DeleteClientScopesScopeMappingsRealmRoles(
		ctx,
		a.token.AccessToken,
		realmName,
		clientScope.ID,
		needDeleteRoles,
	); err != nil {
		return fmt.Errorf("error during delete client scope mapping client roles: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) createOrDeleteClientRoleToClient(
	ctx context.Context, realmName string, fromClientID string, rols []RealmRole, opts ScopeMappingOptions,
) error {
	existingRoles, err := a.getExistingClientRoles(ctx, realmName, fromClientID)
	if err != nil {
		return fmt.Errorf("failed to get client roles: %w", err)
	}

	currentClientRoles, err := prepareRoles(rols, existingRoles)
	if err != nil {
		return fmt.Errorf("error during createOrDeleteRoleScope: %w", err)
	}

	toClient, err := a.GetClient(ctx, realmName, opts.Client)
	if err != nil {
		return fmt.Errorf("error during find client: %w", err)
	}

	existingRolesWithClient, err := a.client.GetClientScopeMappingsClientRoles(
		ctx, a.token.AccessToken, realmName, gocloak.PString(toClient.ID), fromClientID,
	)
	if err != nil {
		return fmt.Errorf("failed to get roles by client: %w", err)
	}

	existingRolesWithClientMap := convertRoleNameMap(existingRolesWithClient)

	if err := a.client.CreateClientScopeMappingsClientRoles(ctx, a.token.AccessToken,
		realmName,
		gocloak.PString(toClient.ID),
		fromClientID, a.needCreateRoles(currentClientRoles, existingRolesWithClientMap)); err != nil {
		return fmt.Errorf("error during create client scope mapping client: %w", err)
	}

	if err := a.client.DeleteClientScopeMappingsClientRoles(ctx, a.token.AccessToken,
		realmName,
		gocloak.PString(toClient.ID),
		fromClientID, a.needDeleteRoles(currentClientRoles, existingRolesWithClientMap)); err != nil {
		return fmt.Errorf("error during delete client scope mapping client: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) DeleteClientScopeMappingRealmRoles(ctx context.Context,
	realmName string,
	fromClientID string,
	opts ScopeMappingOptions,
	currRoles []RealmRole,
) error {
	if opts.checkRequired() {
		return fmt.Errorf("failed to delete not set required fields ClientScope or Client")
	}

	if opts.isOnlyClientScope() {
		clientScope, err := a.GetClientScope(ctx, opts.ClientScope, realmName)
		if err != nil && !IsErrNotFound(err) {
			return fmt.Errorf("unable to get client scope: %w", err)
		}

		existingClientRolesWithScope, err := a.GetScopeMappingRealmRoles(ctx, realmName, clientScope.ID)
		if err != nil {
			return fmt.Errorf("failed to get realm roles by scope: %w", err)
		}

		return a.client.DeleteClientScopesScopeMappingsRealmRoles(
			ctx,
			a.token.AccessToken,
			realmName,
			clientScope.ID,
			a.findRoleMath(currRoles, a.filterRoles(existingClientRolesWithScope, true)),
		)
	}

	if opts.isOnlyClient() {
		toClient, err := a.GetClient(ctx, realmName, opts.Client)
		if err != nil {
			return fmt.Errorf("error during find client: %w", err)
		}

		existingRolesWithClient, err := a.client.GetClientScopeMappingsClientRoles(
			ctx, a.token.AccessToken, realmName, gocloak.PString(toClient.ID), fromClientID,
		)
		if err != nil {
			return fmt.Errorf("failed to get roles by client: %w", err)
		}

		existingRolesWithClientMap := convertRoleNameMap(existingRolesWithClient)

		if err := a.client.DeleteClientScopeMappingsClientRoles(ctx, a.token.AccessToken,
			realmName,
			gocloak.PString(toClient.ID),
			fromClientID, a.findRoleMath(currRoles, existingRolesWithClientMap)); err != nil {
			return fmt.Errorf("error during delete client scope mapping client: %w", err)
		}
	}

	return nil
}

// getExistingClientRoles retrieves all client roles and returns them as a map for efficient lookup.
func (a GoCloakAdapter) getExistingClientRoles(
	ctx context.Context,
	realmName,
	clientID string,
) (map[string]gocloak.Role, error) {
	existingRoles, err := a.client.GetClientRoles(ctx, a.token.AccessToken, realmName, clientID, gocloak.GetRoleParams{
		Max: gocloak.IntP(1000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get client roles: %w", err)
	}

	// Create maps for efficient lookup with preallocated memory
	existingRolesMap := convertRoleNameMap(existingRoles)

	return existingRolesMap, nil
}

// convertRoleNameMap creates a map of role names to role for efficient lookup.
func convertRoleNameMap(roles []*gocloak.Role) map[string]gocloak.Role {
	roleMap := make(map[string]gocloak.Role, len(roles))

	for _, role := range convertRolePointersToValues(roles) {
		if role.Name != nil {
			roleMap[*role.Name] = role
		}
	}

	return roleMap
}
