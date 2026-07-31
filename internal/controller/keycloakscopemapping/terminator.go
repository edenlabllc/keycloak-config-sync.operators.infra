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
	clientScope, client         string
	roles                       []adapter.RealmRole
	kClient                     keycloak.Client
	preserveResourcesOnDeletion bool
}

func makeTerminator(kClient keycloak.Client,
	realmName, scopeID string,
	clientScope, client string,
	roles []adapter.RealmRole,
	preserveResourcesOnDeletion bool,
) *terminator {
	return &terminator{
		kClient:                     kClient,
		realmName:                   realmName,
		scopeID:                     scopeID,
		clientScope:                 clientScope,
		client:                      client,
		roles:                       roles,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues(
		"realm name", t.realmName,
		"scope id", t.scopeID,
		"client scope", t.clientScope,
		"client", t.client,
	)

	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting scope mapping")

	if err := t.delete(ctx); err != nil {
		if adapter.IsErrNotFound(err) {
			log.Info("Scope mapping not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete scope mapping: %w", err)
	}

	log.Info("Scope mapping has been deleted")

	return nil
}

func (t *terminator) delete(ctx context.Context) error {
	if t.scopeID == "" {
		return nil
	}

	if (t.clientScope != "" || t.client != "") && t.scopeID != "" {
		return t.kClient.DeleteClientScopeMappingRealmRoles(ctx, t.realmName, t.scopeID, adapter.ScopeMappingOptions{
			ClientScope: t.clientScope,
			Client:      t.client,
		}, t.roles)
	}

	return t.kClient.DeleteScopeMappingRoles(ctx, t.realmName, t.scopeID, t.roles)
}
