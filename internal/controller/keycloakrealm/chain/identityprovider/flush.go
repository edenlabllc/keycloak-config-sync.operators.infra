package identityprovider

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/objectmeta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
)

type Flush struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewFlush(
	keycloakApiClient keycloak.Client,
	k8sClient client.Client,
) *Flush {
	return &Flush{
		keycloakApiClient: keycloakApiClient,
		k8sClient:         k8sClient,
	}
}

func (h *Flush) Flush(ctx context.Context, realm *keycloakApiAlpha.KeycloakRealm, currAlias []string) error {
	if !h.hasReconciliationStrategyFull(realm) {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	realmName := realm.Spec.RealmName
	storageAlias := realm.Status.AliasIDPs
	deleteAlias := make([]string, 0)

	log.Info("Start flush realm identityProvider")

	for sIndex, aliasIDP := range storageAlias {
		if !hasExistingAlias(aliasIDP, currAlias) {
			deleteAlias = append(deleteAlias, aliasIDP)

			storageAlias = helper.RemoveSliceIndex(storageAlias, sIndex)
		}
	}

	for _, delAlias := range deleteAlias {
		if err := h.keycloakApiClient.DeleteIdentityProvider(ctx, realmName, delAlias); err != nil {
			if adapter.IsErrNotFound(err) {
				log.Info("Flush realm idp not found, skipping deletion.",
					"realm", realmName, "idp alias", delAlias)

				return nil
			}

			return fmt.Errorf("flush unable to delete realm %s idp %q: %w",
				realmName, delAlias, err)
		}
	}

	storageAlias = append(storageAlias, currAlias...)

	realm.Status.AliasIDPs = helper.RemoveDuplicates(storageAlias)

	if err := h.updateStatus(ctx, realm); err != nil {
		return err
	}

	log.Info("Flush realm identityProvider deleted successfully")

	return nil
}

func (h *Flush) updateStatus(ctx context.Context, instance *keycloakApiAlpha.KeycloakRealm) error {
	if err := h.k8sClient.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("unable to update status of idp: %w", err)
	}

	return nil
}

func (h *Flush) hasReconciliationStrategyFull(instance *keycloakApiAlpha.KeycloakRealm) bool {
	return !objectmeta.PreserveResourcesOnDeletion(instance) &&
		(instance.Spec.ReconciliationStrategy == keycloakApiAlpha.ReconciliationStrategyFull ||
			instance.Spec.ReconciliationStrategy == "")
}

func hasExistingAlias(alias string, listAlias []string) bool {
	for _, cAlias := range listAlias {
		if cAlias == alias {
			return true
		}
	}

	return false
}
