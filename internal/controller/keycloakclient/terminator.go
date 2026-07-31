package keycloakclient

import (
	"context"
	"fmt"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
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
	helper                      Helper
	keycloakClientSettings      *keycloakApi.KeycloakClient
}

func makeTerminator(
	helper Helper,
	data DataTerminator,
	realmName string,
	kClient keycloak.Client,
	keycloakClientSettings *keycloakApi.KeycloakClient,
	preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		helper:                      helper,
		realmName:                   realmName,
		kClient:                     kClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
		dataTerminator:              data,
		keycloakClientSettings:      keycloakClientSettings,
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
		if helper.IsUnauthorizedError(err) {
			log.Info("deleteClient refreshToken")
			cl, errCl := t.refreshToken(ctx, t.keycloakClientSettings)
			if errCl != nil {
				log.Error(errCl, "deleteClient refreshToken error")

				return errCl
			}

			t.kClient = cl

			return t.deleteClient(ctx, clientID)
		}

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
		if helper.IsUnauthorizedError(err) {
			log.Info("deleteClientScope refreshToken")
			cl, errCl := t.refreshToken(ctx, t.keycloakClientSettings)
			if errCl != nil {
				log.Error(errCl, "deleteClientScope refreshToken error")

				return errCl
			}

			t.kClient = cl

			return t.deleteClientScope(ctx, clientScopeID)
		}

		if adapter.IsErrNotFound(err) {
			log.Info("Client scope not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete client scope: %w", err)
	}

	log.Info("Client scope has been deleted")

	return nil
}

func (t *terminator) refreshToken(
	ctx context.Context,
	keycloakClientSettings *keycloakApi.KeycloakClient) (keycloak.Client, error) {
	return t.helper.CreateKeycloakClientFromConfigRef(ctx, keycloakClientSettings)
}
