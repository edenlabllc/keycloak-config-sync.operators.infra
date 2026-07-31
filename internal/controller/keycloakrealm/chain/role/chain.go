package role

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type FlushFunc func(
	ctx context.Context,
	realm *keycloakApi.KeycloakRealm,
	rIDs map[string]string) error

// RoleContext holds data that is passed between chain handlers.
type RoleContext struct {
	// RoleID is the Keycloak role ID, set by CreateOrUpdateRole handler.
	RoleID string
}

type RealmRoleHandler interface {
	Serve(
		ctx context.Context,
		role *keycloakApi.Role,
		realmName string,
		roleCtx *RoleContext,
	) error
	WithKeycloakApiClient(kClientV2 *keycloakv2.KeycloakClient)
}

type ControllerHelper interface {
	CreateKeycloakClientFromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (keycloak.Client, error)
	CreateKeycloakClientV2FromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (*keycloakv2.KeycloakClient, error)
}

type Chain struct {
	helper   ControllerHelper
	handlers []RealmRoleHandler
	flush    FlushFunc
}

func (ch *Chain) Use(handlers ...RealmRoleHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	realm *keycloakApi.KeycloakRealm,
) error {
	rIDs := make(map[string]string, len(realm.Spec.Roles))
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting Keycloak Realm Role chain")

	for _, role := range realm.Spec.Roles {
		roleCtx := &RoleContext{}

		if err := ch.run(ctx, realm, &role, realm.Spec.RealmName, roleCtx); err != nil {
			return err
		}

		rIDs[role.Name] = roleCtx.RoleID
	}

	if err := ch.flush(ctx, realm, rIDs); err != nil {
		return err
	}

	log.Info("Handling of KeycloakRealmRole has been finished")

	return nil
}

func (ch *Chain) run(
	ctx context.Context,
	realm *keycloakApi.KeycloakRealm,
	role *keycloakApi.Role,
	realmName string,
	roleCtx *RoleContext) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Starting create role", "role", role.Name)

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, role, realmName, roleCtx)
		// Refresh Token
		if helper.IsUnauthorizedError(err) {
			log.Info("KeycloakRealmRole chain refreshToken")
			cl, errCl := ch.refreshToken(ctx, realm)
			if errCl != nil {
				log.Error(errCl, "KeycloakRealmRole chain refreshToken error")

				return errCl
			}

			h.WithKeycloakApiClient(cl)

			err = h.Serve(ctx, role, realmName, roleCtx)
		}

		if err != nil {
			log.Info("KeycloakRealmRole chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Done create role", "role", role.Name)

	return nil
}

func (ch *Chain) refreshToken(
	ctx context.Context,
	realm *keycloakApi.KeycloakRealm) (*keycloakv2.KeycloakClient, error) {
	return ch.helper.CreateKeycloakClientV2FromConfigRef(ctx, realm)
}

func MakeChain(
	helper ControllerHelper,
	kClientV2 *keycloakv2.KeycloakClient,
	k8sClient client.Client) *Chain {
	ch := &Chain{
		helper: helper,
		flush:  NewFlush(kClientV2, k8sClient).Flush,
	}

	ch.Use(
		NewCreateOrUpdateRole(kClientV2),
		NewSyncComposites(kClientV2),
		NewMakeDefault(kClientV2),
	)

	return ch
}
