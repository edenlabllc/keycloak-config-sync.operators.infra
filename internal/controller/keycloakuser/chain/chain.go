package chain

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

// UserContext holds data that is passed between chain handlers.
type UserContext struct {
	// UserID is the Keycloak user ID, set by CreateOrUpdateUser handler.
	UserID string
}

type RealmUserHandler interface {
	Serve(
		ctx context.Context,
		user *keycloakApiAlpha.KeycloakUser,
		realmName string,
		userCtx *UserContext,
	) error
	WithKeycloakApiClient(kClientV2 *keycloakv2.KeycloakClient)
}

type ControllerHelper interface {
	CreateKeycloakClientV2FromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (*keycloakv2.KeycloakClient, error)
}

type Chain struct {
	helper   ControllerHelper
	handlers []RealmUserHandler
}

func (ch *Chain) Use(handlers ...RealmUserHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	user *keycloakApiAlpha.KeycloakUser,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakRealmUser chain")

	userCtx := &UserContext{}

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, user, realmName, userCtx)
		// Refresh Token
		if helper.IsUnauthorizedError(err) {
			log.Info("KeycloakRealmUser chain refreshToken")
			cl, errCl := ch.refreshToken(ctx, user)
			if errCl != nil {
				log.Error(errCl, "KeycloakRealmUser chain refreshToken error")

				return errCl
			}

			h.WithKeycloakApiClient(cl)

			err = h.Serve(ctx, user, realmName, userCtx)
		}

		if err != nil {
			log.Info("KeycloakRealmUser chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakRealmUser has been finished")

	return nil
}

func (ch *Chain) refreshToken(
	ctx context.Context,
	user *keycloakApiAlpha.KeycloakUser) (*keycloakv2.KeycloakClient, error) {
	return ch.helper.CreateKeycloakClientV2FromConfigRef(ctx, user)
}

func MakeChain(
	helper ControllerHelper,
	k8sClient client.Client,
	kClientV2 *keycloakv2.KeycloakClient,
) *Chain {
	ch := &Chain{
		helper: helper,
	}

	ch.Use(
		NewCreateOrUpdateUser(k8sClient, kClientV2),
		NewSetUserPassword(k8sClient, kClientV2),
		NewSyncUserRoles(kClientV2),
		NewSyncUserGroups(kClientV2),
		NewSyncUserIdentityProviders(kClientV2),
		NewCleanupResource(k8sClient),
	)

	return ch
}
