package keycloakrealm

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	roleChan "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/role"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

// Terminator deletes a Keycloak realm during resource cleanup.
type Terminator struct {
	helper                      Helper
	realm                       *keycloakApiAlpha.KeycloakRealm
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

		err := roleCH.ServeRequest(ctx, &role, t.realmSpec.RealmName)
		// Refresh Token
		if helper.IsUnauthorizedError(err) {
			log.Info("keycloak realm role refreshToken")
			cl, errCl := t.refreshToken(ctx)
			if errCl != nil {
				log.Error(errCl, "keycloak realm role refreshToken error")

				return errCl
			}

			roleCH.WithKeycloakApiClient(cl)

			err = roleCH.ServeRequest(ctx, &role, t.realmSpec.RealmName)
		}

		if err != nil {
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
		if err := t.deleteGroupItem(ctx, log, group.Name); err != nil {
			if helper.IsUnauthorizedError(err) {
				log.Info("Realm DeleteGroup refreshToken")
				cl, errCl := t.refreshToken(ctx)
				if errCl != nil {
					log.Error(errCl, "Realm DeleteGroup refreshToken error")

					return errCl
				}

				t.kcClient = cl

				return t.DeleteGroup(ctx)
			}

			return err
		}
	}

	log.Info("Realm Group has been deleted")

	return nil
}

func (t *Terminator) deleteGroupItem(ctx context.Context, log logr.Logger, groupName string) error {
	existingGroup, _, err := t.kcClient.Groups.FindGroupByName(ctx, t.realmSpec.RealmName, groupName)
	if err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Group not found, skipping deletion")

			return nil
		}

		return fmt.Errorf("unable to search for group %q: %w", groupName, err)
	}

	if _, err = t.kcClient.Groups.DeleteGroup(ctx, t.realmSpec.RealmName, *existingGroup.Id); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Group not found, skipping deletion")

			return nil
		}

		return fmt.Errorf("unable to delete group: %q: %w", groupName, err)
	}

	return nil
}

func (t *Terminator) DeleteIdentityProvider(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm", t.realmSpec.RealmName)
	log.Info("Start deleting keycloak realm IdentityProvider")

	for _, idp := range t.realmSpec.IdentityProviders {
		log.Info("Start deleting keycloak realm idp", "alias", idp.Alias)

		if _, err := t.kcClient.IdentityProviders.DeleteIdentityProvider(ctx, t.realmSpec.RealmName, idp.Alias); err != nil {
			if helper.IsUnauthorizedError(err) {
				log.Info("IdentityProvider refreshToken")
				cl, errCl := t.refreshToken(ctx)
				if errCl != nil {
					log.Error(errCl, "IdentityProvider refreshToken error")

					return errCl
				}

				t.kcClient = cl

				return t.DeleteIdentityProvider(ctx)
			}

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

func (t *Terminator) refreshToken(ctx context.Context) (*keycloakv2.KeycloakClient, error) {
	return t.helper.CreateKeycloakClientV2FromConfigRef(ctx, t.realm)
}

// MakeTerminator creates a Terminator for the given realm.
func MakeTerminator(
	helper Helper,
	realm *keycloakApiAlpha.KeycloakRealm,
	realmSpec keycloakApiAlpha.KeycloakRealmSpec,
	kcClient *keycloakv2.KeycloakClient,
	preserveResourcesOnDeletion bool) *Terminator {
	return &Terminator{
		helper:                      helper,
		realm:                       realm,
		realmSpec:                   realmSpec,
		kcClient:                    kcClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func makeTerminator(
	helper Helper,
	realm *keycloakApiAlpha.KeycloakRealm,
	realmSpec keycloakApiAlpha.KeycloakRealmSpec,
	kcClient *keycloakv2.KeycloakClient,
	preserveResourcesOnDeletion bool) *Terminator {
	return MakeTerminator(helper, realm, realmSpec, kcClient, preserveResourcesOnDeletion)
}
