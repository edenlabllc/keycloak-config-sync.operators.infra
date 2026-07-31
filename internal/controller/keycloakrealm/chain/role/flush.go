package role

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/objectmeta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type Flush struct {
	kClientV2 *keycloakv2.KeycloakClient
	k8sClient client.Client
}

func NewFlush(
	kClientV2 *keycloakv2.KeycloakClient,
	k8sClient client.Client,
) *Flush {
	return &Flush{
		kClientV2: kClientV2,
		k8sClient: k8sClient,
	}
}

func (h *Flush) WithKeycloakApiClient(kClientV2 *keycloakv2.KeycloakClient) {
	h.kClientV2 = kClientV2
}

func (h *Flush) Flush(ctx context.Context, realm *keycloakApiAlpha.KeycloakRealm, rIDs map[string]string) error {
	if !h.hasReconciliationStrategyFull(realm) {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	rolesClient := h.kClientV2.Roles
	realmName := realm.Spec.RealmName

	storageIDs := realm.Status.RoleIDs
	if storageIDs == nil {
		storageIDs = make(map[string]string)
	}

	deleteRolesNames := make([]string, 0)

	log.Info("Start flush realm role")

	for sName := range storageIDs {
		if _, ok := rIDs[sName]; !ok {
			deleteRolesNames = append(deleteRolesNames, sName)

			delete(storageIDs, sName)
		}
	}

	for _, delRoleName := range deleteRolesNames {
		if _, err := rolesClient.DeleteRealmRole(ctx, realmName, delRoleName); err != nil {
			if keycloakv2.IsNotFound(err) {
				log.Info("Flush realm role not found, skipping",
					"realm name", realmName, "role", delRoleName)

				continue
			} else {
				return fmt.Errorf("flush failed to delete realm [%s] role %s: %w",
					realmName, delRoleName, err)
			}
		}
	}

	for cRoleName, cID := range rIDs {
		storageIDs[cRoleName] = cID
	}

	realm.Status.RoleIDs = storageIDs

	if err := h.updateStatus(ctx, realm); err != nil {
		return err
	}

	log.Info("Flush realm role deleted successfully")

	return nil
}

func (h *Flush) updateStatus(ctx context.Context, instance *keycloakApiAlpha.KeycloakRealm) error {
	if err := h.k8sClient.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("unable to update status of role: %w", err)
	}

	return nil
}

func (h *Flush) hasReconciliationStrategyFull(instance *keycloakApiAlpha.KeycloakRealm) bool {
	return !objectmeta.PreserveResourcesOnDeletion(instance) &&
		(instance.Spec.ReconciliationStrategy == keycloakApiAlpha.ReconciliationStrategyFull ||
			instance.Spec.ReconciliationStrategy == "")
}
