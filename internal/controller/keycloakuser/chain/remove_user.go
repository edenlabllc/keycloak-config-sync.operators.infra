package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/objectmeta"
)

type RemoveUser struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewRemoveUser(kClientV2 *keycloakv2.KeycloakClient) *RemoveUser {
	return &RemoveUser{kClientV2: kClientV2}
}

func (h *RemoveUser) WithKeycloakApiClient(kClientV2 *keycloakv2.KeycloakClient) {
	h.kClientV2 = kClientV2
}

func (h *RemoveUser) ServeRequest(ctx context.Context, user *keycloakApiAlpha.KeycloakUser, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start removing user")

	if objectmeta.PreserveResourcesOnDeletion(user) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	keycloakUser, _, err := h.kClientV2.Users.FindUserByUsername(ctx, realmName, user.Spec.Username)
	if err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("User not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to find user %s: %w", user.Spec.Username, err)
	}

	if _, err := h.kClientV2.Users.DeleteUser(ctx, realmName, *keycloakUser.Id); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("User not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete user %s: %w", user.Spec.Username, err)
	}

	log.Info("User deleted successfully")

	return nil
}
