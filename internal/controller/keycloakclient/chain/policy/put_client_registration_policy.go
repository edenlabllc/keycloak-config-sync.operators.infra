package policy

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
)

// PutClientRegistrationPolicy applies the config listed in
// KeycloakClientSpec.ClientRegistrationPolicy to the matching client registration policy
// of the realm, e.g. making client scopes trusted for dynamic client registration.
type PutClientRegistrationPolicy struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
}

func NewPutClientRegistrationPolicy(
	keycloakApiClient keycloak.Client,
	k8sClient client.Client,
) *PutClientRegistrationPolicy {
	return &PutClientRegistrationPolicy{keycloakApiClient: keycloakApiClient, k8sClient: k8sClient}
}

func (el *PutClientRegistrationPolicy) WithKeycloakApiClient(keycloakApiClient keycloak.Client) {
	el.keycloakApiClient = keycloakApiClient
}

func (el *PutClientRegistrationPolicy) Serve(ctx context.Context, policy *DataPolicy, realmName string) error {
	if err := el.putClientRegistrationPolicy(ctx, policy, realmName); err != nil {
		el.setFailureCondition(ctx, policy,
			fmt.Sprintf("Failed to sync client registration policy: %s", err.Error()))

		return fmt.Errorf("error during putClientRegistrationPolicy: %w", err)
	}

	el.setSuccessCondition(ctx, policy, "Client registration policy synchronized")

	return nil
}

func (el *PutClientRegistrationPolicy) setFailureCondition(ctx context.Context, policy *DataPolicy, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, policy,
		chain.ConditionClientRegistrationPolicySynced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutClientRegistrationPolicy) setSuccessCondition(ctx context.Context, policy *DataPolicy, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, policy,
		chain.ConditionClientRegistrationPolicySynced,
		metav1.ConditionTrue,
		chain.ReasonClientRegistrationPolicySynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutClientRegistrationPolicy) putClientRegistrationPolicy(
	ctx context.Context,
	policy *DataPolicy,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	for _, allowedPolicy := range policy.Policies {
		target := toClientRegistrationPolicy(allowedPolicy)
		if len(target.Config) == 0 {
			continue
		}

		log.Info("Start applying client registration policy config",
			"realm", realmName,
			"providerId", target.ProviderID,
			"subType", target.SubType,
			"config", target.Config)

		if err := el.keycloakApiClient.AddClientRegistrationPolicyConfig(ctx, realmName, target); err != nil {
			return fmt.Errorf("unable to apply client registration policy config %s/%s: %w",
				target.ProviderID, target.SubType, err)
		}
	}

	return nil
}

// toClientRegistrationPolicy maps a spec entry onto the adapter DTO. ProviderID is required by
// the CRD and is passed through as is, so that an unsupported value is reported by Keycloak
// instead of being silently substituted. SubType is optional, its CRD default is applied here as
// well so that the handler does not depend on API server defaulting.
func toClientRegistrationPolicy(allowedPolicy keycloakApi.ClientAllowedPolicy) *adapter.ClientRegistrationPolicy {
	subType := allowedPolicy.SubType
	if subType == "" {
		subType = adapter.AllowedClientScopesPolicySubTypeAuthenticated
	}

	return &adapter.ClientRegistrationPolicy{
		Name:       allowedPolicy.Name,
		ProviderID: allowedPolicy.ProviderID,
		SubType:    subType,
		Config:     policyConfig(allowedPolicy.Config),
	}
}

// policyConfig returns the config with blank keys and values dropped and duplicates removed
// per key, preserving the order from the spec. Keys left without values are omitted.
func policyConfig(config map[string][]string) map[string][]string {
	cleaned := make(map[string][]string, len(config))

	for key, values := range config {
		if key == "" {
			continue
		}

		unique := uniqueNonEmpty(values)
		if len(unique) == 0 {
			continue
		}

		cleaned[key] = unique
	}

	return cleaned
}

// uniqueNonEmpty returns the unique non-empty values, preserving the order from the spec.
func uniqueNonEmpty(values []string) []string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, v := range values {
		if v == "" {
			continue
		}

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		unique = append(unique, v)
	}

	return unique
}
