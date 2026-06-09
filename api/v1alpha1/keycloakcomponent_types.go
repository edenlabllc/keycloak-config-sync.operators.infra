package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
)

// KeycloakComponentSpec defines the desired state of KeycloakComponent.
type KeycloakComponentSpec struct {
	// Name of keycloak component.
	Name string `json:"name"`

	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// ConfigRef is reference to Keycloak Config.
	// +required
	ConfigRef common.ConfigRef `json:"configRef"`

	// ProviderID is a provider ID of component.
	ProviderID string `json:"providerId"`

	// ProviderType is a provider type of component.
	ProviderType string `json:"providerType"`

	// SubType is a sub Type of component.
	// +optional
	SubType string `json:"subType"`

	// ParentRef specifies a parent resource.
	// If not specified, then parent is realm specified in realm field.
	// +nullable
	// +optional
	ParentRef *ParentComponent `json:"parentRef,omitempty"`

	// Config is a map of component configuration.
	// Map key is a name of configuration property, map value is an array value of configuration properties.
	// Any configuration property can be a reference to k8s secret, in this case the property should be in format $secretName:secretKey.
	// +kubebuilder:example={"bindDn": ["provider-client"], "bindCredential": ["$clientSecret:secretKey"]}
	// +nullable
	// +optional
	Config map[string][]string `json:"config,omitempty"`

	// ReconciliationStrategy is a strategy to reconcile client.
	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`
}

// KeycloakComponentStatus defines the observed state of KeycloakComponent.
type KeycloakComponentStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// ParentComponent defines the parent component of KeycloakComponent.
type ParentComponent struct {
	// Kind is a kind of parent component. By default, it is KeycloakRealm.
	// +optional
	// +kubebuilder:default=KeycloakRealm
	// +kubebuilder:validation:Enum=KeycloakRealm;KeycloakComponent
	Kind string `json:"kind,omitempty"`

	// Name is a name of parent component custom resource.
	// For example, if Kind is KeycloakRealm, then Name is name of KeycloakRealm custom resource.
	Name string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconciliation status"

// KeycloakComponent is the Schema for the keycloak component API.
type KeycloakComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakComponentSpec   `json:"spec,omitempty"`
	Status KeycloakComponentStatus `json:"status,omitempty"`
}

func (in *KeycloakComponent) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakComponent) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakComponent) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakComponent) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakComponent) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

func (in *KeycloakComponent) GetConfigRef() common.ConfigRef {
	return in.Spec.ConfigRef
}

// +kubebuilder:object:root=true

// KeycloakComponentList contains a list of KeycloakComponent.
type KeycloakComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakComponent{}, &KeycloakComponentList{})
}
