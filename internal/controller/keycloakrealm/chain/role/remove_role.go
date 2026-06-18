package role

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type RemoveRole struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewRemoveRole(
	kClientV2 *keycloakv2.KeycloakClient,
) *RemoveRole {
	return &RemoveRole{
		kClientV2: kClientV2,
	}
}

func (h *RemoveRole) ServeRequest(ctx context.Context, role *keycloakApi.Role, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start removing realm role")

	if _, err := h.kClientV2.Roles.DeleteRealmRole(ctx, realmName, role.Name); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Realm role not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete realm role %s: %w", role.Name, err)
	}

	log.Info("Realm role deleted successfully")

	return nil
}
