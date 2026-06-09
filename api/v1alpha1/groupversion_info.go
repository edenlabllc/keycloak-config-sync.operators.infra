// NOTE: Boilerplate only.  Ignore this file.

// Package v1alpha1 contains API Schema definitions for the v1 v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=config.idp.edenlab.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "config.idp.edenlab.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	AddToScheme = SchemeBuilder.AddToScheme
)

const (
	// TODO: old need remove
	ClusterKeycloakKind      = "ClusterKeycloak"
	ClusterKeycloakRealmKind = "ClusterKeycloakRealm"

	// KeycloakRealmKind is a string value of the kind of KeycloakClient CR.
	KeycloakRealmKind = "KeycloakRealm"
	// KeycloakRealmComponentKind is a string value of the kind of KeycloakClient CR.
	KeycloakRealmComponentKind = "KeycloakRealmComponent"
	KeycloakKind               = "Keycloak"
)
