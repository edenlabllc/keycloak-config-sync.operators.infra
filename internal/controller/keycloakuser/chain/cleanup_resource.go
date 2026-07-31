package chain

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
)

type CleanupResource struct {
	k8sClient client.Client
}

func NewCleanupResource(k8sClient client.Client) *CleanupResource {
	return &CleanupResource{k8sClient: k8sClient}
}

func (h *CleanupResource) WithKeycloakApiClient(_ *keycloakv2.KeycloakClient) {
}

func (h *CleanupResource) Serve(
	ctx context.Context,
	user *keycloakApiAlpha.KeycloakUser,
	_ string,
	_ *UserContext,
) error {
	if user.Spec.KeepResource {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting KeycloakRealmUser resource as KeepResource is false")

	if err := h.k8sClient.Delete(ctx, user); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("unable to delete instance of keycloak realm user: %w", err)
	}

	return nil
}
