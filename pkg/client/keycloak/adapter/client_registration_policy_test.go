package adapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

const allowDefaultScopesKey = "allow-default-scopes"

func allowedClientScopesPolicy(subType string, scopes []string) Component {
	return Component{
		ID:           "policy-" + subType,
		Name:         DefaultAllowedClientScopesPolicyName,
		ProviderID:   AllowedClientScopesPolicyProviderID,
		ProviderType: ClientRegistrationPolicyProviderType,
		SubType:      subType,
		Config: map[string][]string{
			allowDefaultScopesKey:               {"true"},
			AllowedClientScopesPolicyProviderID: scopes,
		},
	}
}

func allowedClientScopesConfig(scopes ...string) map[string][]string {
	return map[string][]string{AllowedClientScopesPolicyProviderID: scopes}
}

func TestGoCloakAdapter_GetComponents(t *testing.T) {
	tests := []struct {
		name           string
		params         *GetComponentsParams
		statusCode     int
		components     []Component
		body           string
		wantQuery      map[string]string
		wantComponents int
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "success without params",
			params:         nil,
			statusCode:     http.StatusOK,
			components:     []Component{allowedClientScopesPolicy("anonymous", nil)},
			wantQuery:      map[string]string{},
			wantComponents: 1,
		},
		{
			name: "success with provider type and name filters",
			params: &GetComponentsParams{
				ProviderType: ptr.To(ClientRegistrationPolicyProviderType),
				Name:         ptr.To(DefaultAllowedClientScopesPolicyName),
			},
			statusCode: http.StatusOK,
			components: []Component{
				allowedClientScopesPolicy("anonymous", nil),
				allowedClientScopesPolicy("authenticated", nil),
			},
			wantQuery: map[string]string{
				keycloakApiParamComponentType: ClientRegistrationPolicyProviderType,
				keycloakApiParamComponentName: DefaultAllowedClientScopesPolicyName,
			},
			wantComponents: 2,
		},
		{
			name:       "empty filters are not sent as query params",
			params:     &GetComponentsParams{ProviderType: ptr.To(""), Name: ptr.To("")},
			statusCode: http.StatusOK,
			components: []Component{},
			wantQuery:  map[string]string{},
		},
		{
			name:       "api error",
			params:     nil,
			statusCode: http.StatusInternalServerError,
			body:       "fatal",
			wantQuery:  map[string]string{},
			wantErr:    true,
			errMsg:     "error during get components request: status: 500 Internal Server Error, body: fatal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := strings.Replace(realmComponent, "{realm}", "realm-name", 1)
				if r.Method != http.MethodGet || r.URL.Path != expectedPath {
					w.WriteHeader(http.StatusNotFound)

					return
				}

				for key, want := range tt.wantQuery {
					require.Equal(t, want, r.URL.Query().Get(key))
				}

				require.Len(t, r.URL.Query(), len(tt.wantQuery))

				if tt.body != "" {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.body))

					return
				}

				setJSONContentType(w)
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.components)
			}))
			defer server.Close()

			kcAdapter, _, _ := initAdapter(t, server)

			components, err := kcAdapter.GetComponents(context.Background(), "realm-name", tt.params)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.errMsg, err.Error())

				return
			}

			require.NoError(t, err)
			require.Len(t, components, tt.wantComponents)
		})
	}
}

