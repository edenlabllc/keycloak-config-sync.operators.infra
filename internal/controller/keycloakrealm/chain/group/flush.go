package group

import (
	"context"
	"fmt"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/objectmeta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Flush struct{}

func NewFlush() *Flush {
	return &Flush{}
}

func (h *Flush) Flush(
	ctx context.Context,
	kClientV2 *keycloakv2.KeycloakClient,
	k8sClient client.Client,
	realm *keycloakApiAlpha.KeycloakRealm,
	groupIDs []string) error {
	if !h.hasReconciliationStrategyFull(realm) {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	groupClient := kClientV2.Groups
	realmName := realm.Spec.RealmName
	storageGroupIDs := realm.Status.GroupIDs
	deleteGroupIDs := make([]string, 0)

	log.Info("Start flush realm group")

	for sIndex, sGroupID := range storageGroupIDs {
		if !hasExistingGrouID(sGroupID, groupIDs) {
			deleteGroupIDs = append(deleteGroupIDs, sGroupID)

			storageGroupIDs = helper.RemoveSliceIndex(storageGroupIDs, sIndex)
		}
	}

	for _, delGroupID := range deleteGroupIDs {
		if _, err := groupClient.DeleteGroup(ctx, realmName, delGroupID); err != nil {
			if keycloakv2.IsNotFound(err) {
				log.Info("Flush group not found, skipping deletion",
					"realm name", realmName, "group", delGroupID)

				return nil
			}

			return fmt.Errorf("flush unable to delete realmName: %s groupID: %q: %w",
				realmName, delGroupID, err)
		}
	}

	storageGroupIDs = append(storageGroupIDs, groupIDs...)

	realm.Status.GroupIDs = helper.RemoveDuplicates(storageGroupIDs)

	if err := h.updateStatus(ctx, k8sClient, realm); err != nil {
		return err
	}

	log.Info("Flush realm role deleted successfully")

	return nil
}

func (h *Flush) updateStatus(ctx context.Context, k8sClient client.Client, instance *keycloakApiAlpha.KeycloakRealm) error {
	if err := k8sClient.Status().Update(ctx, instance); err != nil {
		return fmt.Errorf("unable to update status of group: %w", err)
	}

	return nil
}

func (h *Flush) hasReconciliationStrategyFull(instance *keycloakApiAlpha.KeycloakRealm) bool {
	return !objectmeta.PreserveResourcesOnDeletion(instance) &&
		(instance.Spec.ReconciliationStrategy == keycloakApiAlpha.ReconciliationStrategyFull ||
			instance.Spec.ReconciliationStrategy == "")
}

func hasExistingGrouID(gID string, groups []string) bool {
	for _, groupID := range groups {
		if gID == groupID {
			return true
		}
	}

	return false
}
