package dto

import (
	keycloakApiV1Alpha1 "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
)

type Keycloak struct {
	Url  string
	User string
	Pwd  string `json:"-"`
}

type Realm struct {
	Name  string
	Users []User
	ID    *string
}

type User struct {
	Username   string   `json:"username"`
	RealmRoles []string `json:"realmRoles"`
}

type ServerInfo struct {
	SystemInfo SystemInfo      `json:"systemInfo"`
	Features   []ServerFeature `json:"features"`
}

type ServerFeature struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type SystemInfo struct {
	Version string `json:"version"`
}

func ConvertSpecToRealm(spec *keycloakApiV1Alpha1.KeycloakRealmSpec) *Realm {
	users := make([]User, 0, len(spec.Users))
	for _, item := range spec.Users {
		users = append(users, User(item))
	}

	return &Realm{
		Name:  spec.RealmName,
		Users: users,
		ID:    spec.ID,
	}
}

type Client struct {
	ID                                 string
	ClientId                           string
	ClientSecret                       string `json:"-"`
	RealmName                          string
	Roles                              []ClientRole
	PublicClient                       bool
	DirectAccess                       bool
	WebUrl                             string
	AdminUrl                           string
	HomeUrl                            string
	Protocol                           string
	Attributes                         map[string]string
	AdvancedProtocolMappers            bool
	ServiceAccountEnabled              bool
	FrontChannelLogout                 bool
	RedirectUris                       []string
	BaseUrl                            string
	WebOrigins                         []string
	AuthorizationServicesEnabled       bool
	BearerOnly                         bool
	ClientAuthenticatorType            string
	ConsentRequired                    bool
	Description                        string
	Enabled                            bool
	FullScopeAllowed                   bool
	ImplicitFlowEnabled                bool
	Name                               string
	Origin                             string
	RegistrationAccessToken            string
	StandardFlowEnabled                bool
	SurrogateAuthRequired              bool
	AuthenticationFlowBindingOverrides map[string]string
}

type ClientRole struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	AssociatedClientRoles []string `json:"associatedClientRoles"`
}

type PrimaryRealmRole struct {
	ID                    *string
	Name                  string
	Composites            []string
	CompositesClientRoles map[string][]string
	IsComposite           bool
	Description           string
	Attributes            map[string][]string
	IsDefault             bool
}

type IncludedRealmRole struct {
	Name      string
	Composite string
}

type IdentityProviderMapper struct {
	IdentityProviderMapper string            `json:"identityProviderMapper"`
	IdentityProviderAlias  string            `json:"identityProviderAlias,omitempty"`
	Name                   string            `json:"name"`
	Config                 map[string]string `json:"config"`
	ID                     string            `json:"id"`
}

// Organization represents a Keycloak Organization.
type Organization struct {
	ID          string               `json:"id,omitempty"`
	Name        string               `json:"name"`
	Alias       string               `json:"alias"`
	Description string               `json:"description,omitempty"`
	RedirectURL string               `json:"redirectUrl,omitempty"`
	Attributes  map[string][]string  `json:"attributes,omitempty"`
	Domains     []OrganizationDomain `json:"domains,omitempty"`
}

// OrganizationDomain represents a domain within an Organization.
type OrganizationDomain struct {
	Name string `json:"name"`
}

// OrganizationIdentityProvider represents the link between an Organization and Identity Provider.
type OrganizationIdentityProvider struct {
	Alias string `json:"alias"`
}

// ConvertSpecToOrganization converts a KeycloakOrganization spec to an Organization.
func ConvertSpecToOrganization(org *keycloakApiV1Alpha1.KeycloakOrganization) *Organization {
	orgAdapter := &Organization{
		Name:        org.Spec.Name,
		Alias:       org.Spec.Alias,
		Description: org.Spec.Description,
		RedirectURL: org.Spec.RedirectURL,
		Attributes:  org.Spec.Attributes,
	}

	// Convert domains to OrganizationDomain format
	for _, domain := range org.Spec.Domains {
		orgAdapter.Domains = append(orgAdapter.Domains, OrganizationDomain{
			Name: domain,
		})
	}

	// Set ID from status if available
	if org.Status.OrganizationID != "" {
		orgAdapter.ID = org.Status.OrganizationID
	}

	return orgAdapter
}
