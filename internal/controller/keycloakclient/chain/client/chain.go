package client

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/secretref"
)

type ReasonScope struct {
	ClientID     string
	FailureCount int64
	Generation   int64
}

type DataClient struct {
	CRD         *keycloakApi.KeycloakClient
	Client      keycloakApi.Client
	ClientIndex int
	Namespace   string
	ReasonScope *ReasonScope
}

type ControllerHelper interface {
	CreateKeycloakClientFromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (keycloak.Client, error)
}

type ClientHandler interface {
	Serve(
		ctx context.Context,
		keycloakClient *DataClient,
		realmName string,
	) error
	WithKeycloakApiClient(keycloakApiClient keycloak.Client)
}

type Chain struct {
	helper   ControllerHelper
	handlers []ClientHandler
}

func (ch *Chain) Use(handlers ...ClientHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	keycloakClientSettings *keycloakApi.KeycloakClient,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakClient chain")

	for index, cl := range keycloakClientSettings.Spec.Client {
		name := helper.RemoveSpecialChar(cl.Name)

		dataClient := &DataClient{
			CRD:         keycloakClientSettings,
			ClientIndex: index,
			Client:      cl,
			Namespace:   keycloakClientSettings.Namespace,
			ReasonScope: &ReasonScope{
				ClientID:   keycloakClientSettings.Status.GetClientIDByName(name),
				Generation: keycloakClientSettings.Generation,
			},
		}

		for i := 0; i < len(ch.handlers); i++ {
			h := ch.handlers[i]

			err := h.Serve(ctx, dataClient, realmName)
			// Refresh Token
			if helper.IsUnauthorizedError(err) {
				log.Info("KeycloakClientScope chain refreshToken")
				cl, errCl := ch.refreshToken(ctx, keycloakClientSettings)
				if errCl != nil {
					log.Error(errCl, "KeycloakClientScope chain refreshToken error")

					return errCl
				}

				h.WithKeycloakApiClient(cl)

				err = h.Serve(ctx, dataClient, realmName)
			}

			if err != nil {
				log.Info("KeycloakClient chain finished with error")

				return fmt.Errorf("failed to serve handler: %w", err)
			}
		}

		keycloakClientSettings.Status.PutClientIDByName(name,
			dataClient.ReasonScope.ClientID)
	}

	log.Info("Handling of KeycloakClient has been finished")

	return nil
}

func (ch *Chain) refreshToken(
	ctx context.Context,
	keycloakClientSettings *keycloakApi.KeycloakClient) (keycloak.Client, error) {
	return ch.helper.CreateKeycloakClientFromConfigRef(ctx, keycloakClientSettings)
}

func MakeChain(
	helper ControllerHelper,
	keycloakApiClient keycloak.Client,
	k8sClient client.Client,
) *Chain {
	c := &Chain{
		helper: helper,
	}

	c.Use(
		NewPutClient(keycloakApiClient, k8sClient, secretref.NewSecretRef(k8sClient)),
		NewPutClientRole(keycloakApiClient, k8sClient),
		NewPutRealmRole(keycloakApiClient, k8sClient),
		NewPutClientScope(keycloakApiClient, k8sClient),
		NewPutProtocolMappers(keycloakApiClient, k8sClient),
		NewServiceAccount(keycloakApiClient, k8sClient),
		NewProcessScope(keycloakApiClient, k8sClient),
		NewProcessResources(keycloakApiClient, k8sClient),
		NewProcessPolicy(keycloakApiClient, k8sClient),
		NewProcessPermissions(keycloakApiClient, k8sClient),
		NewPutAdminFineGrainedPermissions(keycloakApiClient, k8sClient),
	)

	return c
}