func TestGoCloakAdapter_AddClientRegistrationPolicyConfig(t *testing.T) {
	tests := []struct {
		name        string
		policy      *ClientRegistrationPolicy
		components  []Component
		getStatus   int
		writeStatus int
		// wantUpdate is the whole config expected in a PUT, keyed by component id.
		wantUpdate map[string]map[string][]string
		// wantCreate is the component expected in a POST, nil when no component must be created.
		wantCreate *Component
		wantErr    bool
		errMsg     string
	}{
		{
			name: "values are added to the authenticated policy",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config:     allowedClientScopesConfig("new-scope"),
			},
			components: []Component{
				allowedClientScopesPolicy("anonymous", []string{"existing-scope"}),
				allowedClientScopesPolicy("authenticated", []string{"existing-scope"}),
			},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusNoContent,
			wantUpdate: map[string]map[string][]string{
				"policy-authenticated": {
					allowDefaultScopesKey:               {"true"},
					AllowedClientScopesPolicyProviderID: {"existing-scope", "new-scope"},
				},
			},
		},
		{
			name: "the anonymous policy is addressed separately",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAnonymous,
				Config:     allowedClientScopesConfig("new-scope"),
			},
			components: []Component{
				allowedClientScopesPolicy("anonymous", nil),
				allowedClientScopesPolicy("authenticated", nil),
			},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusNoContent,
			wantUpdate: map[string]map[string][]string{
				"policy-anonymous": {
					allowDefaultScopesKey:               {"true"},
					AllowedClientScopesPolicyProviderID: {"new-scope"},
				},
			},
		},
		{
			name: "several config keys are merged at once",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config: map[string][]string{
					AllowedClientScopesPolicyProviderID: {"new-scope"},
					"extra-key":                         {"extra-value"},
				},
			},
			components: []Component{
				allowedClientScopesPolicy("authenticated", []string{"existing-scope"}),
			},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusNoContent,
			wantUpdate: map[string]map[string][]string{
				"policy-authenticated": {
					allowDefaultScopesKey:               {"true"},
					AllowedClientScopesPolicyProviderID: {"existing-scope", "new-scope"},
					"extra-key":                         {"extra-value"},
				},
			},
		},
		{
			name: "config keys managed outside of the operator are preserved",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config:     allowedClientScopesConfig("new-scope"),
			},
			components: []Component{
				{
					ID:           "policy-authenticated",
					Name:         DefaultAllowedClientScopesPolicyName,
					ProviderID:   AllowedClientScopesPolicyProviderID,
					ProviderType: ClientRegistrationPolicyProviderType,
					SubType:      AllowedClientScopesPolicySubTypeAuthenticated,
					Config: map[string][]string{
						"foreign-key": {"foreign-value"},
					},
				},
			},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusNoContent,
			wantUpdate: map[string]map[string][]string{
				"policy-authenticated": {
					"foreign-key":                       {"foreign-value"},
					AllowedClientScopesPolicyProviderID: {"new-scope"},
				},
			},
		},
		{
			name: "the policy is created when the realm does not have it",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config:     allowedClientScopesConfig("scope-a", "scope-b"),
			},
			components:  []Component{},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusCreated,
			wantUpdate:  map[string]map[string][]string{},
			wantCreate: &Component{
				Name:         DefaultAllowedClientScopesPolicyName,
				ProviderID:   AllowedClientScopesPolicyProviderID,
				ProviderType: ClientRegistrationPolicyProviderType,
				SubType:      AllowedClientScopesPolicySubTypeAuthenticated,
				Config:       allowedClientScopesConfig("scope-a", "scope-b"),
			},
		},
		{
			name: "the policy is created with the configured name and the whole config",
			policy: &ClientRegistrationPolicy{
				Name:       "Custom Policy Name",
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAnonymous,
				Config: map[string][]string{
					AllowedClientScopesPolicyProviderID: {"scope-a"},
					allowDefaultScopesKey:               {"true"},
				},
			},
			components:  []Component{},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusCreated,
			wantUpdate:  map[string]map[string][]string{},
			wantCreate: &Component{
				Name:         "Custom Policy Name",
				ProviderID:   AllowedClientScopesPolicyProviderID,
				ProviderType: ClientRegistrationPolicyProviderType,
				SubType:      AllowedClientScopesPolicySubTypeAnonymous,
				Config: map[string][]string{
					AllowedClientScopesPolicyProviderID: {"scope-a"},
					allowDefaultScopesKey:               {"true"},
				},
			},
		},
		{
			name: "the policy is created when only the other subType exists",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config:     allowedClientScopesConfig("scope-a"),
			},
			components: []Component{
				allowedClientScopesPolicy("anonymous", []string{"scope-a"}),
			},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusCreated,
			wantUpdate:  map[string]map[string][]string{},
			wantCreate: &Component{
				Name:         DefaultAllowedClientScopesPolicyName,
				ProviderID:   AllowedClientScopesPolicyProviderID,
				ProviderType: ClientRegistrationPolicyProviderType,
				SubType:      AllowedClientScopesPolicySubTypeAuthenticated,
				Config:       allowedClientScopesConfig("scope-a"),
			},
		},
		{
			name: "no write is sent when every value is already present",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config: map[string][]string{
					AllowedClientScopesPolicyProviderID: {"existing-scope"},
					allowDefaultScopesKey:               {"true"},
				},
			},
			components: []Component{
				allowedClientScopesPolicy("authenticated", []string{"existing-scope"}),
			},
			getStatus:  http.StatusOK,
			wantUpdate: map[string]map[string][]string{},
		},
		{
			name: "another providerId addresses its own component",
			policy: &ClientRegistrationPolicy{
				ProviderID: "allowed-protocol-mappers",
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config: map[string][]string{
					"allowed-protocol-mapper-types": {"oidc-full-name-mapper"},
				},
			},
			components: []Component{
				{
					ID:           "protocol-mappers",
					Name:         "Allowed Protocol Mapper Types",
					ProviderID:   "allowed-protocol-mappers",
					ProviderType: ClientRegistrationPolicyProviderType,
					SubType:      AllowedClientScopesPolicySubTypeAuthenticated,
				},
				allowedClientScopesPolicy("authenticated", nil),
			},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusNoContent,
			wantUpdate: map[string]map[string][]string{
				"protocol-mappers": {
					"allowed-protocol-mapper-types": {"oidc-full-name-mapper"},
				},
			},
		},
		{
			name: "keycloak is not called for an empty config",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
			},
			getStatus:  http.StatusOK,
			wantUpdate: map[string]map[string][]string{},
		},
		{
			name: "error when components cannot be fetched",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config:     allowedClientScopesConfig("scope-a"),
			},
			getStatus:  http.StatusInternalServerError,
			wantUpdate: map[string]map[string][]string{},
			wantErr:    true,
			errMsg: "unable to get client registration policies: error during get components request: " +
				"status: 500 Internal Server Error, body: fatal",
		},
		{
			name: "error when the policy update fails",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config:     allowedClientScopesConfig("new-scope"),
			},
			components: []Component{
				allowedClientScopesPolicy("authenticated", nil),
			},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusBadRequest,
			wantUpdate: map[string]map[string][]string{
				"policy-authenticated": {
					allowDefaultScopesKey:               {"true"},
					AllowedClientScopesPolicyProviderID: {"new-scope"},
				},
			},
			wantErr: true,
			errMsg: "unable to update client registration policy allowed-client-scopes/authenticated: " +
				"error during update component request: status: 400 Bad Request, body: ",
		},
		{
			name: "error when the policy creation fails",
			policy: &ClientRegistrationPolicy{
				ProviderID: AllowedClientScopesPolicyProviderID,
				SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
				Config:     allowedClientScopesConfig("new-scope"),
			},
			components:  []Component{},
			getStatus:   http.StatusOK,
			writeStatus: http.StatusForbidden,
			wantUpdate:  map[string]map[string][]string{},
			wantCreate: &Component{
				Name:         DefaultAllowedClientScopesPolicyName,
				ProviderID:   AllowedClientScopesPolicyProviderID,
				ProviderType: ClientRegistrationPolicyProviderType,
				SubType:      AllowedClientScopesPolicySubTypeAuthenticated,
				Config:       allowedClientScopesConfig("new-scope"),
			},
			wantErr: true,
			errMsg: "unable to create client registration policy allowed-client-scopes/authenticated: " +
				"error during request: status: 403 Forbidden, body: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUpdate := make(map[string]map[string][]string)

			var gotCreate *Component

			getPath := strings.Replace(realmComponent, "{realm}", "realm-name", 1)
			updatePathPrefix := getPath + "/"

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == getPath:
					if tt.getStatus != http.StatusOK {
						w.WriteHeader(tt.getStatus)
						_, _ = w.Write([]byte("fatal"))

						return
					}

					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(tt.components)

				case r.Method == http.MethodPost && r.URL.Path == getPath:
					gotCreate = decodeComponent(t, r.Body)

					w.WriteHeader(tt.writeStatus)

				case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, updatePathPrefix):
					component := decodeComponent(t, r.Body)
					gotUpdate[strings.TrimPrefix(r.URL.Path, updatePathPrefix)] = component.Config

					w.WriteHeader(tt.writeStatus)

				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			kcAdapter, _, _ := initAdapter(t, server)

			err := kcAdapter.AddClientRegistrationPolicyConfig(context.Background(), "realm-name", tt.policy)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, tt.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.wantUpdate, gotUpdate)

			if tt.wantCreate == nil {
				require.Nil(t, gotCreate, "no component must be created")
			} else {
				require.Equal(t, tt.wantCreate, gotCreate)
			}
		})
	}
}

