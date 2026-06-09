package realm

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/realm/handler"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

const TargetRealmLabel = "targetRealm"

type SetLabels struct {
	next   handler.RealmHandler
	client client.Client
}

func (s SetLabels) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	if realm.Labels == nil {
		realm.Labels = make(map[string]string)
	}

	if tr, ok := realm.Labels[TargetRealmLabel]; !ok || tr != realm.Spec.RealmName {
		realm.Labels[TargetRealmLabel] = realm.Spec.RealmName
	}

	if err := s.client.Update(ctx, realm); err != nil {
		return fmt.Errorf("unable to update realm with new labels, realm: %+v: %w", realm, err)
	}

	return nextServeOrNil(ctx, s.next, realm, kClientV2)
}
