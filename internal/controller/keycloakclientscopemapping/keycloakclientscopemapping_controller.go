package keycloakclientscopemapping

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Nerzal/gocloak/v12"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/objectmeta"
)

const finalizerName = "keycloak.clientscopemapping.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromRealmRef(ctx context.Context, object helper.ObjectWithRealmRef) (keycloak.Client, error)
}

type Reconcile struct {
	client client.Client
	helper Helper
}

func NewReconcile(k8sClient client.Client, controllerHelper Helper) *Reconcile {
	return &Reconcile{
		client: k8sClient,
		helper: controllerHelper,
	}
}

func (r *Reconcile) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.Funcs{
		UpdateFunc: isSpecUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakClientScopeMapping{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakClientScopeMapping controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApi.KeycloakClientScopeMapping)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApi.KeycloakClientScopeMapping)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

// +kubebuilder:rbac:groups=v1.edp.edenlab.io,namespace=keycloak,resources=keycloakclientscopemappings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=v1.edp.edenlab.io,namespace=keycloak,resources=keycloakclientscopemappings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=v1.edp.edenlab.io,namespace=keycloak,resources=keycloakclientscopemappings/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakClientScopeMapping object.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakClientScopeMapping")

	scopeMapping := &keycloakApi.KeycloakClientScopeMapping{}
	if err := r.client.Get(ctx, request.NamespacedName, scopeMapping); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("unable to get keycloak client scope mapping from k8s: %w", err)
	}

	oldStatus := scopeMapping.Status

	fromClientID, err := r.tryReconcile(ctx, scopeMapping)
	if err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return helper.RequeueOnKeycloakNotAvailable, nil
		}

		scopeMapping.Status.Value = err.Error()

		if statusErr := r.updateKeycloakClientScopeMappingStatus(ctx, scopeMapping, oldStatus); statusErr != nil {
			return reconcile.Result{}, statusErr
		}

		return reconcile.Result{}, err
	}

	scopeMapping.Status.Value = common.StatusOK
	scopeMapping.Status.ID = fromClientID

	if statusErr := r.updateKeycloakClientScopeMappingStatus(ctx, scopeMapping, oldStatus); statusErr != nil {
		return reconcile.Result{}, statusErr
	}

	log.Info("Reconciling KeycloakClientScopeMapping done")

	return reconcile.Result{}, nil
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *keycloakApi.KeycloakClientScopeMapping) (string, error) {
	err := r.helper.SetRealmOwnerRef(ctx, instance)
	if err != nil {
		return "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	cl, err := r.helper.CreateKeycloakClientFromRealmRef(ctx, instance)
	if err != nil {
		// if the realm is already deleted try to delete finalizer
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, instance, finalizerName); removeErr != nil {
				return "", fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return "", nil
		}

		return "", fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, instance, cl)
	if err != nil {
		return "", fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	fromClientID, err := syncClientScopeMapping(ctx, instance, gocloak.PString(realm.Realm), cl)
	if err != nil {
		return "", fmt.Errorf("unable to sync client scope mapping: %w", err)
	}

	if _, err = r.helper.TryToDelete(
		ctx, instance,
		makeTerminator(
			cl,
			gocloak.PString(realm.Realm),
			instance.Status.ID,
			instance.Spec.ClientScope,
			instance.Spec.Client,
			getAdapterRoles(instance),
			objectmeta.PreserveResourcesOnDeletion(instance),
		),
		finalizerName,
	); err != nil {
		return "", fmt.Errorf("unable to delete client scope mapping: %w", err)
	}

	return fromClientID, nil
}

func (r *Reconcile) updateKeycloakClientScopeMappingStatus(
	ctx context.Context,
	scopeMapping *keycloakApi.KeycloakClientScopeMapping,
	oldStatus keycloakApi.KeycloakClientScopeMappingStatus,
) error {
	if scopeMapping.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, scopeMapping); err != nil {
		return fmt.Errorf("failed to update KeycloakClientScopeMapping status: %w", err)
	}

	return nil
}

func syncClientScopeMapping(ctx context.Context,
	instance *keycloakApi.KeycloakClientScopeMapping,
	realmName string,
	cl keycloak.Client,
) (string, error) {
	existingClient, err := cl.GetClient(ctx, realmName, instance.Spec.FromClient)
	if err != nil {
		return "", fmt.Errorf("unable to get client: %w", err)
	}

	fromClientID := gocloak.PString(existingClient.ID)

	if err := cl.SyncRealmClientScopeMapping(
		ctx, fromClientID, realmName, getAdapterRoles(instance),
		adapter.ScopeMappingOptions{
			ClientScope: instance.Spec.ClientScope,
			Client:      instance.Spec.Client,
		},
	); err != nil {
		return "", fmt.Errorf("failed to SyncRealmClientScopeMapping: %w", err)
	}

	instance.Status.ID = fromClientID

	return instance.Status.ID, nil
}

func getAdapterRoles(instance *keycloakApi.KeycloakClientScopeMapping) []adapter.RealmRole {
	adapterRoles := make([]adapter.RealmRole, 0, len(instance.Spec.Roles))
	for _, roleMapping := range instance.Spec.Roles {
		adapterRoles = append(adapterRoles, adapter.RealmRole{
			Name: roleMapping.Name,
		})
	}

	return adapterRoles
}
