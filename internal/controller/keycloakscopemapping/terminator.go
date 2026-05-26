package keycloakscopemapping

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
)

type terminator struct {
	realmName, scopeID          string
	roles                       []adapter.RealmRole
	kClient                     keycloak.Client
	preserveResourcesOnDeletion bool
}

func makeTerminator(kClient keycloak.Client,
	realmName, scopeID string,
	roles []adapter.RealmRole,
	preserveResourcesOnDeletion bool,
) *terminator {
	return &terminator{
		kClient:                     kClient,
		realmName:                   realmName,
		scopeID:                     scopeID,
		roles:                       roles,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("realm name", t.realmName, "scope id", t.scopeID)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting scope mapping")

	if err := t.kClient.DeleteScopeMappingRoles(ctx, t.realmName, t.scopeID, t.roles); err != nil {
		if adapter.IsErrNotFound(err) {
			log.Info("Scope mapping not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete scope mapping: %w", err)
	}

	log.Info("Scope mapping has been deleted")

	return nil
}
