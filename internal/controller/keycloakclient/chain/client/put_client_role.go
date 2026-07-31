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

type PutClientRole struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutClientRole(keycloakApiClient keycloak.Client, k8sClient client.Client) *PutClientRole {
	return &PutClientRole{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutClientRole) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	el.keycloakApiClient = keycloakApiClient
}

func (el *PutClientRole) Serve(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	if err := el.putKeycloakClientRole(ctx, keycloakClient, realmName); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync client roles: %s", err.Error()))

		return fmt.Errorf("unable to put keycloak client role: %w", err)
	}

	el.setSuccessCondition(ctx, keycloakClient, "Client roles synchronized")

	return nil
}

func (el *PutClientRole) setFailureCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionClientRolesSynced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutClientRole) setSuccessCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionClientRolesSynced,
		metav1.ConditionTrue,
		chain.ReasonClientRolesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutClientRole) putKeycloakClientRole(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client role")

	clientDto := ConvertDataClientToClient(&keycloakClient.Client, "", realmName, nil)

	if err := el.keycloakApiClient.SyncClientRoles(ctx, realmName, clientDto); err != nil {
		return fmt.Errorf("unable to sync client roles: %w", err)
	}

	reqLog.Info("End put keycloak client role")

	return nil
}
