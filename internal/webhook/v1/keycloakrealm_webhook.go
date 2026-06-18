package v1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// keycloakrealmlog is for logging in this package.
var keycloakrealmlog = logf.Log.WithName("keycloakrealm-resource")

// SetupKeycloakRealmWebhookWithManager registers the webhook for KeycloakRealm in the manager.
func SetupKeycloakRealmWebhookWithManager(mgr ctrl.Manager, k8sClient client.Client) error {
	return ctrl.NewWebhookManagedBy(mgr, &keycloakApiAlpha.KeycloakRealm{}).
		WithValidator(NewKeycloakRealmCustomValidator(k8sClient)).
		Complete()
}

// +kubebuilder:webhook:path=/validate-config-idp-edenlab-io-v1alpha1-keycloakrealm,mutating=false,failurePolicy=fail,sideEffects=None,groups=config.idp.edenlab.io,resources=keycloakrealms,verbs=create,versions=v1alpha1,name=vkeycloakrealm-v1.kb.io,admissionReviewVersions=v1

// KeycloakRealmCustomValidator struct is responsible for validating the KeycloakRealm resource
// when it is created, updated, or deleted.
type KeycloakRealmCustomValidator struct {
	k8sclient client.Client
}

func NewKeycloakRealmCustomValidator(k8sclient client.Client) *KeycloakRealmCustomValidator {
	return &KeycloakRealmCustomValidator{
		k8sclient: k8sclient,
	}
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateCreate(ctx context.Context, obj *keycloakApiAlpha.KeycloakRealm) (admission.Warnings, error) {
	keycloakrealmlog.Info("Validation for KeycloakRealm upon creation", "name", obj.GetName())

	for _, group := range obj.Spec.Groups {
		keycloakrealmlog.Info("Validate group webhook", "group", group.Name)
		// Validate that SubGroups and ParentGroup are not used together.
		if len(group.SubGroups) > 0 && group.ParentGroup != nil {
			return nil, fmt.Errorf("cannot use both SubGroups (deprecated) and ParentGroup fields - migrate to ParentGroup approach")
		}
	}

	// Check if the combination of RealmName and KeycloakRef is unique across all KeycloakRealm resources in the cluster.
	existingKeycloakRealms := &keycloakApiAlpha.KeycloakRealmList{}
	if err := v.k8sclient.List(ctx, existingKeycloakRealms); err != nil {
		return nil, fmt.Errorf("failed to list KeycloakRealm resources: %w", err)
	}

	for _, existingRealm := range existingKeycloakRealms.Items {
		isSameResource := existingRealm.Namespace == obj.Namespace && existingRealm.Name == obj.Name

		if existingRealm.Spec.RealmName == obj.Spec.RealmName && !isSameResource {
			return nil, fmt.Errorf(
				"realm name %s is already in use by another KeycloakRealm resource (%s/%s)",
				obj.Spec.RealmName,
				existingRealm.Namespace,
				existingRealm.Name,
			)
		}
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *keycloakApiAlpha.KeycloakRealm) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateDelete(ctx context.Context, obj *keycloakApiAlpha.KeycloakRealm) (admission.Warnings, error) {
	return nil, nil
}
