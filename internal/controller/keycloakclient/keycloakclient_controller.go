package keycloakclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v12"
	chainClient "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain/client"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	chainScope "github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain/scope"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/objectmeta"
)

type Helper interface {
	SetFailureCount(fc helper.FailureCountable) time.Duration
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	TryToDelete(ctx context.Context, obj client.Object, terminator helper.Terminator, finalizer string) (isDeleted bool, resultErr error)
	SetRealmOwnerRef(ctx context.Context, object helper.ObjectWithRealmRef) error
	GetKeycloakRealmFromRef(ctx context.Context, object helper.ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClientFromConfigRef(ctx context.Context, object helper.ObjectWithConfigRef) (keycloak.Client, error)
}

const (
	keyCloakClientOperatorFinalizerName = "keycloak.client.operator.finalizer.name"
)

func NewReconcileKeycloakClient(k8sClient client.Client, controllerHelper Helper) *ReconcileKeycloakClientSettings {
	return &ReconcileKeycloakClientSettings{
		client: k8sClient,
		helper: controllerHelper,
	}
}

// ReconcileKeycloakClientSettings reconciles a KeycloakClientSettings object.
type ReconcileKeycloakClientSettings struct {
	client                  client.Client
	helper                  Helper
	successReconcileTimeout time.Duration
}

func (r *ReconcileKeycloakClientSettings) SetupWithManager(mgr ctrl.Manager, successReconcileTimeout time.Duration) error {
	r.successReconcileTimeout = successReconcileTimeout

	pred := predicate.Funcs{
		UpdateFunc: helper.IsFailuresUpdated,
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&keycloakApi.KeycloakClient{}, builder.WithPredicates(pred)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to setup KeycloakClientSettings controller: %w", err)
	}

	return nil
}

// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=config.idp.edenlab.io,namespace=keycloak,resources=keycloakclients/finalizers,verbs=update

// Reconcile is a loop for reconciling KeycloakClientSettings object.
func (r *ReconcileKeycloakClientSettings) Reconcile(ctx context.Context, request reconcile.Request) (result reconcile.Result, resultErr error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Reconciling KeycloakClientSettings")

	var instance keycloakApi.KeycloakClient
	if err := r.client.Get(ctx, request.NamespacedName, &instance); err != nil {
		if k8sErrors.IsNotFound(err) {
			return result, resultErr
		}

		resultErr = err

		return result, resultErr
	}

	if err := r.tryReconcile(ctx, &instance); err != nil {
		if errors.Is(err, helper.ErrKeycloakIsNotAvailable) {
			return ctrl.Result{
				RequeueAfter: helper.RequeueOnKeycloakNotAvailablePeriod,
			}, nil
		}

		// Set Ready condition to False
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               chain.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             chain.ReasonKeycloakAPIError,
			Message:            fmt.Sprintf("Reconciliation failed: %s", err.Error()),
			ObservedGeneration: instance.Generation,
		})

		// Backward compatibility: set Value field
		instance.Status.Value = err.Error()
		result.RequeueAfter = r.helper.SetFailureCount(&instance)

		log.Error(err, "an error has occurred while handling keycloak client settings", "name", request.Name)
	} else {
		// Set Ready condition to True
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:               chain.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             chain.ReasonReconciliationSucceeded,
			Message:            "KeycloakClientSettings reconciliation completed successfully",
			ObservedGeneration: instance.Generation,
		})

		// Backward compatibility: set Value field
		helper.SetSuccessStatus(&instance)

		result.RequeueAfter = r.successReconcileTimeout
	}

	// Final status update for Ready condition and Value field
	if err := r.client.Status().Update(ctx, &instance); err != nil {
		resultErr = fmt.Errorf("unable to update status: %w", err)
	}

	return result, resultErr
}

