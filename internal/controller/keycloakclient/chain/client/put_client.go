package client

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/helper"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/internal/controller/keycloakclient/chain"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	keycloakApi "github.com/edenlabllc/keycloak-config-sync.operators.infra/api/v1alpha1"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/dto"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/secretref"
)

const (
	secretLength = 32 // 32 bytes = 256 bits of entropy

	browserAuthFlow     = "browser"
	directGrantAuthFlow = "direct_grant"
)

// secretRef is an interface for getting secret from ref.
type secretRef interface {
	GetSecretFromRef(ctx context.Context, refVal, secretNamespace string) (string, error)
}

type PutClient struct {
	keycloakApiClient keycloak.Client
	k8sClient         client.Client
	secretRef         secretRef
}

func NewPutClient(keycloakApiClient keycloak.Client, k8sClient client.Client, secretRef secretRef) *PutClient {
	return &PutClient{
		keycloakApiClient: keycloakApiClient,
		k8sClient:         k8sClient,
		secretRef:         secretRef,
	}
}

func (el *PutClient) Serve(ctx context.Context, keycloakClient *DataClient, realmName string) error {
	id, err := el.putKeycloakClient(ctx, keycloakClient, realmName)
	if err != nil {
		el.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync client: %s", err.Error()))

		return fmt.Errorf("unable to put keycloak client: %w", err)
	}

	keycloakClient.ReasonScope.ClientID = id

	el.setSuccessCondition(ctx, keycloakClient, "Client synchronized with Keycloak")

	return nil
}

func (el *PutClient) setFailureCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionClientSynced,
		metav1.ConditionFalse,
		chain.ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (el *PutClient) setSuccessCondition(ctx context.Context, keycloakClient *DataClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, el.k8sClient, keycloakClient,
		chain.ConditionClientSynced,
		metav1.ConditionTrue,
		chain.ReasonClientUpdated,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (el *PutClient) putKeycloakClient(ctx context.Context, keycloakClient *DataClient, realmName string) (string, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start creation of Keycloak client")

	var (
		authFlowOverrides map[string]string
		err               error
	)

	if keycloakClient.Client.AuthenticationFlowBindingOverrides != nil {
		authFlowOverrides, err = el.getAuthFlows(keycloakClient, realmName)
		if err != nil {
			return "", fmt.Errorf("unable to get auth flows: %w", err)
		}
	}

	clientDto, err := el.convertCrToDto(ctx, keycloakClient, realmName, authFlowOverrides)
	if err != nil {
		return "", fmt.Errorf("error during convertCrToDto: %w", err)
	}

	clientID, err := el.keycloakApiClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
	if err != nil && !adapter.IsErrNotFound(err) {
		return "", fmt.Errorf("unable to check client id: %w", err)
	}

	if clientID != "" {
		log.Info("Client already exists")

		clientDto.ID = clientID
		if updErr := el.keycloakApiClient.UpdateClient(ctx, clientDto); updErr != nil {
			return "", fmt.Errorf("unable to update keycloak client: %w", updErr)
		}

		return clientID, nil
	}

	err = el.keycloakApiClient.CreateClient(ctx, clientDto)
	if err != nil {
		return "", fmt.Errorf("unable to create client: %w", err)
	}

	log.Info("End put keycloak client")

	id, err := el.keycloakApiClient.GetClientID(clientDto.ClientId, clientDto.RealmName)
	if err != nil {
		return "", fmt.Errorf("unable to check client id: %w", err)
	}

	return id, nil
}

func (el *PutClient) convertCrToDto(ctx context.Context, keycloakClient *DataClient, realmName string, authflowOverrides map[string]string) (*dto.Client, error) {
	if keycloakClient.Client.Public {
		res := ConvertDataClientToClient(&keycloakClient.Client, "", realmName, authflowOverrides)
		return res, nil
	}

	secret, err := el.getSecret(ctx, keycloakClient)
	if err != nil {
		return nil, fmt.Errorf("unable to get secret, err: %w", err)
	}

	return ConvertDataClientToClient(&keycloakClient.Client, secret, realmName, authflowOverrides), nil
}

func (el *PutClient) getSecret(ctx context.Context, keycloakClient *DataClient) (string, error) {
	if keycloakClient.Client.Secret != "" {
		// We need to set secret in a new format for old clients for backward compatibility.
		// TODO: This code can be removed in the future.
		if !secretref.HasSecretRef(keycloakClient.Client.Secret) {
			if err := el.setSecretRef(ctx, keycloakClient); err != nil {
				return "", err
			}
		}

		secretVal, err := el.secretRef.GetSecretFromRef(ctx, keycloakClient.Client.Secret, keycloakClient.Namespace)
		if err != nil {
			return "", fmt.Errorf("unable to get secret from ref: %w", err)
		}

		return secretVal, nil
	}

	return el.generateSecret(ctx, keycloakClient)
}

func (el *PutClient) generateSecret(ctx context.Context, keycloakClient *DataClient) (string, error) {
	var clientSecret corev1.Secret

	secretName := fmt.Sprintf("keycloak-client-%s-secret",
		helper.RemoveSpecialChar(keycloakClient.Client.Name))

	secretErr := el.k8sClient.Get(ctx, types.NamespacedName{
		Namespace: keycloakClient.Namespace,
		Name:      secretName,
	}, &clientSecret)
	if secretErr != nil && !k8sErrors.IsNotFound(secretErr) {
		return "", fmt.Errorf("unable to check client secret existence: %w", secretErr)
	}

	pass, err := generateSecureSecret()
	if err != nil {
		return "", fmt.Errorf("unable to generate secret: %w", err)
	}

	if k8sErrors.IsNotFound(secretErr) {
		clientSecret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: keycloakClient.Namespace,
				Name:      secretName,
			},
			Data: map[string][]byte{
				keycloakApi.ClientSecretKey: []byte(pass),
			},
		}

		if err := controllerutil.SetControllerReference(
			keycloakClient.CRD, &clientSecret, el.k8sClient.Scheme()); err != nil {
			return "", fmt.Errorf("unable to set controller ref for secret: %w", err)
		}

		if err := el.k8sClient.Create(ctx, &clientSecret); err != nil {
			return "", fmt.Errorf("unable to create secret %+v, err: %w", clientSecret, err)
		}
	}

	keycloakClient.Client.Secret = secretref.GenerateSecretRef(clientSecret.Name, keycloakApi.ClientSecretKey)

	updateCRDSecretRef(keycloakClient)

	if err := el.k8sClient.Update(ctx, keycloakClient.CRD); err != nil {
		return "", fmt.Errorf("unable to update client with new secret: %s, err: %w", clientSecret.Name, err)
	}

	return string(clientSecret.Data[keycloakApi.ClientSecretKey]), nil
}

