package client

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
)

type PutClientScope struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutClientScope(keycloakApiClient keycloak.Client, k8sClient client.Client) *PutClientScope {
	return &PutClientScope{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutClientScope) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	el.keycloakApiClient = keycloakApiClient
}

func (el *PutClientScope) Serve(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	if err := el.putClientScope(ctx, keycloakClient, realmName); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync client scopes: %s", err.Error()))

		return fmt.Errorf("error during putClientScope: %w", err)
	}

	el.setSuccessCondition(ctx, keycloakClient, "Client scopes synchronized")

	return nil
}

func (el *PutClientScope) setFailureCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionClientScopesSynced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutClientScope) setSuccessCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionClientScopesSynced,
		metav1.ConditionTrue,
		chain.ReasonClientScopesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutClientScope) putClientScope(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	if err := el.putDefaultClientScope(ctx, keycloakClient, realmName); err != nil {
		return err
	}

	if err := el.putOptionalClientScope(ctx, keycloakClient, realmName); err != nil {
		return err
	}

	return nil
}

func (el *PutClientScope) putDefaultClientScope(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	kCloakSpec := keycloakClient.Client

	if len(kCloakSpec.DefaultClientScopes) == 0 {
		return nil
	}

	defaultScopes, err := el.keycloakApiClient.GetClientScopesByNames(ctx, realmName, kCloakSpec.DefaultClientScopes)
	if err != nil {
		return fmt.Errorf("error during GetClientScope: %w", err)
	}

	err = el.keycloakApiClient.AddDefaultScopeToClient(ctx, realmName, kCloakSpec.ClientId, defaultScopes)
	if err != nil {
		return fmt.Errorf("failed to add default scope to client %s: %w", keycloakClient.Client.Name, err)
	}

	return nil
}

func (el *PutClientScope) putOptionalClientScope(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	kCloakSpec := keycloakClient.Client

	if len(kCloakSpec.OptionalClientScopes) == 0 {
		return nil
	}

	optionalScopes, err := el.keycloakApiClient.GetClientScopesByNames(ctx, realmName, kCloakSpec.OptionalClientScopes)
	if err != nil {
		return fmt.Errorf("error during GetClientScope: %w", err)
	}

	err = el.keycloakApiClient.AddOptionalScopeToClient(ctx, realmName, kCloakSpec.ClientId, optionalScopes)
	if err != nil {
		return fmt.Errorf("failed to add default scope to client %s: %w", keycloakClient.Client.Name, err)
	}

	return nil
}
