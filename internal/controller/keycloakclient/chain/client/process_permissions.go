package client

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
)

const permissionLogKey = "permission"

type ProcessPermissions struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewProcessPermissions(keycloakApiClient keycloak.Client, k8sClient client.Client) *ProcessPermissions {
	return &ProcessPermissions{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (h *ProcessPermissions) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	h.keycloakApiClient = keycloakApiClient
}

func (h *ProcessPermissions) Serve(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	if keycloakClient.Client.Authorization == nil {
		log.Info("Authorization settings are not specified")
		return nil
	}

	clientID, err := h.keycloakApiClient.GetClientID(keycloakClient.Client.ClientId, realmName)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

		return fmt.Errorf("failed to get client id: %w", err)
	}

	existingPermissions, err := h.keycloakApiClient.GetPermissions(ctx, realmName, clientID)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

		return fmt.Errorf("failed to get permissions: %w", err)
	}

	for i := 0; i < len(keycloakClient.Client.Authorization.Permissions); i++ {
		log.Info("Processing permission", permissionLogKey, keycloakClient.Client.Authorization.Permissions[i].Name)

		var permissionRepresentation *gocloak.PermissionRepresentation

		if permissionRepresentation, err = h.toPermissionRepresentation(ctx, &keycloakClient.Client.Authorization.Permissions[i], clientID, realmName); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

			return fmt.Errorf("failed to convert permission: %w", err)
		}

		existingPermission, ok := existingPermissions[keycloakClient.Client.Authorization.Permissions[i].Name]
		if ok {
			permissionRepresentation.ID = existingPermission.ID
			if err = h.keycloakApiClient.UpdatePermission(ctx, realmName, clientID, *permissionRepresentation); err != nil {
				h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

				return fmt.Errorf("failed to update permission: %w", err)
			}

			log.Info("Permission updated", permissionLogKey, keycloakClient.Client.Authorization.Permissions[i].Name)

			delete(existingPermissions, keycloakClient.Client.Authorization.Permissions[i].Name)

			continue
		}

		if _, err = h.keycloakApiClient.CreatePermission(ctx, realmName, clientID, *permissionRepresentation); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

			return fmt.Errorf("failed to create permission: %w", err)
		}

		log.Info("Permission created", permissionLogKey, keycloakClient.Client.Authorization.Permissions[i].Name)
	}

	if keycloakClient.Client.ReconciliationStrategy != keycloakApi.ReconciliationStrategyAddOnly {
		if err = h.deletePermissions(ctx, existingPermissions, realmName, clientID); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization permissions: %s", err.Error()))

			return err
		}
	}

	h.setSuccessCondition(ctx, keycloakClient, "Authorization permissions synchronized")

	return nil
}

func (h *ProcessPermissions) setFailureCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		chain.ConditionAuthorizationPermissionsSynced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *ProcessPermissions) setSuccessCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		chain.ConditionAuthorizationPermissionsSynced,
		metav1.ConditionTrue,
		chain.ReasonAuthorizationPermissionsSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *ProcessPermissions) deletePermissions(ctx context.Context, existingPermissions map[string]gocloak.PermissionRepresentation, realmName string, clientID string) error {
	log := ctrl.LoggerFrom(ctx)

	for name := range existingPermissions {
		if name == "Default Permission" {
			continue
		}

		if err := h.keycloakApiClient.DeletePermission(ctx, realmName, clientID, *existingPermissions[name].ID); err != nil {
			if !adapter.IsErrNotFound(err) {
				return fmt.Errorf("failed to delete permission: %w", err)
			}
		}

		log.Info("Permission deleted", permissionLogKey, name)
	}

	return nil
}

