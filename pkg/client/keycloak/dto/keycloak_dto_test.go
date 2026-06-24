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
