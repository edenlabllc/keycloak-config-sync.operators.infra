package role

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

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
}

type Chain struct {
	handlers []RealmRoleHandler
}

func (ch *Chain) Use(handlers ...RealmRoleHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	realm *keycloakApi.KeycloakRealm,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting Keycloak Realm Role chain")

	for _, role := range realm.Spec.Roles {
		roleCtx := &RoleContext{}

		if err := ch.run(ctx, &role, realm.Spec.RealmName, roleCtx); err != nil {
			return err
		}
	}

	log.Info("Handling of KeycloakRealmRole has been finished")

	return nil
}

func (ch *Chain) run(
	ctx context.Context,
	role *keycloakApi.Role,
	realmName string,
	roleCtx *RoleContext) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Starting create role", "role", role.Name)

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, role, realmName, roleCtx)
		if err != nil {
			log.Info("KeycloakRealmRole chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Done create role", "role", role.Name)

	return nil
}

func MakeChain(kClientV2 *keycloakv2.KeycloakClient) *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateOrUpdateRole(kClientV2),
		NewSyncComposites(kClientV2),
		NewMakeDefault(kClientV2),
	)

	return ch
}
