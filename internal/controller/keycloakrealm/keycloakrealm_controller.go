package keycloakrealm

import (
	"context"
	"errors"
	"fmt"
	"time"

	groupChan "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/group"
	idpChan "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/identityprovider"
	realmChan "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/realm"
	realmChanHandler "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/realm/handler"
	roleChan "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakrealm/chain/role"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/objectmeta"
)

const keyCloakRealmOperatorFinalizerName = "keycloak.realm.operator.finalizer.name"

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	CreateKeycloakClientV2FromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (*keycloakv2.KeycloakClient, error)
	CreateKeycloakClientFromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (keycloak.Client, error)
}

func NewReconcileKeycloakRealm(
	k8sClient client.Client,
	scheme *runtime.Scheme,
	controllerHelper Helper,
) *ReconcileKeycloakRealm {
	return &ReconcileKeycloakRealm{
		client:    k8sClient,
		helper:    controllerHelper,
		realmChan: realmChan.CreateDefChain(k8sClient, scheme),
	}
}

// ReconcileKeycloakRealm reconciles a KeycloakRealm object.
type ReconcileKeycloakRealm struct {
	client                  client.Client
	helper                  Helper
	realmChan               realmChanHandler.RealmHandler
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakRealm) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout
	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApiAlpha.KeycloakRealm{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakRealm controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakrealms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakrealms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakrealms/finalizers,verbs=update
// +kubebuilder:rbac:groups="",namespace=keycloak,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is a loop for reconciling KeycloakRealm object.
func (r *ReconcileKeycloakRealm) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakRealm")

	instance := &keycloakApiAlpha.KeycloakRealm{}
	if err := r.client.Get(ctx, request.NamespacedName, instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return result, resultErr
		}

		resultErr = err

		return result, resultErr
	}

	if err := r.tryReconcile(ctx, instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		instance.Status.Available = false
		instance.Status.Error = err.Error()
		instance.Status.Phase = common.PhaseFailed
		result.RequeueAfter = r.helper.SetFailureCount(instance)

		log.Error(err, "an error has occurred while handling keycloak realm", "name", request.Name)
	} else {
		instance.Status.Available = true
		instance.Status.Error = ""
		instance.Status.Phase = common.PhaseCompleted
		instance.Status.FailureCount = 0
		result.RequeueAfter = r.successReconcileTimeout
	}

	if err := r.client.Status().Update(ctx, instance); err != nil {
		resultErr = fmt.Errorf("unable to update status: %w", err)
	}

	return result, resultErr
}

func (r *ReconcileKeycloakRealm) tryReconcile(ctx context.Context, realm *keycloakApiAlpha.KeycloakRealm) error {
	kClientV2, err := r.helper.CreateKeycloakClientV2FromConfigRef(ctx, realm)
	if err != nil {
		return fmt.Errorf("failed to create keycloak v2 client for realm: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromConfigRef(ctx, realm)
	if err != nil {
		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	deleted, err := r.helper.TryToDelete(
		ctx,
		realm,
		makeTerminator(realm.Spec, kClientV2, objectmeta.PreserveResourcesOnDeletion(realm)),
		keyCloakRealmOperatorFinalizerName,
	)
	if err != nil {
		return fmt.Errorf("failed to delete realm: %w", err)
	}

	if deleted {
		return nil
	}

	if err = r.realmChan.ServeRequest(ctx, realm, kClientV2); err != nil {
		return fmt.Errorf("error during realm chain: %w", err)
	}

	if err = roleChan.MakeChain(kClientV2, r.client).Serve(ctx, realm); err != nil {
		return fmt.Errorf("error during realm role chain: %w", err)
	}

	if err = groupChan.MakeChain().Serve(ctx, realm, kClientV2, r.client); err != nil {
		return fmt.Errorf("error during realm group chain: %w", err)
	}

	// TODO: need fix, use only kClientV2 and remove kClient
	if err = idpChan.MakeChain(kClient, r.client).Serve(ctx, realm); err != nil {
		return fmt.Errorf("unable to serve keycloak realm idp: %w", err)
	}

	return nil
}
