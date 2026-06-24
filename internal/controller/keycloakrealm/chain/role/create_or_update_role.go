package role

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type CreateOrUpdateRole struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewCreateOrUpdateRole(kClientV2 *keycloakv2.KeycloakClient) *CreateOrUpdateRole {
	return &CreateOrUpdateRole{kClientV2: kClientV2}
}

func (h *CreateOrUpdateRole) Serve(
	ctx context.Context,
	role *keycloakApi.Role,
	realmName string,
	roleCtx *RoleContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating realm role")

	rolesClient := h.kClientV2.Roles

	existingRole, _, err := rolesClient.GetRealmRole(ctx, realmName, role.Name)
	if err != nil && !keycloakv2.IsNotFound(err) {
		return fmt.Errorf("failed to get realm role: %w", err)
	}

	isComposite := role.Composite
	attrs := role.Attributes
	desc := role.Description

	if existingRole == nil {
		if _, err = rolesClient.CreateRealmRole(ctx, realmName, keycloakv2.RoleRepresentation{
			Name:        &role.Name,
			Description: &desc,
			Composite:   &isComposite,
			Attributes:  &attrs,
		}); err != nil {
			return fmt.Errorf("failed to create realm role: %w", err)
		}

		existingRole, _, err = rolesClient.GetRealmRole(ctx, realmName, role.Name)
		if err != nil {
			return fmt.Errorf("failed to get created realm role: %w", err)
		}
	} else {
		existingRole.Description = &desc
		existingRole.Composite = &isComposite
		existingRole.Attributes = &attrs

		if _, err = rolesClient.UpdateRealmRole(ctx, realmName, role.Name, *existingRole); err != nil {
			return fmt.Errorf("failed to update realm role: %w", err)
		}
	}

	if existingRole.Id != nil {
		roleCtx.RoleID = *existingRole.Id
	}

	log.Info("Realm role has been synced")

	return nil
}
