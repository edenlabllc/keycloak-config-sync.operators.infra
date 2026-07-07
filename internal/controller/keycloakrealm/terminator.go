package keycloakrealm

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	roleChan "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/role"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

// Terminator deletes a Keycloak realm during resource cleanup.
type Terminator struct {
	realmSpec                   keycloakApiAlpha.KeycloakRealmSpec
	kcClient                    *keycloakv2.KeycloakClient
	preserveResourcesOnDeletion bool
}

func (t *Terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm", t.realmSpec.RealmName)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")

		return nil
	}

	if err := t.DeleteIdentityProvider(ctx); err != nil {
		return err
	}

	if err := t.DeleteRole(ctx); err != nil {
		return err
	}

	if err := t.DeleteGroup(ctx); err != nil {
		return err
	}

	log.Info("Start deleting keycloak realm")

	if _, err := t.kcClient.Realms.DeleteRealm(ctx, t.realmSpec.RealmName); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Realm not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete keycloak realm: %w", err)
	}

	log.Info("Realm has been deleted")

	return nil
}

func (t *Terminator) DeleteRole(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm", t.realmSpec.RealmName)
	log.Info("Start deleting keycloak realm role")

	roleCH := roleChan.NewRemoveRole(t.kcClient)

	for _, role := range t.realmSpec.Roles {
		if t.realmSpec.DefaultRole != nil && t.realmSpec.DefaultRole.Name == role.Name {
			log.Info("Realm Role use default, skipping deletion.")

			continue
		}

		if err := roleCH.ServeRequest(ctx, &role, t.realmSpec.RealmName); err != nil {
			if keycloakv2.IsNotFound(err) {
				log.Info("Realm Role not found, skipping deletion.", "role", role.Name)

				return nil
			}

			return fmt.Errorf("[%s] failed to delete keycloak realm role: %w", role.Name, err)
		}
	}

	log.Info("Realm Role has been deleted")

	return nil
}

func (t *Terminator) DeleteGroup(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm", t.realmSpec.RealmName)
	log.Info("Start deleting keycloak realm group")

	for _, group := range helper.SortRealmGroupByParentFirstChild(t.realmSpec.Groups) {
		existingGroup, _, err := t.kcClient.Groups.FindGroupByName(ctx, t.realmSpec.RealmName, group.Name)
		if err != nil {
			if keycloakv2.IsNotFound(err) {
				log.Info("Group not found, skipping deletion")

				return nil
			}

			return fmt.Errorf("unable to search for group %q: %w", group.Name, err)
		}

		if _, err = t.kcClient.Groups.DeleteGroup(ctx, t.realmSpec.RealmName, *existingGroup.Id); err != nil {
			if keycloakv2.IsNotFound(err) {
				log.Info("Group not found, skipping deletion")

				return nil
			}

			return fmt.Errorf("unable to delete group: %q: %w", group.Name, err)
		}
	}

	log.Info("Realm Group has been deleted")

	return nil
}

func (t *Terminator) DeleteIdentityProvider(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm", t.realmSpec.RealmName)
	log.Info("Start deleting keycloak realm IdentityProvider")

	for _, idp := range t.realmSpec.IdentityProviders {
		log.Info("Start deleting keycloak realm idp", "alias", idp.Alias)

		if _, err := t.kcClient.IdentityProviders.DeleteIdentityProvider(ctx, t.realmSpec.RealmName, idp.Alias); err != nil {
			if adapter.IsErrNotFound(err) {
				log.Info("Realm idp not found, skipping deletion.")

				return nil
			}

			return fmt.Errorf("unable to delete realm idp %q: %w", idp.Alias, err)
		}

		log.Info("Realm idp has been deleted", "alias", idp.Alias)
	}

	log.Info("Realm IdentityProvider has been deleted")

	return nil
}

// MakeTerminator creates a Terminator for the given realm.
func MakeTerminator(
	realmSpec keycloakApiAlpha.KeycloakRealmSpec,
	kcClient *keycloakv2.KeycloakClient,
	preserveResourcesOnDeletion bool) *Terminator {
	return &Terminator{
		realmSpec:                   realmSpec,
		kcClient:                    kcClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func makeTerminator(
	realmSpec keycloakApiAlpha.KeycloakRealmSpec,
	kcClient *keycloakv2.KeycloakClient,
	preserveResourcesOnDeletion bool) *Terminator {
	return MakeTerminator(realmSpec, kcClient, preserveResourcesOnDeletion)
}
