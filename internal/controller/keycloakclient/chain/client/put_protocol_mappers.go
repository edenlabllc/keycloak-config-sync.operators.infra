package client

import (
	"context"
	"fmt"
	"maps"

	"github.com/Nerzal/gocloak/v12"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
)

type PutProtocolMappers struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutProtocolMappers(keycloakApiClient keycloak.Client, k8sClient client.Client) *PutProtocolMappers {
	return &PutProtocolMappers{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutProtocolMappers) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	el.keycloakApiClient = keycloakApiClient
}

func (el *PutProtocolMappers) Serve(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	if err := el.putProtocolMappers(keycloakClient, realmName); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync protocol mappers: %s", err.Error()))

		return fmt.Errorf("unable to put protocol mappers: %w", err)
	}

	el.setSuccessCondition(ctx, keycloakClient, "Protocol mappers synchronized")

	return nil
}

func (el *PutProtocolMappers) setFailureCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionProtocolMappersSynced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutProtocolMappers) setSuccessCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionProtocolMappersSynced,
		metav1.ConditionTrue,
		chain.ReasonProtocolMappersSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutProtocolMappers) putProtocolMappers(keycloakClient *DataClient, realmName string) error {
	var protocolMappers []gocloak.ProtocolMapperRepresentation

	if keycloakClient.Client.ProtocolMappers != nil {
		protocolMappers = make([]gocloak.ProtocolMapperRepresentation, 0,
			len(*keycloakClient.Client.ProtocolMappers))

		for _, mapper := range *keycloakClient.Client.ProtocolMappers {
			configCopy := make(map[string]string, len(mapper.Config))
			maps.Copy(configCopy, mapper.Config)

			protocolMappers = append(protocolMappers, gocloak.ProtocolMapperRepresentation{
				Name:           gocloak.StringP(mapper.Name),
				Protocol:       gocloak.StringP(mapper.Protocol),
				ProtocolMapper: gocloak.StringP(mapper.ProtocolMapper),
				Config:         &configCopy,
			})
		}
	}

	if err := el.keycloakApiClient.SyncClientProtocolMapper(
		ConvertDataClientToClient(&keycloakClient.Client, "", realmName, nil),
		protocolMappers, keycloakClient.Client.GetReconciliationStrategy() == keycloakApi.ReconciliationStrategyAddOnly,
	); err != nil {
		return fmt.Errorf("unable to sync protocol mapper: %w", err)
	}

	return nil
}
