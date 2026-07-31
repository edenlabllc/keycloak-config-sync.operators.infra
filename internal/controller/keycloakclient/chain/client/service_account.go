package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
)

type ServiceAccount struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewServiceAccount(keycloakApiClient keycloak.Client, k8sClient client.Client) *ServiceAccount {
	return &ServiceAccount{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *ServiceAccount) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	el.keycloakApiClient = keycloakApiClient
}

func (el *ServiceAccount) Serve(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	if keycloakClient.Client.ServiceAccount == nil || !keycloakClient.Client.ServiceAccount.Enabled {
		return nil
	}

	if keycloakClient.Client.ServiceAccount != nil && keycloakClient.Client.Public {
		return errors.New("service account can not be configured with public client")
	}

	clientRoles := make(map[string][]string)
	for _, v := range keycloakClient.Client.ServiceAccount.ClientRoles {
		clientRoles[v.ClientID] = v.Roles
	}

	addOnly := keycloakClient.Client.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly

	if err := el.keycloakApiClient.SyncServiceAccountRoles(realmName,
		keycloakClient.ReasonScope.ClientID, keycloakClient.Client.ServiceAccount.RealmRoles, clientRoles, addOnly); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))

		return fmt.Errorf("unable to sync service account roles: %w", err)
	}

	if keycloakClient.Client.ServiceAccount.Groups != nil {
		if err := el.keycloakApiClient.SyncServiceAccountGroups(realmName,
			keycloakClient.ReasonScope.ClientID, keycloakClient.Client.ServiceAccount.Groups, addOnly); err != nil {
			el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))

			return fmt.Errorf("unable to sync service account groups: %w", err)
		}
	}

	if keycloakClient.Client.ServiceAccount.AttributesV2 != nil {
		if err := el.keycloakApiClient.SetServiceAccountAttributes(realmName, keycloakClient.ReasonScope.ClientID,
			keycloakClient.Client.ServiceAccount.AttributesV2, addOnly); err != nil {
			el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync service account: %s", err.Error()))

			return fmt.Errorf("unable to set service account attributes: %w", err)
		}
	}

	el.setSuccessCondition(ctx, keycloakClient, "Service account synchronized")

	return nil
}

func (el *ServiceAccount) setFailureCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionServiceAccountSynced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *ServiceAccount) setSuccessCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionServiceAccountSynced,
		metav1.ConditionTrue,
		chain.ReasonServiceAccountSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}
