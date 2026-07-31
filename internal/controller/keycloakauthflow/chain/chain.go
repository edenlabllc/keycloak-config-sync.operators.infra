package chain

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

// AuthFlowHandler is a single step in the KeycloakAuthFlow reconciliation chain.
type AuthFlowHandler interface {
	Serve(ctx context.Context, flow *keycloakApiAlpha.KeycloakAuthFlow, realmName string) error
	WithKeycloakApiClient(kClientV2 *keycloakv2.KeycloakClient)
}

// Chain sequentially executes a list of AuthFlowHandlers.
type Chain struct {
	helper   ControllerHelper
	handlers []AuthFlowHandler
}

type ControllerHelper interface {
	CreateKeycloakClientV2FromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (*keycloakv2.KeycloakClient, error)
}

func (ch *Chain) Use(handlers ...AuthFlowHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(ctx context.Context, flow *keycloakApiAlpha.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakAuthFlow chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, flow, realmName)
		// Refresh Token
		if helper.IsUnauthorizedError(err) {
			log.Info("KeycloakAuthFlow chain refreshToken")
			cl, errCl := ch.refreshToken(ctx, flow)
			if errCl != nil {
				log.Error(errCl, "KeycloakAuthFlow chain refreshToken error")

				return errCl
			}

			h.WithKeycloakApiClient(cl)

			err = h.Serve(ctx, flow, realmName)
		}

		if err != nil {
			log.Info("KeycloakAuthFlow chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakAuthFlow has been finished")

	return nil
}

func (ch *Chain) refreshToken(
	ctx context.Context,
	flow *keycloakApiAlpha.KeycloakAuthFlow) (*keycloakv2.KeycloakClient, error) {
	return ch.helper.CreateKeycloakClientV2FromConfigRef(ctx, flow)
}

// MakeChain creates the default reconciliation chain for KeycloakAuthFlow.
func MakeChain(helper ControllerHelper, kClient *keycloakv2.KeycloakClient) *Chain {
	ch := &Chain{
		helper: helper,
	}

	ch.Use(
		NewCreateOrUpdateAuthFlow(kClient),
		NewSyncAuthFlowExecutions(kClient),
	)

	return ch
}
