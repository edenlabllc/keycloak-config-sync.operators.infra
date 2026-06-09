package client

import (
	"context"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
)

const (
	// RealmManagementClient built-in Keycloak client for the realm
	// This client manages admin fine-grained permissions for other clients.
	RealmManagementClient = "realm-management"
)

type PutAdminFineGrainedPermissions struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutAdminFineGrainedPermissions(keycloakApiClient keycloak.Client, k8sClient client.Client) *PutAdminFineGrainedPermissions {
	return &PutAdminFineGrainedPermissions{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutAdminFineGrainedPermissions) Serve(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	featureFlagEnabled, err := el.keycloakApiClient.FeatureFlagEnabled(ctx, adapter.FeatureFlagAdminFineGrainedAuthz)
	if err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync admin fine-grained permissions: %s", err.Error()))

		return fmt.Errorf("failed to check feature flag ADMIN_FINE_GRAINED_AUTHZ: %w", err)
	}

	if !featureFlagEnabled {
		log.Info("Feature flag is not enabled, skipping admin fine grained permissions", "featureFlag", adapter.FeatureFlagAdminFineGrainedAuthz)

		return nil
	}

	clientID, err := el.keycloakApiClient.GetClientID(keycloakClient.Client.ClientId, realmName)
	if err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync admin fine-grained permissions: %s", err.Error()))

		return fmt.Errorf("failed to get client id: %w", err)
	}

	if err := el.putKeycloakClientAdminFineGrainedPermissions(ctx, keycloakClient, realmName, clientID); err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync admin fine-grained permissions: %s", err.Error()))

		return fmt.Errorf("unable to put keycloak client admin fine grained permissions: %w", err)
	}

	if keycloakClient.Client.AdminFineGrainedPermissionsEnabled && keycloakClient.Client.Permission != nil {
		if err := el.putKeycloakClientAdminPermissionPolicies(ctx, keycloakClient, realmName, clientID); err != nil {
			el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync admin fine-grained permissions: %s", err.Error()))

			return fmt.Errorf("unable to put keycloak client admin permission policies: %w", err)
		}
	}

	el.setSuccessCondition(ctx, keycloakClient, "Admin fine-grained permissions synchronized")

	return nil
}

func (el *PutAdminFineGrainedPermissions) setFailureCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionAdminFineGrainedPermissionsV1Synced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutAdminFineGrainedPermissions) setSuccessCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionAdminFineGrainedPermissionsV1Synced,
		metav1.ConditionTrue,
		chain.ReasonAdminFineGrainedPermissionsV1Synced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutAdminFineGrainedPermissions) putKeycloakClientAdminFineGrainedPermissions(
	ctx context.Context,
	keycloakClient *DataClient,
	realmName, clientID string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client admin fine grained permissions")

	managementPermissions := adapter.ManagementPermissionRepresentation{
		Enabled: &keycloakClient.Client.AdminFineGrainedPermissionsEnabled,
	}

	if err := el.keycloakApiClient.UpdateClientManagementPermissions(realmName, clientID, managementPermissions); err != nil {
		return fmt.Errorf("unable to update client management permissions: %w", err)
	}

	reqLog.Info("End put keycloak client admin fine grained permissions")

	return nil
}

func (el *PutAdminFineGrainedPermissions) putKeycloakClientAdminPermissionPolicies(
	ctx context.Context,
	keycloakClient *DataClient,
	realmName, clientID string) error {
	reqLog := ctrl.LoggerFrom(ctx)
	reqLog.Info("Start put keycloak client admin permission policies")

	realmManagementClientID, err := el.keycloakApiClient.GetClientID(RealmManagementClient, realmName)
	if err != nil {
		return fmt.Errorf("failed to get %s client id: %w", RealmManagementClient, err)
	}

	realmManagementPermissions, err := el.keycloakApiClient.GetPermissions(ctx, realmName, realmManagementClientID)
	if err != nil {
		return fmt.Errorf("failed to get permissions for %s client: %w", RealmManagementClient, err)
	}

	existingClientPermissions, err := el.keycloakApiClient.GetClientManagementPermissions(realmName, clientID)
	if err != nil {
		return fmt.Errorf("failed to get client permissions: %w", err)
	}

	existingScopePermissions := *existingClientPermissions.ScopePermissions

	for i := 0; i < len(keycloakClient.Client.Permission.ScopePermissions); i++ {
		name := keycloakClient.Client.Permission.ScopePermissions[i].Name
		reqLog.Info("Processing scope permission", scopeLogKey, name)

		if _, ok := existingScopePermissions[name]; !ok {
			return fmt.Errorf("scope %s not found in permissions", name)
		}

		permissionName := fmt.Sprintf("%s.permission.client.%s", name, clientID)

		if permission, ok := realmManagementPermissions[permissionName]; ok {
			permission.Policies = &keycloakClient.Client.Permission.ScopePermissions[i].Policies
			if err = el.keycloakApiClient.UpdatePermission(ctx, realmName, realmManagementClientID, permission); err != nil {
				return fmt.Errorf("failed to update permission %s: %w", permissionName, err)
			}

			reqLog.Info("Scope permission updated", scopeLogKey, name, permissionLogKey, permissionName)
		}
	}

	reqLog.Info("End put keycloak client admin permission policies")

	return nil
}
