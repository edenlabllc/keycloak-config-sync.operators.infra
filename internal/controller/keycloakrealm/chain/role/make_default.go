package role

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type MakeDefault struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewMakeDefault(kClientV2 *keycloakv2.KeycloakClient) *MakeDefault {
	return &MakeDefault{kClientV2: kClientV2}
}

func (h *MakeDefault) WithKeycloakApiClient(kClientV2 *keycloakv2.KeycloakClient) {
	h.kClientV2 = kClientV2
}

func (h *MakeDefault) Serve(
	ctx context.Context,
	role *keycloakApi.Role,
	realmName string,
	roleCtx *RoleContext,
) error {
	if !role.IsDefault {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Making role default")

	name := role.Name
	defaultRoleName := "default-roles-" + realmName

	if _, err := h.kClientV2.Roles.AddRealmRoleComposites(ctx, realmName, defaultRoleName, []keycloakv2.RoleRepresentation{
		{
			Id:   &roleCtx.RoleID,
			Name: &name,
		},
	}); err != nil {
		return fmt.Errorf("failed to add role to default-roles: %w", err)
	}

	log.Info("Role has been made default")

	return nil
}