func (el *PutClient) setSecretRef(ctx context.Context, keycloakClient *DataClient) error {
	ref := secretref.GenerateSecretRef(keycloakClient.Client.Secret, keycloakApi.ClientSecretKey)
	keycloakClient.Client.Secret = ref

	updateCRDSecretRef(keycloakClient)

	if err := el.k8sClient.Update(ctx, keycloakClient.CRD); err != nil {
		return fmt.Errorf("unable to update client with secret ref %s: %w", ref, err)
	}

	return nil
}

func (el *PutClient) getAuthFlows(keycloakClient *DataClient, realmName string) (map[string]string, error) {
	clientAuthFlows := keycloakClient.Client.AuthenticationFlowBindingOverrides

	flows, err := el.keycloakApiClient.GetRealmAuthFlows(realmName)
	if err != nil {
		return nil, fmt.Errorf("unable to get realm: %w", err)
	}

	realmAuthFlows := make(map[string]string)
	for i := range flows {
		realmAuthFlows[flows[i].Alias] = flows[i].ID
	}

	authFlowOverrides := make(map[string]string)

	if clientAuthFlows.Browser != "" {
		if _, ok := realmAuthFlows[clientAuthFlows.Browser]; !ok {
			return nil, fmt.Errorf("auth flow %s not found in realm %s", clientAuthFlows.Browser, realmName)
		}

		authFlowOverrides[browserAuthFlow] = realmAuthFlows[clientAuthFlows.Browser]
	}

	if clientAuthFlows.DirectGrant != "" {
		if _, ok := realmAuthFlows[clientAuthFlows.DirectGrant]; !ok {
			return nil, fmt.Errorf("auth flow %s not found in realm %s", clientAuthFlows.DirectGrant, realmName)
		}

		authFlowOverrides[directGrantAuthFlow] = realmAuthFlows[clientAuthFlows.DirectGrant]
	}

	return authFlowOverrides, nil
}

func updateCRDSecretRef(keycloakClient *DataClient) {
	// update scd data
	for index, cl := range keycloakClient.CRD.Spec.Client {
		if index == keycloakClient.ClientIndex {
			cl.Secret = keycloakClient.Client.Secret
		}
	}
}

// generateSecureSecret generates a cryptographically secure random secret
func generateSecureSecret() (string, error) {
	bytes := make([]byte, secretLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}
