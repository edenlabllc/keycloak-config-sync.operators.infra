package realm

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/realm/handler"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	keycloakv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type PutRealm struct {
	next   handler.RealmHandler
	client client.Client
}

func (h PutRealm) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start putting realm")

	realmName := realm.Spec.RealmName

	_, _, err := kClientV2.Realms.GetRealm(ctx, realmName)
	if err != nil {
		if !keycloakv2.IsNotFound(err) {
			return fmt.Errorf("unable to check realm existence: %w", err)
		}

		if err = h.createRealm(ctx, realm, kClientV2); err != nil {
			return err
		}

		return nextServeOrNil(ctx, h.next, realm, kClientV2)
	}

	rLog.Info("Realm already exists")

	return nextServeOrNil(ctx, h.next, realm, kClientV2)
}

func (h PutRealm) createRealm(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	realmName := realm.Spec.RealmName

	if _, err := kClientV2.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: ptr.To(true),
	}); err != nil {
		return fmt.Errorf("unable to create realm with default config: %w", err)
	}

	if err := h.putRealmRoles(ctx, realm, kClientV2); err != nil {
		return fmt.Errorf("unable to create realm roles on no sso scenario: %w", err)
	}

	log.WithValues("realm name", realmName).Info("End putting realm!")

	return nil
}

func (h PutRealm) putRealmRoles(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	realmName := realm.Spec.RealmName
	allRoles := make(map[string]struct{})

	for _, u := range realm.Spec.Users {
		for _, rr := range u.RealmRoles {
			allRoles[rr] = struct{}{}
		}
	}

	for r := range allRoles {
		_, _, err := kClientV2.Roles.GetRealmRole(ctx, realmName, r)
		if err != nil {
			if !keycloakv2.IsNotFound(err) {
				return fmt.Errorf("unable to check realm role existence: %w", err)
			}

			roleName := r
			if _, err := kClientV2.Roles.CreateRealmRole(ctx, realmName, keycloakv2.RoleRepresentation{Name: &roleName}); err != nil {
				return fmt.Errorf("unable to create new realm role: %w", err)
			}
		}
	}

	return nil
}