func TestGoCloakAdapter_AddClientRegistrationPolicyConfig_DoesNotMutateInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))

			return
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	kcAdapter, _, _ := initAdapter(t, server)

	// a slice with spare capacity would be written to by a naive append
	scopes := make([]string, 1, 4)
	scopes[0] = "scope-a"

	policy := &ClientRegistrationPolicy{
		ProviderID: AllowedClientScopesPolicyProviderID,
		SubType:    AllowedClientScopesPolicySubTypeAuthenticated,
		Config:     map[string][]string{AllowedClientScopesPolicyProviderID: scopes},
	}

	require.NoError(t, kcAdapter.AddClientRegistrationPolicyConfig(context.Background(), "realm-name", policy))
	require.Equal(t, map[string][]string{AllowedClientScopesPolicyProviderID: {"scope-a"}}, policy.Config)
}

func decodeComponent(t *testing.T, body io.Reader) *Component {
	t.Helper()

	raw, err := io.ReadAll(body)
	require.NoError(t, err)

	var component Component
	require.NoError(t, json.Unmarshal(raw, &component))

	return &component
}

func TestAppendMissing(t *testing.T) {
	tests := []struct {
		name       string
		current    []string
		values     []string
		wantResult []string
		wantAdded  []string
	}{
		{
			name:       "all values are new",
			current:    nil,
			values:     []string{"a", "b"},
			wantResult: []string{"a", "b"},
			wantAdded:  []string{"a", "b"},
		},
		{
			name:       "nothing to add",
			current:    []string{"a", "b"},
			values:     []string{"b", "a"},
			wantResult: []string{"a", "b"},
			wantAdded:  nil,
		},
		{
			name:       "duplicated values are added once",
			current:    []string{"a"},
			values:     []string{"b", "b"},
			wantResult: []string{"a", "b"},
			wantAdded:  []string{"b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, added := appendMissing(tt.current, tt.values)
			require.Equal(t, tt.wantResult, result)
			require.Equal(t, tt.wantAdded, added)
		})
	}
}

func TestAppendMissing_DoesNotMutateInput(t *testing.T) {
	// a slice with spare capacity would be written to by a naive append
	current := make([]string, 1, 4)
	current[0] = "a"

	result, added := appendMissing(current, []string{"b"})

	require.Equal(t, []string{"a", "b"}, result)
	require.Equal(t, []string{"b"}, added)
	require.Equal(t, []string{"a"}, current)
	require.Empty(t, current[:2][1], "spare capacity of the input must not be written to")
}

func TestCloneConfig(t *testing.T) {
	values := make([]string, 1, 4)
	values[0] = "a"

	config := map[string][]string{"key": values}

	cloned := cloneConfig(config)
	require.Equal(t, config, cloned)

	cloned["key"] = append(cloned["key"], "b")

	require.Equal(t, []string{"a"}, config["key"], "the source config must not be modified")
}
