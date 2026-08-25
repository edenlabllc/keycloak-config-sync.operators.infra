package policy

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
)

// DataPolicy carries the client registration policy configuration through the chain.
// Unlike the client and scope chains the handlers are served once for the whole list,
// because each entry addresses a single Keycloak policy component on its own.
type DataPolicy struct {
	CRD        *keycloakApi.KeycloakClient
	Policies   []keycloakApi.ClientAllowedPolicy
	Generation int64
}

type PolicyHandler interface {
	Serve(
		ctx context.Context,
		policy *DataPolicy,
		realmName string,
	) error
	WithKeycloakApiClient(keycloakApiClient keycloak.Client)
}

type ControllerHelper interface {
	CreateKeycloakClientFromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (keycloak.Client, error)
}

type Chain struct {
	helper   ControllerHelper
	handlers []PolicyHandler
}

func (ch *Chain) Use(handlers ...PolicyHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	keycloakClientSettings *keycloakApi.KeycloakClient,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakClientRegistrationPolicy chain")

	if keycloakClientSettings.Spec.ClientRegistrationPolicy == nil {
		return nil
	}

	dataPolicy := &DataPolicy{
		CRD:        keycloakClientSettings,
		Policies:   *keycloakClientSettings.Spec.ClientRegistrationPolicy,
		Generation: keycloakClientSettings.Generation,
	}

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, dataPolicy, realmName)
		// Refresh Token
		if helper.IsUnauthorizedError(err) {
			log.Info("KeycloakClientRegistrationPolicy chain refreshToken")

			cl, errCl := ch.refreshToken(ctx, keycloakClientSettings)
			if errCl != nil {
				log.Error(errCl, "KeycloakClientRegistrationPolicy chain refreshToken error")

				return errCl
			}

			h.WithKeycloakApiClient(cl)

			err = h.Serve(ctx, dataPolicy, realmName)
		}

		if err != nil {
			log.Info("KeycloakClientRegistrationPolicy chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakClientRegistrationPolicy has been finished")

	return nil
}

func (ch *Chain) refreshToken(
	ctx context.Context,
	keycloakClientSettings *keycloakApi.KeycloakClient) (keycloak.Client, error) {
	return ch.helper.CreateKeycloakClientFromConfigRef(ctx, keycloakClientSettings)
}

func MakeChain(
	controllerHelper ControllerHelper,
	keycloakApiClient keycloak.Client,
	k8sClient client.Client,
) *Chain {
	c := &Chain{
		helper: controllerHelper,
	}

	c.Use(
		NewPutClientRegistrationPolicy(keycloakApiClient, k8sClient),
	)

	return c
}
