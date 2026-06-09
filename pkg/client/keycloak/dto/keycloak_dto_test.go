package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	keycloakApiV1Alpha1 "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
)

func TestConvertSpecToRealm(t *testing.T) {
	tests := []struct {
		name     string
		input    *keycloakApiV1Alpha1.KeycloakRealmSpec
		expected *Realm
	}{
		{
			name: "basic realm conversion",
			input: &keycloakApiV1Alpha1.KeycloakRealmSpec{
				RealmName: "test-realm",
				ID:        ptr.To("realm-123"),
			},
			expected: &Realm{
				Name:  "test-realm",
				ID:    ptr.To("realm-123"),
				Users: []User{},
			},
		},
		{
			name: "realm with users",
			input: &keycloakApiV1Alpha1.KeycloakRealmSpec{
				RealmName: "realm-with-users",
				Users: []keycloakApiV1Alpha1.User{
					{
						Username:   "user1",
						RealmRoles: []string{"role1", "role2"},
					},
					{
						Username:   "user2",
						RealmRoles: []string{"role3"},
					},
				},
			},
			expected: &Realm{
				Name: "realm-with-users",
				Users: []User{
					{
						Username:   "user1",
						RealmRoles: []string{"role1", "role2"},
					},
					{
						Username:   "user2",
						RealmRoles: []string{"role3"},
					},
				},
			},
		},
		{
			name: "realm with empty users slice",
			input: &keycloakApiV1Alpha1.KeycloakRealmSpec{
				RealmName: "empty-users-realm",
				Users:     []keycloakApiV1Alpha1.User{},
			},
			expected: &Realm{
				Name:  "empty-users-realm",
				Users: []User{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSpecToRealm(tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertSpecToOrganization(t *testing.T) {
	tests := []struct {
		name     string
		input    *keycloakApiV1Alpha1.KeycloakOrganization
		expected *Organization
	}{
		{
			name: "basic organization conversion",
			input: &keycloakApiV1Alpha1.KeycloakOrganization{
				Spec: keycloakApiV1Alpha1.KeycloakOrganizationSpec{
					Name:  "test-org",
					Alias: "test-alias",
				},
			},
			expected: &Organization{
				Name:    "test-org",
				Alias:   "test-alias",
				Domains: nil,
			},
		},
		{
			name: "organization with all fields",
			input: &keycloakApiV1Alpha1.KeycloakOrganization{
				Spec: keycloakApiV1Alpha1.KeycloakOrganizationSpec{
					Name:        "full-org",
					Alias:       "full-alias",
					Description: "Full organization description",
					RedirectURL: "https://redirect.example.com",
					Domains:     []string{"example.com", "test.com"},
					Attributes: map[string][]string{
						"attr1": {"value1", "value2"},
						"attr2": {"value3"},
					},
				},
				Status: keycloakApiV1Alpha1.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			expected: &Organization{
				ID:          "org-123",
				Name:        "full-org",
				Alias:       "full-alias",
				Description: "Full organization description",
				RedirectURL: "https://redirect.example.com",
				Attributes: map[string][]string{
					"attr1": {"value1", "value2"},
					"attr2": {"value3"},
				},
				Domains: []OrganizationDomain{
					{Name: "example.com"},
					{Name: "test.com"},
				},
			},
		},
		{
			name: "organization with empty domains",
			input: &keycloakApiV1Alpha1.KeycloakOrganization{
				Spec: keycloakApiV1Alpha1.KeycloakOrganizationSpec{
					Name:    "empty-domains-org",
					Alias:   "empty-alias",
					Domains: []string{},
				},
			},
			expected: &Organization{
				Name:    "empty-domains-org",
				Alias:   "empty-alias",
				Domains: nil,
			},
		},
		{
			name: "organization without status ID",
			input: &keycloakApiV1Alpha1.KeycloakOrganization{
				Spec: keycloakApiV1Alpha1.KeycloakOrganizationSpec{
					Name:  "no-id-org",
					Alias: "no-id-alias",
				},
				Status: keycloakApiV1Alpha1.KeycloakOrganizationStatus{
					OrganizationID: "", // empty ID
				},
			},
			expected: &Organization{
				Name:    "no-id-org",
				Alias:   "no-id-alias",
				Domains: nil,
				// ID should not be set when OrganizationID is empty
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSpecToOrganization(tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}
