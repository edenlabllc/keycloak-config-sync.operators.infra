package realm

import (
	"context"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/realm/handler"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/realmbuilder"
)

type RealmSettings struct {
	next handler.RealmHandler
}

func (h RealmSettings) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start updating of Keycloak realm settings")

	if err := realmbuilder.ApplyRealmEventConfig(ctx, realm.Spec.RealmName, realm.Spec.RealmEventConfig, kClientV2.Realms); err != nil {
		return err
	}

	overlay := realmbuilder.BuildRealmRepresentationFromV1(realm)

	if err := realmbuilder.ApplyRealmSettings(ctx, realm.Spec.RealmName, overlay, kClientV2.Realms); err != nil {
		return err
	}

	rLog.Info("Realm settings is updating done.")

	return nextServeOrNil(ctx, h.next, realm, kClientV2)
}
