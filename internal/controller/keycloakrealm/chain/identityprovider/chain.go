package identityprovider

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/secretref"
)

type FlushFunc func(ctx context.Context, realm *keycloakApiAlpha.KeycloakRealm, alias []string) error

type ClientHandler interface {
	Serve(
		ctx context.Context,
		group *keycloakApiAlpha.IdentityProvider,
		realmName, namespace string,
	) error
}

type Chain struct {
	handlers []ClientHandler
	flush    FlushFunc
}

func (ch *Chain) Use(handlers ...ClientHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	realm *keycloakApiAlpha.KeycloakRealm,
) error {
	log := ctrl.LoggerFrom(ctx)
	alias := make([]string, len(realm.Spec.IdentityProviders))

	log.Info("Starting KeycloakIDP chain")

	for _, ip := range realm.Spec.IdentityProviders {
		if err := ch.run(ctx, &ip, realm.Spec.RealmName, realm.Namespace); err != nil {
			return err
		}

		alias = append(alias, ip.Alias)
	}

	if err := ch.flush(ctx, realm, alias); err != nil {
		return err
	}

	log.Info("Handling of KeycloakIDP has been finished")

	return nil
}

func (ch *Chain) run(ctx context.Context,
	ip *keycloakApiAlpha.IdentityProvider,
	realmName, namespace string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Starting create group", "identityProvider", ip.Alias)

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, ip, realmName, namespace)
		if err != nil {
			log.Info("KeycloakIDP chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Done create group", "identityProvider", ip.Alias)

	return nil
}

func MakeChain(
	keycloakApiClient keycloak.Client,
	k8sClient client.Client,
) *Chain {
	c := &Chain{
		flush: NewFlush(keycloakApiClient, k8sClient).Flush,
	}

	c.Use(
		NewPutIDP(keycloakApiClient, k8sClient, secretref.NewSecretRef(k8sClient)),
		NewPutIDPMappers(keycloakApiClient, k8sClient, secretref.NewSecretRef(k8sClient)),
		NewPutAdminFineGrainedPermissions(keycloakApiClient),
	)

	return c
}
