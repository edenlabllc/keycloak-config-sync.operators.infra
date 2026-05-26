package keycloakclientscopemapping

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
)

type terminator struct {
	realmName, clientID         string
	clientScope, client         string
	roles                       []adapter.RealmRole
	kClient                     keycloak.Client
	preserveResourcesOnDeletion bool
}

func makeTerminator(kClient keycloak.Client,
	realmName, clientID string,
	clientScope, client string,
	roles []adapter.RealmRole,
	preserveResourcesOnDeletion bool,
) *terminator {
	return &terminator{
		kClient:                     kClient,
		realmName:                   realmName,
		clientID:                    clientID,
		clientScope:                 clientScope,
		client:                      client,
		roles:                       roles,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues(
		"realm name", t.realmName, "client id", t.clientID,
		"client scope", t.clientScope, "client", t.client,
	)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting client scope")

	if err := t.kClient.DeleteClientScopeMappingRealmRoles(ctx, t.realmName, t.clientID, adapter.ScopeMappingOptions{
		ClientScope: t.clientScope,
		Client:      t.client,
	}, t.roles); err != nil {
		if adapter.IsErrNotFound(err) {
			log.Info("Client scope not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete client roles: %w", err)
	}

	log.Info("Client scope has been deleted")

	return nil
}
