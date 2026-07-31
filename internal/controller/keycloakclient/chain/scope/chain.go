package scope

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/secretref"
)

type ReasonScope struct {
	ClientScopeID string
	FailureCount  int64
	Conditions    []metav1.Condition
}

type Scope struct {
	CRD         *keycloakApi.KeycloakClient
	ClientScope keycloakApi.ClientScope
	ReasonScope *ReasonScope
}

type ClientHandler interface {
	Serve(
		ctx context.Context,
		keycloakClient *Scope,
		realmName string,
	) error
	WithKeycloakApiClient(keycloakApiClient keycloak.Client)
}

type ControllerHelper interface {
	CreateKeycloakClientFromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (keycloak.Client, error)
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

	log.Info("Starting KeycloakClientScope chain")

	if keycloakClientSettings.Spec.ClientScope == nil {
		return nil
	}

	for _, scope := range *keycloakClientSettings.Spec.ClientScope {
		name := helper.RemoveSpecialChar(scope.Name)

		dataScope := &Scope{
			CRD:         keycloakClientSettings,
			ClientScope: scope,
			ReasonScope: &ReasonScope{
				ClientScopeID: keycloakClientSettings.Status.GetClientScopeIDByName(name),
				Conditions:    make([]metav1.Condition, 0),
			},
		}

		for i := 0; i < len(ch.handlers); i++ {
			h := ch.handlers[i]

			err := h.Serve(ctx, dataScope, realmName)
			// Refresh Token
			if helper.IsUnauthorizedError(err) {
				log.Info("KeycloakClientScope chain refreshToken")
				cl, errCl := ch.refreshToken(ctx, keycloakClientSettings)
				if errCl != nil {
					log.Error(errCl, "KeycloakClientScope chain refreshToken error")

					return errCl
				}

				h.WithKeycloakApiClient(cl)

				err = h.Serve(ctx, dataScope, realmName)
			}

			if err != nil {
				log.Info("KeycloakClientScope chain finished with error")

				return fmt.Errorf("failed to serve handler: %w", err)
			}
		}

		keycloakClientSettings.Status.PutClientScopeIDByName(name,
			dataScope.ReasonScope.ClientScopeID)
	}

	log.Info("Handling of KeycloakClientScope has been finished")

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
		NewCreateScope(keycloakApiClient, k8sClient, secretref.NewSecretRef(k8sClient)),
	)

	return c
}
