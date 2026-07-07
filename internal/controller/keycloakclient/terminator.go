package keycloakclient

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	ctrl "sigs.k8s.io/controller-runtime"
)

type DataTerminator struct {
	ClientIDs, ClientScopeIDs map[string]string
}

type terminator struct {
	realmName                   string
	kClient                     keycloak.Client
	preserveResourcesOnDeletion bool
	dataTerminator              DataTerminator
}

func makeTerminator(data DataTerminator, realmName string, kClient keycloak.Client, preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		realmName:                   realmName,
		kClient:                     kClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
		dataTerminator:              data,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	// delete client scope
	for _, clientScopeID := range t.dataTerminator.ClientScopeIDs {
		if err := t.deleteClientScope(ctx, clientScopeID); err != nil {
			return err
		}
	}

	// delete client
	for _, clientID := range t.dataTerminator.ClientIDs {
		if err := t.deleteClient(ctx, clientID); err != nil {
			return err
		}
	}

	return nil
}

func (t *terminator) deleteClient(ctx context.Context, clientID string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("client_id", clientID)
	log.Info("Start deleting keycloak client")

	if err := t.kClient.DeleteClient(ctx, clientID, t.realmName); err != nil {
		if adapter.IsErrNotFound(err) {
			log.Info("Client not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete keycloak client: %w", err)
	}

	log.Info("Keycloak client has been deleted")

	return nil
}

func (t *terminator) deleteClientScope(ctx context.Context, clientScopeID string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("realm name", t.realmName, "scope id", clientScopeID)
	log.Info("Start deleting client scope")

	if err := t.kClient.DeleteClientScope(ctx, t.realmName, clientScopeID); err != nil {
		if adapter.IsErrNotFound(err) {
			log.Info("Client scope not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete client scope: %w", err)
	}

	log.Info("Client scope has been deleted")

	return nil
}