// toPermissionRepresentation converts keycloakApi.Permission to gocloak.PermissionRepresentation.
func (h *ProcessPermissions) toPermissionRepresentation(ctx context.Context, permission *keycloakApi.Permission, clientID, realm string) (*gocloak.PermissionRepresentation, error) {
	keycloakPermission := getBasePermissionRepresentation(permission)

	if err := h.mapResources(ctx, permission, keycloakPermission, realm, clientID); err != nil {
		return nil, fmt.Errorf("failed to map resources: %w", err)
	}

	if err := h.mapPolicies(ctx, permission, keycloakPermission, realm, clientID); err != nil {
		return nil, fmt.Errorf("failed to map policies: %w", err)
	}

	if permission.Type == keycloakApi.PermissionTypeScope {
		if err := h.mapScopes(ctx, permission, keycloakPermission, realm, clientID); err != nil {
			return nil, fmt.Errorf("failed to map scopes: %w", err)
		}
	}

	return keycloakPermission, nil
}

func (h *ProcessPermissions) mapResources(
	ctx context.Context,
	permission *keycloakApi.Permission,
	keycloakPermission *gocloak.PermissionRepresentation,
	realm,
	clientID string,
) error {
	if len(permission.Resources) == 0 {
		keycloakPermission.Resources = &[]string{}

		return nil
	}

	existingResources, err := h.keycloakApiClient.GetResources(ctx, realm, clientID)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	permissionResources := make([]string, 0, len(permission.Resources))

	for _, r := range permission.Resources {
		existingResource, ok := existingResources[r]
		if !ok {
			return fmt.Errorf("resource %s does not exist", r)
		}

		if existingResource.ID == nil {
			return fmt.Errorf("resource %s does not have ID", r)
		}

		permissionResources = append(permissionResources, *existingResource.ID)
	}

	keycloakPermission.Resources = &permissionResources

	return nil
}

func (h *ProcessPermissions) mapPolicies(
	ctx context.Context,
	permission *keycloakApi.Permission,
	keycloakPermission *gocloak.PermissionRepresentation,
	realm,
	clientID string,
) error {
	if len(permission.Policies) == 0 {
		keycloakPermission.Policies = &[]string{}

		return nil
	}

	existingPolicies, err := h.keycloakApiClient.GetPolicies(ctx, realm, clientID)
	if err != nil {
		return fmt.Errorf("failed to get polices: %w", err)
	}

	permissionPolicies := make([]string, 0, len(permission.Policies))

	for _, r := range permission.Policies {
		existingPolicy, ok := existingPolicies[r]
		if !ok {
			return fmt.Errorf("policy %s does not exist", r)
		}

		if existingPolicy.ID == nil {
			return fmt.Errorf("policy %s does not have ID", r)
		}

		permissionPolicies = append(permissionPolicies, *existingPolicy.ID)
	}

	keycloakPermission.Policies = &permissionPolicies

	return nil
}

func (h *ProcessPermissions) mapScopes(
	ctx context.Context,
	permission *keycloakApi.Permission,
	keycloakPermission *gocloak.PermissionRepresentation,
	realm,
	clientID string,
) error {
	if len(permission.Scopes) == 0 {
		keycloakPermission.Scopes = &[]string{}

		return nil
	}

	existingScopes, err := h.keycloakApiClient.GetScopes(ctx, realm, clientID)
	if err != nil {
		return fmt.Errorf("failed to get scopes: %w", err)
	}

	permissionScopes := make([]string, 0, len(permission.Scopes))

	for _, r := range permission.Scopes {
		existingScope, ok := existingScopes[r]
		if !ok {
			return fmt.Errorf("scope %s does not exist", r)
		}

		if existingScope.ID == nil {
			return fmt.Errorf("scope %s does not have ID", r)
		}

		permissionScopes = append(permissionScopes, *existingScope.ID)
	}

	keycloakPermission.Scopes = &permissionScopes

	return nil
}

func getBasePermissionRepresentation(policy *keycloakApi.Permission) *gocloak.PermissionRepresentation {
	keycloakPermission := &gocloak.PermissionRepresentation{}

	name := policy.Name
	keycloakPermission.Name = &name

	pType := policy.Type
	keycloakPermission.Type = &pType

	desc := policy.Description
	decisionStrategy := gocloak.DecisionStrategy(policy.DecisionStrategy)

	keycloakPermission.DecisionStrategy = &decisionStrategy
	keycloakPermission.Description = &desc

	logic := gocloak.Logic(policy.Logic)
	keycloakPermission.Logic = &logic

	return keycloakPermission
}
