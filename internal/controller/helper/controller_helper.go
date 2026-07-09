package helper

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
	keycloakApiAlpha "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	keycloakclientv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
)

const (
	RequeueOnKeycloakNotAvailablePeriod = time.Minute
)

var RequeueOnKeycloakNotAvailable = ctrl.Result{
	RequeueAfter: RequeueOnKeycloakNotAvailablePeriod,
}

type Terminator interface {
	DeleteResource(ctx context.Context) error
}

type ObjectWithRealmRef interface {
	common.HasRealmRef
	client.Object
}

type ObjectWithKeycloakRef interface {
	common.HasKeycloakRef
	client.Object
}

type ObjectWithConfigRef interface {
	common.HasConfigRef
	client.Object
}

// nolint:logcheck
type adapterBuilder func(
	ctx context.Context,
	conf adapter.GoCloakConfig,
	adminType string,
	log logr.Logger,
	restyClient *resty.Client,
) (keycloak.Client, error)

// ControllerHelper interface defines methods for working with keycloak client and owner references.
type ControllerHelper interface {
	SetRealmOwnerRef(ctx context.Context, object ObjectWithRealmRef) error
	SetFailureCount(fc FailureCountable) time.Duration
	TryToDelete(ctx context.Context, obj client.Object, terminator Terminator, finalizer string) (isDeleted bool, resultErr error)
	TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error
	GetKeycloakRealmFromRef(ctx context.Context, object ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error)
	CreateKeycloakClient(ctx context.Context, url, user, password, adminType, caCert string, insecureSkipVerify bool) (keycloak.Client, error)
	CreateKeycloakClientFomAuthData(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error)
	InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error
	GetRealmNameFromRef(ctx context.Context, object ObjectWithRealmRef) (string, error)
	CreateKeycloakClientFromConfigRef(ctx context.Context, object ObjectWithConfigRef) (keycloak.Client, error)
	CreateKeycloakClientV2FromConfigRef(ctx context.Context, object ObjectWithConfigRef) (*keycloakclientv2.KeycloakClient, error)
}

type Helper struct {
	client          client.Client
	scheme          *runtime.Scheme
	restyClient     *resty.Client
	adapterBuilder  adapterBuilder
	tokenSecretLock *sync.Mutex
	watchNamespace  string
	// enableOwnerRef is a flag to enable legacy owner reference to Keycloak and KeycloakRealm for operator objects.
	// This is needed for backward compatibility with the old version of the operator.
	enableOwnerRef bool
}

func MakeHelper(k8sClient client.Client, scheme *runtime.Scheme, watchNamespace string, options ...func(*Helper)) *Helper {
	helper := &Helper{
		tokenSecretLock: new(sync.Mutex),
		client:          k8sClient,
		scheme:          scheme,
		watchNamespace:  watchNamespace,
		enableOwnerRef:  false,
		adapterBuilder: func(
			ctx context.Context,
			conf adapter.GoCloakConfig,
			adminType string,
			log logr.Logger,
			restyClient *resty.Client,
		) (keycloak.Client, error) {
			if adminType == keycloakApiAlpha.KeycloakAdminTypeServiceAccount {
				goKeycloakAdapter, err := adapter.MakeFromServiceAccount(ctx, conf, "master", log, restyClient)
				if err != nil {
					return nil, fmt.Errorf("failed to make go keycloak adapter from service account: %w", err)
				}

				return goKeycloakAdapter, nil
			}

			goKeycloakAdapter, err := adapter.Make(ctx, conf, log, restyClient)
			if err != nil {
				return nil, fmt.Errorf("failed to make go keycloak adapter: %w", err)
			}

			return goKeycloakAdapter, nil
		},
	}

	for _, option := range options {
		option(helper)
	}

	return helper
}

// EnableOwnerRef is an option to set the enableOwnerRef field in Helper.
func EnableOwnerRef(setOwnerRef bool) func(*Helper) {
	return func(h *Helper) {
		h.enableOwnerRef = setOwnerRef
	}
}

// SetRealmOwnerRef sets owner reference for object.
//
//nolint:dupl,cyclop
func (h *Helper) SetRealmOwnerRef(ctx context.Context, object ObjectWithRealmRef) error {
	if !h.enableOwnerRef {
		return nil
	}

	if metav1.GetControllerOf(object) != nil {
		return nil
	}

	kind := object.GetRealmRef().Kind
	name := object.GetRealmRef().Name

	switch kind {
	case keycloakApiAlpha.KeycloakRealmKind:
		realm := &keycloakApiAlpha.KeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      name,
		}, realm); err != nil {
			return fmt.Errorf("failed to get KeycloakRealm: %w", err)
		}

		if err := controllerutil.SetControllerReference(realm, object, h.scheme); err != nil {
			return fmt.Errorf("failed to set controller reference for %s: %w", object.GetName(), err)
		}

		if err := h.client.Update(ctx, object); err != nil {
			return fmt.Errorf("failed to update realm owner reference %s: %w", realm.GetName(), err)
		}

		return nil

	default:
		return fmt.Errorf("unknown realm kind: %s", kind)
	}
}

func (h *Helper) TryRemoveFinalizer(ctx context.Context, obj client.Object, finalizer string) error {
	if !obj.GetDeletionTimestamp().IsZero() {
		if controllerutil.RemoveFinalizer(obj, finalizer) {
			if err := h.client.Update(ctx, obj); err != nil {
				return fmt.Errorf("unable to update instance: %w", err)
			}
		}
	}

	return nil
}