func (r *ReconcileKeycloakClientSettings) tryReconcile(ctx context.Context, keycloakClientSettings *keycloakApi.KeycloakClient) error {
	err := r.helper.SetRealmOwnerRef(ctx, keycloakClientSettings)
	if err != nil {
		return fmt.Errorf("unable to set realm owner ref: %w", err)
	}

	kClient, err := r.helper.CreateKeycloakClientFromConfigRef(ctx, keycloakClientSettings)
	if err != nil {
		// if the realm is already deleted try to delete finalizer
		if errors.Is(err, helper.ErrKeycloakRealmNotFound) {
			if removeErr := r.helper.TryRemoveFinalizer(ctx, keycloakClientSettings, keyCloakClientOperatorFinalizerName); removeErr != nil {
				return fmt.Errorf("unable to remove finalizer: %w", removeErr)
			}

			return nil
		}

		return fmt.Errorf("unable to create keycloak client from realm ref: %w", err)
	}

	realm, err := r.getKeycloakRealm(ctx, keycloakClientSettings, kClient)
	if err != nil {
		return fmt.Errorf("unable to get keycloak realm: %w", err)
	}

	deleted, err := r.helper.TryToDelete(
		ctx,
		keycloakClientSettings,
		makeTerminator(DataTerminator{
			ClientIDs:      keycloakClientSettings.Status.ClientIDs,
			ClientScopeIDs: keycloakClientSettings.Status.ClientScopeIDs,
		}, realm, kClient, objectmeta.PreserveResourcesOnDeletion(keycloakClientSettings)),
		keyCloakClientOperatorFinalizerName,
	)
	if err != nil {
		return fmt.Errorf("deleting keycloak client settings: %w", err)
	}

	if deleted {
		return nil
	}

	if err = r.flushResources(
		ctx,
		keycloakClientSettings,
		kClient,
		realm,
		objectmeta.PreserveResourcesOnDeletion(keycloakClientSettings),
	); err != nil {
		return fmt.Errorf("unable to remove finalizer: %w", err)
	}

	if err = chainScope.MakeChain(kClient, r.client).Serve(ctx, keycloakClientSettings, realm); err != nil {
		return fmt.Errorf("unable to serve keycloak client scope: %w", err)
	}

	if err = chainClient.MakeChain(kClient, r.client).Serve(ctx, keycloakClientSettings, realm); err != nil {
		return fmt.Errorf("unable to serve keycloak client: %w", err)
	}

	return nil
}

func (r *ReconcileKeycloakClientSettings) getKeycloakRealm(
	ctx context.Context,
	keycloakClient *keycloakApi.KeycloakClient,
	adapterClient keycloak.Client,
) (string, error) {
	realm, err := r.helper.GetKeycloakRealmFromRef(ctx, keycloakClient, adapterClient)
	if err != nil {
		return "", fmt.Errorf("unable to get keycloak realm from ref: %w", err)
	}

	return gocloak.PString(realm.Realm), nil
}

func (r *ReconcileKeycloakClientSettings) flushResources(ctx context.Context,
	keycloakClient *keycloakApi.KeycloakClient,
	adapterClient keycloak.Client,
	realmName string,
	preserveResourcesOnDeletion bool) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start deleting not actual resources")

	spec := keycloakClient.Spec

	for name, clientScopeID := range keycloakClient.Status.ClientScopeIDs {
		isClientScopeDelete := false

		if !hasExist(name, convertToName(spec.ClientScope)) {
			log.Info("Start deleting keycloak clientScope",
				"name", name, "clientScopeID", clientScopeID)

			if !preserveResourcesOnDeletion {
				err := adapterClient.DeleteClientScope(ctx, realmName, clientScopeID)
				if err != nil && !adapter.IsErrNotFound(err) {
					return fmt.Errorf("[%s] %s clientScope err: %w", name, clientScopeID, err)
				}

				isClientScopeDelete = true
			} else {
				log.Info("PreserveResourcesOnDeletion is enabled, skipping clientScope deletion.",
					"name", name, "clientScopeID", clientScopeID)
			}

			delete(keycloakClient.Status.ClientScopeIDs, name)

			if isClientScopeDelete {
				log.Info("Keycloak clientScope has been deleted",
					"name", name, "clientScopeID", clientScopeID)
			}
		}
	}

	for name, clientID := range keycloakClient.Status.ClientIDs {
		isClientDelete := false

		if !hasExist(name, convertToName(spec.Client)) {
			log.Info("Start deleting keycloak client",
				"name", name, "clientID", clientID)

			if !preserveResourcesOnDeletion {
				findClientID, _ := adapterClient.GetClientID(clientID, realmName)
				if findClientID != "" {
					err := adapterClient.DeleteClient(ctx, clientID, realmName)
					if err != nil && !adapter.IsErrNotFound(err) {
						return fmt.Errorf("[%s] %s client err: %w", name, clientID, err)
					}
				}
			} else {
				log.Info("PreserveResourcesOnDeletion is enabled, skipping client deletion.",
					"name", name, "clientID", clientID)
			}

			delete(keycloakClient.Status.ClientIDs, name)

			if isClientDelete {
				log.Info("Keycloak client has been deleted",
					"name", name, "clientID", clientID)
			}
		}
	}

	return nil
}

func convertToName(obj any) []string {
	var result []string

	if obj == nil {
		return result
	}

	switch v := obj.(type) {
	case []keycloakApi.Client:
		for _, clientItem := range v {
			result = append(result, helper.RemoveSpecialChar(clientItem.Name))
		}
	case *[]keycloakApi.ClientScope:
		for _, clientScope := range *v {
			result = append(result, helper.RemoveSpecialChar(clientScope.Name))
		}
	}

	return result
}

func hasExist(name string, data []string) bool {
	for _, actualName := range data {
		if name == actualName {
			return true
		}
	}

	return false
}
