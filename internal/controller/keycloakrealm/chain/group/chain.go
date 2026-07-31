package group

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type FlushFunc func(
	ctx context.Context,
	kClientV2 *keycloakv2.KeycloakClient,
	k8sClient client.Client,
	realm *keycloakApiAlpha.KeycloakRealm,
	groupIDs []string) error

// GroupContext holds data that is passed between chain handlers.
type GroupContext struct {
	// GroupID is the Keycloak group ID, set by CreateOrUpdateGroup handler.
	GroupID string

	// ParentGroupID is the parent group's Keycloak ID (empty if top-level).
	ParentGroupID string

	// RealmName is the Keycloak realm name.
	RealmName string
}

// RealmGroupHandler defines the interface for chain handlers.
type RealmGroupHandler interface {
	Serve(
		ctx context.Context,
		group *keycloakApiAlpha.Group,
		kClientV2 *keycloakv2.KeycloakClient,
		groupCtx *GroupContext,
	) error
}

type ControllerHelper interface {
	CreateKeycloakClientFromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (keycloak.Client, error)
	CreateKeycloakClientV2FromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (*keycloakv2.KeycloakClient, error)
}

// Chain executes a sequence of RealmGroupHandler handlers.
type Chain struct {
	helper   ControllerHelper
	handlers []RealmGroupHandler
	flush    FlushFunc
}

func (ch *Chain) Use(handlers ...RealmGroupHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	realm *keycloakApiAlpha.KeycloakRealm,
	kClientV2 *keycloakv2.KeycloakClient,
	k8sClient client.Client,
) error {
	groupIDs := make([]string, len(realm.Spec.Groups))
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting Keycloak Realm Group chain")

	for _, group := range helper.SortRealmGroupByParent(realm.Spec.Groups) {
		parentGroupID, err := ch.getParentGroupID(
			ctx, realm.Spec.RealmName, group, kClientV2)
		if err != nil {
			return err
		}

		groupCtx := &GroupContext{
			RealmName:     realm.Spec.RealmName,
			ParentGroupID: parentGroupID,
		}

		if err = ch.run(ctx, realm, &group, kClientV2, groupCtx); err != nil {
			return err
		}

		groupIDs = append(groupIDs, groupCtx.GroupID)
	}

	if err := ch.flush(ctx, kClientV2, k8sClient, realm, groupIDs); err != nil {
		return err
	}

	log.Info("Handling of Keycloak Realm Group has been finished")

	return nil
}

func (ch *Chain) run(ctx context.Context,
	realm *keycloakApiAlpha.KeycloakRealm,
	group *keycloakApiAlpha.Group,
	kClientV2 *keycloakv2.KeycloakClient,
	groupCtx *GroupContext) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Starting create group", "group", group.Name)

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, group, kClientV2, groupCtx)
		// Refresh Token
		if helper.IsUnauthorizedError(err) {
			log.Info("Realm Group chain refreshToken")
			cl, errCl := ch.refreshToken(ctx, realm)
			if errCl != nil {
				log.Error(errCl, "Realm Group chain refreshToken error")

				return errCl
			}

			err = h.Serve(ctx, group, cl, groupCtx)
		}

		if err != nil {
			log.Info("Realm Group chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Done create group", "group", group.Name)

	return nil
}

func (ch *Chain) getParentGroupID(
	ctx context.Context,
	realmName string,
	cGroup keycloakApiAlpha.Group,
	kClientV2 *keycloakv2.KeycloakClient,
) (string, error) {
	// No parent group specified
	if cGroup.ParentGroup == nil {
		return "", nil
	}

	existingGroup, _, err := kClientV2.Groups.FindGroupByName(ctx, realmName, cGroup.Name)
	if err != nil {
		return "", fmt.Errorf("unable to search for group %q: %w", cGroup.Name, err)
	}

	return *existingGroup.Id, nil
}

func (ch *Chain) refreshToken(
	ctx context.Context,
	realm *keycloakApiAlpha.KeycloakRealm) (*keycloakv2.KeycloakClient, error) {
	return ch.helper.CreateKeycloakClientV2FromConfigRef(ctx, realm)
}

func MakeChain(helper ControllerHelper) *Chain {
	ch := &Chain{
		helper: helper,
		flush:  NewFlush().Flush,
	}

	ch.Use(
		NewCreateOrUpdateGroup(),
		NewSyncRealmRoles(),
		NewSyncClientRoles(),
		NewSyncSubGroups(),
	)

	return ch
}
