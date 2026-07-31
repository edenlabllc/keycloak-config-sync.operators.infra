package scope

import (
	"context"
	"fmt"
	"maps"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// secretRef is an interface for getting secret from ref.
type secretRef interface {
	GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (string, error)
}

type CreateScope struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
	secretRef         secretRef
}

func NewCreateScope(
	keycloakApiClient keycloak.Client,
	k8sClient client.Client,
	secretRef secretRef) *CreateScope {
	return &CreateScope{
		keycloakApiClient: keycloakApiClient,
		k8sClient:         k8sClient,
		secretRef:         secretRef,
	}
}

func (el *CreateScope) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	el.keycloakApiClient = keycloakApiClient
}

func (el *CreateScope) Serve(ctx context.Context, scope *Scope, realmName string) error {
	id, err := el.createScope(ctx, scope, realmName)
	if err != nil {
		return fmt.Errorf("unable to crate keycloak client scope: %w", err)
	}

	scope.ReasonScope.ClientScopeID = id

	return el.updateKeycloakClientScopeStatus(ctx, scope)
}

func (el *CreateScope) updateKeycloakClientScopeStatus(
	ctx context.Context,
	scope *Scope,
) error {
	if err := el.k8sClient.Status().Update(ctx, scope.CRD); err != nil {
		return fmt.Errorf("failed to update KeycloakClientScope status: %w", err)
	}

	return nil
}

func (el *CreateScope) createScope(
	ctx context.Context, scope *Scope, realmName string) (string, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creation of Keycloak client scope name", "scopeName", scope.ClientScope.Name)

	clientScope, err := el.keycloakApiClient.GetClientScope(ctx, scope.ClientScope.Name, realmName)
	if err != nil && !adapter.IsErrNotFound(err) {
		return "", fmt.Errorf("unable to get client scope: %w", err)
	}

	cScope := adapter.ClientScope{
		Name:            scope.ClientScope.Name,
		Attributes:      scope.ClientScope.Attributes,
		Protocol:        scope.ClientScope.Protocol,
		ProtocolMappers: convertProtocolMappers(scope.ClientScope.ProtocolMappers),
		Description:     scope.ClientScope.Description,
		Type:            scope.ClientScope.GetType(),
	}

	if err == nil {
		if scope.ReasonScope.ClientScopeID == "" || scope.ReasonScope.ClientScopeID != clientScope.ID {
			scope.ReasonScope.ClientScopeID = clientScope.ID
		}

		if err = el.keycloakApiClient.UpdateClientScope(ctx,
			realmName, scope.ReasonScope.ClientScopeID, &cScope); err != nil {
			return "", fmt.Errorf("unable to update client scope: %w", err)
		}

		return scope.ReasonScope.ClientScopeID, nil
	}

	id, err := el.keycloakApiClient.CreateClientScope(ctx, realmName, &cScope)
	if err != nil {
		return "", fmt.Errorf("unable to create client scope: %w", err)
	}

	scope.ReasonScope.ClientScopeID = id

	return scope.ReasonScope.ClientScopeID, nil
}

func convertProtocolMappers(mappers []keycloakApi.ProtocolMapper) []adapter.ProtocolMapper {
	aMappers := make([]adapter.ProtocolMapper, 0, len(mappers))

	for _, m := range mappers {
		pm := adapter.ProtocolMapper{
			Name:           m.Name,
			Config:         make(map[string]string, len(m.Config)),
			ProtocolMapper: m.ProtocolMapper,
			Protocol:       m.Protocol,
		}

		maps.Copy(pm.Config, m.Config)

		aMappers = append(aMappers, pm)
	}

	return aMappers
}
