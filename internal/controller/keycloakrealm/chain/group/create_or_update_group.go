package group

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

type CreateOrUpdateGroup struct{}

func NewCreateOrUpdateGroup() *CreateOrUpdateGroup {
	return &CreateOrUpdateGroup{}
}

func (h *CreateOrUpdateGroup) Serve(
	ctx context.Context,
	group *keycloakApiAlpha.Group,
	kClient *keycloakv2.KeycloakClient,
	groupCtx *GroupContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating group in Keycloak")

	realm := groupCtx.RealmName

	var (
		existingGroup *keycloakv2.GroupRepresentation
		err           error
	)

	if groupCtx.ParentGroupID != "" {
		existingGroup, _, err = kClient.Groups.FindChildGroupByName(ctx, realm, groupCtx.ParentGroupID, group.Name)
	} else {
		existingGroup, _, err = kClient.Groups.FindGroupByName(ctx, realm, group.Name)
	}

	if err != nil && !keycloakv2.IsNotFound(err) {
		return fmt.Errorf("unable to search for group %q: %w", group.Name, err)
	}

	if keycloakv2.IsNotFound(err) {
		groupRep := keycloakv2.GroupRepresentation{
			Name:        &group.Name,
			Description: &group.Description,
			Path:        &group.Path,
			Attributes:  &group.Attributes,
		}

		var resp *keycloakv2.Response

		if groupCtx.ParentGroupID != "" {
			resp, err = kClient.Groups.CreateChildGroup(ctx, realm, groupCtx.ParentGroupID, groupRep)
		} else {
			resp, err = kClient.Groups.CreateGroup(ctx, realm, groupRep)
		}

		if err != nil {
			return fmt.Errorf("unable to create group %q: %w", group.Name, err)
		}

		groupCtx.GroupID = keycloakv2.GetResourceIDFromResponse(resp)
		log.Info("Group created", "groupID", groupCtx.GroupID)
	} else {
		groupCtx.GroupID = *existingGroup.Id
		existingGroup.Description = &group.Description
		existingGroup.Path = &group.Path
		existingGroup.Attributes = &group.Attributes

		if _, err := kClient.Groups.UpdateGroup(ctx, realm, groupCtx.GroupID, *existingGroup); err != nil {
			return fmt.Errorf("unable to update group %q: %w", group.Name, err)
		}

		log.Info("Group updated", "groupID", groupCtx.GroupID)
	}

	return nil
}
