package keycloakscopemapping

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Nerzal/gocloak/v12"
	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/objectmeta"
)

const finalizerName = "keycloak.scopemapping.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (keycloak.Client, error)
}

type SyncErr struct {
	Err error
}

func (se *SyncErr) Error() string {
	if se.Err == nil {
		return ""
	}

	return se.Err.Error()
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
		For(&keycloakApiAlpha.KeycloakScopeMapping{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakScopeMapping controller: %w", err)
	}

	return nil
}

func isSpecUpdated(e event.UpdateEvent) bool {
	oo, ok := e.ObjectOld.(*keycloakApiAlpha.KeycloakScopeMapping)
	if !ok {
		return false
	}

	no, ok := e.ObjectNew.(*keycloakApiAlpha.KeycloakScopeMapping)
	if !ok {
		return false
	}

	return !reflect.DeepEqual(oo.Spec, no.Spec) ||
		(oo.GetDeletionTimestamp().IsZero() && !no.GetDeletionTimestamp().IsZero())
}

// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakscopemappings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakscopemappings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakscopemappings/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakScopeMapping object.
func (r *Reconcile) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakScopeMapping")

	scope := &keycloakApiAlpha.KeycloakScopeMapping{}
	if err := r.client.Get(ctx, request.NamespacedName, scope); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("unable to get keycloak scope mapping from k8s: %w", err)
	}

	// Check for paused annotation
	if objectmeta.ReconcilePaused(scope) {
		log.Info("Reconciliation is paused for this resource", "name", "KeycloakScopeMapping")
		return reconcile.Result{}, nil // Stop reconciliation, do not requeue
	}

	oldStatus := scope.Status

	id, err := r.tryReconcile(ctx, scope)
	if err != nil && !isSyncErr(err) {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return helper.RequeueOnKeycloakNotAvailable, nil
		}

		scope.Status.Error = err.Error()
		scope.Status.Phase = common.PhaseFailed

		if statusErr := r.updateKeycloakScopeMappingStatus(ctx, scope, oldStatus); statusErr != nil {
			return reconcile.Result{}, statusErr
		}

		return reconcile.Result{}, err
	}

	scope.Status.Error = ""
	if err != nil && isSyncErr(err) {
		scope.Status.Error = err.Error()
	}

	if id != "" {
		scope.Status.ID = id
	}

	scope.Status.Phase = common.PhaseCompleted

	if statusErr := r.updateKeycloakScopeMappingStatus(ctx, scope, oldStatus); statusErr != nil {
		return reconcile.Result{}, statusErr
	}

	log.Info("Reconciling KeycloakScopeMapping done")

	return reconcile.Result{}, nil
}

func (r *Reconcile) tryReconcile(ctx context.Context, instance *keycloakApiAlpha.KeycloakScopeMapping) (string, error) {
	syncErr := &SyncErr{}

	err := r.helper.SetRealmOwnerRef(ctx, instance)
	if err != nil {
		return "", fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	cl, err := r.helper.CreateKeycloakClientFromConfigRef(ctx, instance)
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

	id, err := r.Sync(ctx, instance, gocloak.PString(realm.Realm), cl)
	if err != nil {
		if !helper.IsNoMatchScopeMapping(err) {
			return "", fmt.Errorf("unable to create keycloak scope mapping: %w", err)
		}

		syncErr.Err = fmt.Errorf("unable to create keycloak scope mapping: %w", err)
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
		return "", fmt.Errorf("unable to delete scope mapping: %w", err)
	}

	return id, syncErr
}

func (r *Reconcile) Sync(ctx context.Context,
	instance *keycloakApiAlpha.KeycloakScopeMapping,
	realmName string,
	cl keycloak.Client) (string, error) {
	spec := instance.Spec

	if (spec.Client == "" || spec.FromClient == "") && spec.ClientScope == "" {
		return "", fmt.Errorf("required fields client or fromClient or clientScope")
	}

	if spec.Client != "" || spec.FromClient != "" {
		return syncClientScopeMapping(ctx, instance, realmName, cl)
	}

	return syncScopeMapping(ctx, instance, realmName, cl)
}

func (r *Reconcile) updateKeycloakScopeMappingStatus(
	ctx context.Context,
	scopeMapping *keycloakApiAlpha.KeycloakScopeMapping,
	oldStatus keycloakApiAlpha.KeycloakScopeMappingStatus,
) error {
	if scopeMapping.Status == oldStatus {
		return nil
	}

	if err := r.client.Status().Update(ctx, scopeMapping); err != nil {
		return fmt.Errorf("failed to update KeycloakScopeMapping status: %w", err)
	}

	return nil
}

func syncScopeMapping(ctx context.Context,
	instance *keycloakApiAlpha.KeycloakScopeMapping,
	realmName string,
	cl keycloak.Client,
) (string, error) {
	clientScope, err := cl.GetClientScope(ctx, instance.Spec.ClientScope, realmName)
	if err != nil && !adapter.IsErrNotFound(err) {
		return "", fmt.Errorf("unable to get client scope: %w", err)
	}

	if clientScope == nil || adapter.IsErrNotFound(err) {
		return "", fmt.Errorf("not fount syncScopeMapping clientScope %s", instance.Spec.ClientScope)
	}

	if err := cl.SyncRealmScopeMapping(ctx, realmName, clientScope.ID, getAdapterRoles(instance)); err != nil {
		return "", fmt.Errorf("failed to SyncRealmScopeMapping: %w", err)
	}

	instance.Status.ID = clientScope.ID

	return instance.Status.ID, nil
}

func syncClientScopeMapping(ctx context.Context,
	instance *keycloakApiAlpha.KeycloakScopeMapping,
	realmName string,
	cl keycloak.Client,
) (string, error) {
	existingClient, err := cl.GetClient(ctx, realmName, instance.Spec.FromClient)
	if err != nil && !adapter.IsErrNotFound(err) {
		return "", fmt.Errorf("unable to get client: %w", err)
	}

	if existingClient == nil || adapter.IsErrNotFound(err) {
		return "", fmt.Errorf("not fount syncClientScopeMapping FromClient %s", instance.Spec.FromClient)
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

func isSyncErr(err error) bool {
	if _, ok := err.(*SyncErr); ok {
		return true
	}

	return false
}

func getAdapterRoles(instance *keycloakApiAlpha.KeycloakScopeMapping) []adapter.RealmRole {
	adapterRoles := make([]adapter.RealmRole, 0, len(instance.Spec.Roles))
	for _, roleMapping := range instance.Spec.Roles {
		adapterRoles = append(adapterRoles, adapter.RealmRole{
			Name: roleMapping.Name,
		})
	}

	return adapterRoles
}
