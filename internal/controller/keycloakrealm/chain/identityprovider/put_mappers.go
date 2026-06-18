package identityprovider

import (
	"context"
	"fmt"
	"maps"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
)

type PutIDPMappers struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
	secretRef         refClient
}

func NewPutIDPMappers(keycloakApiClient keycloak.Client, k8sClient client.Client, secretRef refClient) *PutIDPMappers {
	return &PutIDPMappers{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient, secretRef: secretRef}
}

func (el *PutIDPMappers) Serve(ctx context.Context, keycloakRealmIDP *keycloakApiAlpha.IdentityProvider, realmName, _ string) error {
	err := syncIDPMappers(ctx, keycloakRealmIDP, el.keycloakApiClient, realmName)
	if err != nil {
		return fmt.Errorf("unable to sync idp mappers: %w", err)
	}

	return nil
}

func syncIDPMappers(ctx context.Context, idpSpec *keycloakApiAlpha.IdentityProvider, kClient keycloak.Client, targetRealm string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak idp mappers")

	if len(idpSpec.Mappers) == 0 {
		return nil
	}

	mappers, err := kClient.GetIDPMappers(ctx, targetRealm, idpSpec.Alias)
	if err != nil {
		return fmt.Errorf("unable to get idp mappers: %w", err)
	}

	for _, m := range mappers {
		if err = kClient.DeleteIDPMapper(ctx, targetRealm, idpSpec.Alias, m.ID); err != nil {
			return fmt.Errorf("unable to delete idp mapper: %w", err)
		}
	}

	for _, m := range idpSpec.Mappers {
		if m.IdentityProviderAlias == "" {
			m.IdentityProviderAlias = idpSpec.Alias
		}

		if _, err = kClient.CreateIDPMapper(ctx, targetRealm, idpSpec.Alias,
			createKeycloakIDPMapperFromSpec(&m)); err != nil {
			return fmt.Errorf("unable to create idp mapper: %w", err)
		}
	}

	reqLog.Info("End put keycloak idp mappers")

	return nil
}

func createKeycloakIDPMapperFromSpec(spec *keycloakApiAlpha.IdentityProviderMapper) *adapter.IdentityProviderMapper {
	m := &adapter.IdentityProviderMapper{
		IdentityProviderMapper: spec.IdentityProviderMapper,
		Name:                   spec.Name,
		Config:                 make(map[string]string, len(spec.Config)),
		IdentityProviderAlias:  spec.IdentityProviderAlias,
	}

	maps.Copy(m.Config, spec.Config)

	return m
}
