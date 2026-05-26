package chain

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/handler"
)

type RealmLocalizationTexts struct {
	next handler.RealmHandler
}

func (h RealmLocalizationTexts) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClientV2 *keycloakv2.KeycloakClient) error {
	if realm.Spec.Localization == nil || len(realm.Spec.Localization.LocalizationTexts) == 0 {
		return nextServeOrNil(ctx, h.next, realm, kClientV2)
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Start applying realm localization texts")

	if err := SyncLocalizationTexts(ctx, kClientV2, realm.Spec.RealmName, realm.Spec.Localization.LocalizationTexts); err != nil {
		return err
	}

	log.Info("Realm localization texts applied")

	return nextServeOrNil(ctx, h.next, realm, kClientV2)
}

// SyncLocalizationTexts posts any missing or changed locale keys to Keycloak.
// Keys absent from desired but already present in Keycloak are left unchanged —
// removal is not supported by the POST /localization API.
func SyncLocalizationTexts(ctx context.Context, kClientV2 *keycloakv2.KeycloakClient, realmName string, desired map[string]map[string]string) error {
	log := ctrl.LoggerFrom(ctx)

	for _, locale := range slices.Sorted(maps.Keys(desired)) {
		kv := desired[locale]
		if len(kv) == 0 {
			continue
		}

		current, _, err := kClientV2.Realms.GetRealmLocalization(ctx, realmName, locale)
		if err != nil {
			log.V(1).Info("Failed to get current localization, will overwrite", "locale", locale, "err", err)
		} else if localesAlreadyInSync(current, kv) {
			continue
		}

		if _, err := kClientV2.Realms.PostRealmLocalization(ctx, realmName, locale, maps.Clone(kv)); err != nil {
			return fmt.Errorf("unable to set realm localization for locale %q: %w", locale, err)
		}
	}

	return nil
}

// localesAlreadyInSync returns true when every key in desired already exists with
// the same value in current. Extra keys present in current but absent from desired
// are ignored — removal is not supported by the Keycloak POST /localization API.
func localesAlreadyInSync(current, desired map[string]string) bool {
	for k, v := range desired {
		if current[k] != v {
			return false
		}
	}

	return true
}
