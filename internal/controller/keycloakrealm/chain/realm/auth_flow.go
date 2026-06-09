package realm

import (
	"context"
	"fmt"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/realm/handler"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type AuthFlow struct {
	next handler.RealmHandler
}

func (a AuthFlow) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start configuring keycloak realm auth flow", "flow", realm.Spec.BrowserFlow)

	if realm.Spec.BrowserFlow == nil {
		rLog.Info("Browser flow is empty, exit")
		return nextServeOrNil(ctx, a.next, realm, kClientV2)
	}

	if _, err := kClientV2.Realms.SetRealmBrowserFlow(ctx, realm.Spec.RealmName, *realm.Spec.BrowserFlow); err != nil {
		return fmt.Errorf("unable to set realm auth flow: %w", err)
	}

	rLog.Info("End of configuring keycloak realm auth flow")

	return nextServeOrNil(ctx, a.next, realm, kClientV2)
}
