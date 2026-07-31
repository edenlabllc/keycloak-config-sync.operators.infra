package client

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/dto"
)

type PutRealmRole struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutRealmRole(keycloakApiClient keycloak.Client, k8sClient client.Client) *PutRealmRole {
	return &PutRealmRole{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutRealmRole) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	el.keycloakApiClient = keycloakApiClient
}

func (el *PutRealmRole) Serve(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	if err := el.putRealmRoles(ctx, keycloakClient, realmName); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync realm roles: %s", err.Error()))

		return fmt.Errorf("unable to put realm roles: %w", err)
	}

	el.setSuccessCondition(ctx, keycloakClient, "Realm roles synchronized")

	return nil
}

func (el *PutRealmRole) setFailureCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionRealmRolesSynced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutRealmRole) setSuccessCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionRealmRolesSynced,
		metav1.ConditionTrue,
		chain.ReasonRealmRolesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutRealmRole) putRealmRoles(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put realm roles")

	if keycloakClient.Client.RealmRoles == nil || len(*keycloakClient.Client.RealmRoles) == 0 {
		reqLog.Info("Keycloak client does not have realm roles")
		return nil
	}

	for _, role := range *keycloakClient.Client.RealmRoles {
		roleDto := &dto.IncludedRealmRole{
			Name:      role.Name,
			Composite: role.Composite,
		}

		exist, err := el.keycloakApiClient.ExistRealmRole(realmName, roleDto.Name)
		if err != nil {
			return fmt.Errorf("error during ExistRealmRole: %w", err)
		}

		if exist {
			reqLog.Info("Client already exists")
			return nil
		}

		err = el.keycloakApiClient.CreateIncludedRealmRole(realmName, roleDto)
		if err != nil {
			return fmt.Errorf("error during CreateRealmRole: %w", err)
		}
	}

	reqLog.Info("End put realm roles")

	return nil
}