func (h *Helper) TryToDelete(ctx context.Context, obj client.Object, terminator Terminator, finalizer string) (isDeleted bool, resultErr error) {
	logger := ctrl.LoggerFrom(ctx)

	if obj.GetDeletionTimestamp().IsZero() {
		logger.Info("instance timestamp is zero")

		if controllerutil.AddFinalizer(obj, finalizer) {
			logger.Info("Adding finalizer to instance")

			if err := h.client.Update(ctx, obj); err != nil {
				return false, fmt.Errorf("unable to update deletable object: %w", err)
			}
		}

		logger.Info("processing finalizers done, exit.")

		return false, nil
	}

	logger.Info("terminator deleting resource")

	if err := terminator.DeleteResource(ctx); err != nil {
		return false, fmt.Errorf("error during keycloak resource deletion: %w", err)
	}

	logger.Info("terminator removing finalizers")

	if controllerutil.RemoveFinalizer(obj, finalizer) {
		if err := h.client.Update(ctx, obj); err != nil {
			return false, fmt.Errorf("unable to update instance: %w", err)
		}
	}

	logger.Info("terminator deleting instance done, exit")

	return true, nil
}

func (h *Helper) GetKeycloakRealmFromRef(ctx context.Context, object ObjectWithRealmRef, kcClient keycloak.Client) (*gocloak.RealmRepresentation, error) {
	kind := object.GetRealmRef().Kind
	name := object.GetRealmRef().Name

	if !h.enableOwnerRef {
		kcRealm, err := kcClient.GetRealm(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get realm: [%s] %w", name, err)
		}

		return kcRealm, nil
	}

	switch kind {
	case keycloakApiAlpha.KeycloakRealmKind:
		realm := &keycloakApiAlpha.KeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      name,
		}, realm); err != nil {
			return nil, fmt.Errorf("failed to get KeycloakRealm: %w", err)
		}

		kcRealm, err := kcClient.GetRealm(ctx, realm.Spec.RealmName)
		if err != nil {
			return nil, fmt.Errorf("failed to get realm: %w", err)
		}

		return kcRealm, nil

	default:
		return nil, fmt.Errorf("unknown realm kind: %s", kind)
	}
}

// GetRealmNameFromRef resolves the Keycloak realm name from a RealmRef without calling the Keycloak API.
// It reads the realm name directly from the CR spec.
func (h *Helper) GetRealmNameFromRef(ctx context.Context, object ObjectWithRealmRef) (string, error) {
	kind := object.GetRealmRef().Kind
	name := object.GetRealmRef().Name

	if !h.enableOwnerRef {
		return name, nil
	}

	switch kind {
	case keycloakApiAlpha.KeycloakRealmKind:
		realm := &keycloakApiAlpha.KeycloakRealm{}
		if err := h.client.Get(ctx, types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      name,
		}, realm); err != nil {
			return "", fmt.Errorf("failed to get KeycloakRealm: %w", err)
		}

		return realm.Spec.RealmName, nil

	default:
		return "", fmt.Errorf("unknown realm kind: %s", kind)
	}
}

// RemoveFinalizersOnRealmNotFound removes the given finalizers from obj and persists the update
// when the realm is gone and the object is being deleted.
// Returns true when cleanup was performed and reconciliation should stop.
func RemoveFinalizersOnRealmNotFound(ctx context.Context, k8sClient client.Client, obj client.Object, finalizers ...string) (bool, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Keycloak realm not found, removing finalizers")

	removed := false
	for _, f := range finalizers {
		removed = controllerutil.RemoveFinalizer(obj, f) || removed
	}

	if removed {
		if err := k8sClient.Update(ctx, obj); err != nil {
			return false, fmt.Errorf("failed to remove finalizers: %w", err)
		}
	}

	log.Info("Finalizers removed")

	return true, nil
}

func RemoveSpecialChar(s string) string {
	result := strings.Trim(s, "!@#{$}")
	result = strings.ReplaceAll(result, "_", "-")

	return strings.ToLower(strings.ReplaceAll(result, " ", "-"))
}

func SortRealmGroupByParent(items []keycloakApiAlpha.Group) []keycloakApiAlpha.Group {
	byName := make(map[string]keycloakApiAlpha.Group)
	children := make(map[string][]keycloakApiAlpha.Group)

	for _, item := range items {
		byName[item.Name] = item

		if item.ParentGroup != nil {
			children[item.ParentGroup.Name] = append(children[item.ParentGroup.Name], item)
		}
	}

	var (
		result []keycloakApiAlpha.Group
		visit  func(keycloakApiAlpha.Group)
	)

	visit = func(node keycloakApiAlpha.Group) {
		result = append(result, node)

		for _, child := range children[node.Name] {
			visit(child)
		}
	}

	// start from roots
	for _, item := range items {
		if item.ParentGroup == nil {
			visit(item)
		}
	}

	return result
}

func SortRealmGroupByParentFirstChild(items []keycloakApiAlpha.Group) []keycloakApiAlpha.Group {
	children := make(map[string][]keycloakApiAlpha.Group)

	for _, item := range items {
		if item.ParentGroup != nil {
			children[item.ParentGroup.Name] = append(children[item.ParentGroup.Name], item)
		}
	}

	var (
		result []keycloakApiAlpha.Group
		visit  func(keycloakApiAlpha.Group)
	)

	visit = func(node keycloakApiAlpha.Group) {
		for _, child := range children[node.Name] {
			visit(child)
		}

		// parent goes after all children
		result = append(result, node)
	}

	for _, item := range items {
		if item.ParentGroup == nil {
			visit(item)
		}
	}

	return result
}

func RemoveDuplicates[T comparable](slice []T) []T {
	seen := make(map[T]struct{})
	result := []T{}

	for _, val := range slice {
		if _, exists := seen[val]; !exists {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	return result
}

func RemoveSliceIndex[T any](s []T, i int) []T {
	if i < 0 || i >= len(s) {
		return s
	}

	return append(s[:i], s[i+1:]...)
}
