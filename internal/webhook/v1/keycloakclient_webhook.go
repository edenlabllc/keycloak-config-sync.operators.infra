package v1

import (
	"context"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// log is for logging in this package.
var keycloakclientlog = logf.Log.WithName("keycloakclient-resource")

// SetupKeycloakClientWebhookWithManager registers the webhook for KeycloakClient in the manager.
func SetupKeycloakClientWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &keycloakApiAlpha.KeycloakClient{}).
		WithDefaulter(&KeycloakClientCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-config-idp-edenlab-io-v1alpha1-keycloakclient,mutating=true,failurePolicy=fail,sideEffects=None,groups=config.idp.edenlab.io,resources=keycloakclients,verbs=create;update,versions=v1alpha1,name=mkeycloakclient-v1.kb.io,admissionReviewVersions=v1

// KeycloakClientCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind KeycloakClient when those are created or updated.
type KeycloakClientCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind KeycloakClient.
func (d *KeycloakClientCustomDefaulter) Default(_ context.Context, obj *keycloakApiAlpha.KeycloakClient) error {
	keycloakclientlog.Info("Defaulting for KeycloakClient", "name", obj.GetName())

	for i, client := range obj.Spec.Client {
		updated := d.applyDefaults(obj.GetName(), &obj.Spec.Client[i])

		if updated {
			keycloakclientlog.Info("Applied defaults to KeycloakClient",
				"name", obj.GetName(), "clientID", client.ClientId)
		}
	}

	return nil
}

const (
	ClientAttributeLogoutRedirectUris         = "post.logout.redirect.uris"
	ClientAttributeLogoutRedirectUrisDefValue = "+"
)

// applyDefaults applies default values to KeycloakClient.
func (r *KeycloakClientCustomDefaulter) applyDefaults(objName string, keycloakClient *keycloakApiAlpha.Client) bool {
	if keycloakClient.Attributes == nil {
		keycloakClient.Attributes = make(map[string]string)
	}

	updated := false

	if _, ok := keycloakClient.Attributes[ClientAttributeLogoutRedirectUris]; !ok {
		// set default value for logout redirect uris to "+" is required for correct logout from keycloak
		keycloakClient.Attributes[ClientAttributeLogoutRedirectUris] = ClientAttributeLogoutRedirectUrisDefValue
		updated = true
	}

	if keycloakClient.WebOrigins == nil && keycloakClient.WebUrl != "" {
		keycloakClient.WebOrigins = []string{
			keycloakClient.WebUrl,
		}

		updated = true
	}

	if migrated := r.migrateClientRoles(objName, keycloakClient); migrated {
		updated = true
	}

	if keycloakClient.ServiceAccount != nil {
		if migrated := r.migrateServiceAccountAttributes(objName, keycloakClient); migrated {
			updated = true
		}
	}

	return updated
}

// migrateClientRoles migrates ClientRoles to ClientRolesV2 format.
// This function converts the old string-based client roles to the new ClientRole struct format.
// It only performs migration if ClientRolesV2 is empty and ClientRoles is not empty.
func (r *KeycloakClientCustomDefaulter) migrateClientRoles(objName string, keycloakClient *keycloakApiAlpha.Client) bool {
	if len(keycloakClient.ClientRolesV2) == 0 && len(keycloakClient.ClientRoles) > 0 {
		keycloakclientlog.Info("Migrating ClientRoles to ClientRolesV2",
			"name", objName,
			"clientID", keycloakClient.ClientId,
			"roleCount", len(keycloakClient.ClientRoles))

		// Convert string-based roles to ClientRole structs
		for _, roleName := range keycloakClient.ClientRoles {
			clientRole := keycloakApiAlpha.ClientRole{
				Name: roleName,
				// Composite field is left empty as it wasn't available in the old format
			}
			keycloakClient.ClientRolesV2 = append(keycloakClient.ClientRolesV2, clientRole)
		}

		// Keep the original ClientRoles field for backward compatibility
		// keycloakClient.Spec.ClientRoles remains unchanged

		return true
	}

	return false
}

// migrateServiceAccountAttributes migrates Attributes to AttributesV2 format.
// This function converts the old string-based attributes to the new []string format.
// It only performs migration if AttributesV2 is empty and Attributes is not empty.
func (r *KeycloakClientCustomDefaulter) migrateServiceAccountAttributes(objName string, keycloakClient *keycloakApiAlpha.Client) bool {
	if len(keycloakClient.ServiceAccount.AttributesV2) == 0 && len(keycloakClient.ServiceAccount.Attributes) > 0 {
		keycloakclientlog.Info("Migrating ServiceAccount.Attributes to AttributesV2",
			"name", objName,
			"clientID", keycloakClient.ClientId,
			"attributeCount", len(keycloakClient.ServiceAccount.Attributes))

		keycloakClient.ServiceAccount.AttributesV2 = make(map[string][]string, len(keycloakClient.ServiceAccount.Attributes))

		// Convert string bases attributes to []string
		for attr, value := range keycloakClient.ServiceAccount.Attributes {
			keycloakClient.ServiceAccount.AttributesV2[attr] = []string{value}
		}

		// Keep the original Attributes field for backward compatibility
		// keycloakClient.Spec.ServiceAccount.Attributes remains unchanged

		return true
	}

	return false
}
