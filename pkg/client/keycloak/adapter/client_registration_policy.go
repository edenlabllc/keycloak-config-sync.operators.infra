package adapter

import (
	"context"
	"fmt"
	"slices"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// ClientRegistrationPolicyProviderType is a providerType of the realm components
	// which represent client registration policies.
	ClientRegistrationPolicyProviderType = "org.keycloak.services.clientregistration.policy." +
		"ClientRegistrationPolicy"

	// AllowedClientScopesPolicyProviderID is a providerId of the client registration policy
	// which restricts the client scopes a client may request during registration.
	AllowedClientScopesPolicyProviderID = "allowed-client-scopes"

	// AllowedClientScopesPolicySubTypeAuthenticated is the subType of the policies shown as
	// "Authenticated access policies" in the Keycloak admin console.
	AllowedClientScopesPolicySubTypeAuthenticated = "authenticated"

	// AllowedClientScopesPolicySubTypeAnonymous is the subType of the policies shown as
	// "Anonymous access policies" in the Keycloak admin console.
	AllowedClientScopesPolicySubTypeAnonymous = "anonymous"

	// DefaultAllowedClientScopesPolicyName is the name Keycloak gives to the policy by default.
	DefaultAllowedClientScopesPolicyName = "Allowed Client Scopes"

	keycloakApiParamComponentType = "type"
	keycloakApiParamComponentName = "name"
)

// GetComponentsParams represents parameters for getting realm components.
type GetComponentsParams struct {
	// ProviderType filters components by their providerType.
	ProviderType *string

	// Name filters components by their name.
	Name *string
}

// GetComponents returns realm components, optionally filtered by params.
func (a GoCloakAdapter) GetComponents(
	ctx context.Context,
	realmName string,
	params *GetComponentsParams,
) ([]Component, error) {
	var components []Component

	req := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetResult(&components)

	if params != nil {
		if params.ProviderType != nil && *params.ProviderType != "" {
			req = req.SetQueryParam(keycloakApiParamComponentType, *params.ProviderType)
		}

		if params.Name != nil && *params.Name != "" {
			req = req.SetQueryParam(keycloakApiParamComponentName, *params.Name)
		}
	}

	rsp, err := req.Get(a.buildPath(realmComponent))
	if err = a.checkError(err, rsp); err != nil {
		return nil, fmt.Errorf("error during get components request: %w", err)
	}

	return components, nil
}

// ClientRegistrationPolicy describes the desired state of a client registration policy component.
type ClientRegistrationPolicy struct {
	// Name is a display name of the policy component, used only when it has to be created.
	Name string

	// ProviderID is a providerId of the policy component.
	ProviderID string

	// SubType is an access type of the policy: authenticated or anonymous.
	SubType string

	// Config is a component configuration, e.g. allowed-client-scopes mapped to a list of
	// client scope names.
	Config map[string][]string
}

func (p *ClientRegistrationPolicy) name() string {
	if p.Name == "" {
		return DefaultAllowedClientScopesPolicyName
	}

	return p.Name
}

// AddClientRegistrationPolicyConfig adds the given config values to a client registration policy
// of the realm. The policy is looked up by providerId and subType, so the "Authenticated access
// policies" and "Anonymous access policies" groups are addressed separately.
//
// A realm which was not bootstrapped with the Keycloak default client registration policies has
// no such policy at all, in which case it is created with the given config.
//
// Values which are already present under a config key are left untouched and the policy is updated
// only when the resulting config differs from the current one, which keeps the call idempotent
// across reconciles. Values and config keys managed outside of the operator are preserved,
// nothing is ever removed.
func (a GoCloakAdapter) AddClientRegistrationPolicyConfig(
	ctx context.Context,
	realmName string,
	policy *ClientRegistrationPolicy,
) error {
	log := ctrl.LoggerFrom(ctx)

	if len(policy.Config) == 0 {
		return nil
	}

	component, err := a.getClientRegistrationPolicy(ctx, realmName, policy.ProviderID, policy.SubType)
	if err != nil {
		return err
	}

	if component == nil {
		return a.createClientRegistrationPolicy(ctx, realmName, policy)
	}

	added := make(map[string][]string, len(policy.Config))

	for key, values := range policy.Config {
		merged, addedValues := appendMissing(component.Config[key], values)
		if len(addedValues) == 0 {
			continue
		}

		if component.Config == nil {
			component.Config = make(map[string][]string, len(policy.Config))
		}

		component.Config[key] = merged
		added[key] = addedValues
	}

	if len(added) == 0 {
		return nil
	}

	if err := a.UpdateComponent(ctx, realmName, component); err != nil {
		return fmt.Errorf("unable to update client registration policy %s/%s: %w",
			policy.ProviderID, policy.SubType, err)
	}

	log.Info("Client registration policy config has been updated",
		logKeyRealm, realmName,
		"providerId", policy.ProviderID,
		"subType", policy.SubType,
		"added", added)

	return nil
}

// getClientRegistrationPolicy returns the client registration policy component matching
// providerID and subType, or nil when the realm has no such policy.
func (a GoCloakAdapter) getClientRegistrationPolicy(
	ctx context.Context,
	realmName, providerID, subType string,
) (*Component, error) {
	providerType := ClientRegistrationPolicyProviderType

	components, err := a.GetComponents(ctx, realmName, &GetComponentsParams{ProviderType: &providerType})
	if err != nil {
		return nil, fmt.Errorf("unable to get client registration policies: %w", err)
	}

	for i := range components {
		if components[i].ProviderID == providerID && components[i].SubType == subType {
			return &components[i], nil
		}
	}

	return nil, nil
}

func (a GoCloakAdapter) createClientRegistrationPolicy(
	ctx context.Context,
	realmName string,
	policy *ClientRegistrationPolicy,
) error {
	log := ctrl.LoggerFrom(ctx)

	component := Component{
		Name:         policy.name(),
		ProviderID:   policy.ProviderID,
		ProviderType: ClientRegistrationPolicyProviderType,
		SubType:      policy.SubType,
		Config:       cloneConfig(policy.Config),
	}

	if err := a.CreateComponent(ctx, realmName, &component); err != nil {
		return fmt.Errorf("unable to create client registration policy %s/%s: %w",
			policy.ProviderID, policy.SubType, err)
	}

	log.Info("Client registration policy has been created",
		logKeyRealm, realmName,
		"name", component.Name,
		"providerId", policy.ProviderID,
		"subType", policy.SubType,
		"config", policy.Config)

	return nil
}

// cloneConfig deep copies a component config, so that the created component does not share
// its value slices with the caller.
func cloneConfig(config map[string][]string) map[string][]string {
	cloned := make(map[string][]string, len(config))

	for key, values := range config {
		cloned[key] = slices.Clone(values)
	}

	return cloned
}

// appendMissing appends to current the values which are not present in it yet and
// returns the resulting slice together with the values which have actually been added.
func appendMissing(current, values []string) (result, added []string) {
	// copy to avoid mutating the slice owned by the caller
	result = make([]string, len(current), len(current)+len(values))
	copy(result, current)

	for _, v := range values {
		if slices.Contains(result, v) {
			continue
		}

		result = append(result, v)
		added = append(added, v)
	}

	return result, added
}
