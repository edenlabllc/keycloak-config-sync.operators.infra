package helper

import (
	"context"
	"errors"
	"fmt"

	"github.com/edenlabllc/keycloak-config-sync.operators.infra/api/common"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak"
	"github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloak/adapter"
	keycloakclientv2 "github.com/edenlabllc/keycloak-config-sync.operators.infra/pkg/client/keycloakv2"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	keycloakTokenSecretPrefix = "kc-token-"
	keycloakTokenSecretKey    = "token"
)

var (
	ErrKeycloakIsNotAvailable = errors.New("keycloak is not available")
	ErrKeycloakRealmNotFound  = errors.New("keycloak realm is not available")
)

// KeycloakAuthData contains data for keycloak authentication.
type KeycloakAuthData struct {
	// Url is keycloak url.
	Url string

	// SecretName is name of secret with keycloak credentials.
	SecretName string

	// SecretNamespace is namespace of secret with keycloak credentials.
	SecretNamespace string

	// AdminType is type of keycloak admin.
	AdminType string

	// KeycloakCRName is name of keycloak CR.
	KeycloakCRName string

	// CACert is root certificate authority.
	CACert string `json:"caCert,omitempty"`

	// InsecureSkipVerify controls whether api client verifies the server's certificate chain and host name.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

func (h *Helper) CreateKeycloakClientV2FromConfigRef(ctx context.Context, object ObjectWithConfigRef) (*keycloakclientv2.KeycloakClient, error) {
	authData, err := h.getKeycloakAuthDataFromConfigRef(object)
	if err != nil {
		return nil, err
	}

	return h.createKeycloakClientV2FromAuthData(ctx, authData)
}

func (h *Helper) CreateKeycloakClientFromConfigRef(ctx context.Context, object ObjectWithConfigRef) (keycloak.Client, error) {
	authData, err := h.getKeycloakAuthDataFromConfigRef(object)
	if err != nil {
		return nil, err
	}

	return h.CreateKeycloakClientFomAuthData(ctx, authData)
}

func (h *Helper) createKeycloakClientV2FromAuthData(ctx context.Context, authData *KeycloakAuthData) (*keycloakclientv2.KeycloakClient, error) {
	username, password, err := h.getCredentialsFromSecret(ctx, authData.SecretName, authData.SecretNamespace)
	if err != nil {
		return nil, fmt.Errorf("unable to get credentials: %w", err)
	}

	options := []keycloakclientv2.ClientOption{
		keycloakclientv2.WithPasswordGrant(username, password),
	}

	if authData.CACert != "" {
		options = append(options, keycloakclientv2.WithCACert(authData.CACert))
	}

	if authData.InsecureSkipVerify {
		options = append(options, keycloakclientv2.WithTLSInsecureSkipVerify(true))
	}

	kcClient, err := keycloakclientv2.NewKeycloakClient(ctx, authData.Url, keycloakclientv2.DefaultAdminClientID, options...)
	if err != nil {
		return nil, fmt.Errorf("unable to create keycloak v2 client: %w", err)
	}

	return kcClient, nil
}

func (h *Helper) CreateKeycloakClient(ctx context.Context, url, user, password, adminType, caCert string, insecureSkipVerify bool) (keycloak.Client, error) {
	clientAdapter, err := h.adapterBuilder(
		ctx,
		adapter.GoCloakConfig{
			Url:                url,
			User:               user,
			Password:           password,
			RootCertificate:    caCert,
			InsecureSkipVerify: insecureSkipVerify,
		},
		adminType, ctrl.LoggerFrom(ctx), h.restyClient,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to init kc client adapter: %w", err)
	}

	return clientAdapter, nil
}

func (h *Helper) InvalidateKeycloakClientTokenSecret(ctx context.Context, namespace, rootKeycloakName string) error {
	var secret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: tokenSecretName(rootKeycloakName)},
		&secret); err != nil {
		return fmt.Errorf("unable to get client token secret: %w", err)
	}

	if err := h.client.Delete(ctx, &secret); err != nil {
		return fmt.Errorf("unable to delete client token secret: %w", err)
	}

	return nil
}

func (h *Helper) CreateKeycloakClientFomAuthData(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error) {
	h.tokenSecretLock.Lock()
	defer h.tokenSecretLock.Unlock()

	clientAdapter, err := h.createKeycloakClientFromTokenSecret(ctx, authData)
	if err == nil {
		return clientAdapter, nil
	}

	if !k8sErrors.IsNotFound(err) && !adapter.IsErrTokenExpired(err) {
		return nil, fmt.Errorf("unable to create kc client from token secret: %w", err)
	}

	clientAdapter, err = h.createKeycloakClientFromLoginPassword(ctx, authData)
	if err != nil {
		return nil, fmt.Errorf("unable to create kc client from login password: %w", err)
	}

	return clientAdapter, nil
}

func (h *Helper) getCredentialsFromSecret(ctx context.Context, secretName, secretNamespace string) (username, password string, err error) {
	var secret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: secretNamespace,
	}, &secret); err != nil {
		return "", "", fmt.Errorf("secret not found: %w", err)
	}

	return string(secret.Data["username"]), string(secret.Data["password"]), nil
}

func (h *Helper) createKeycloakClientFromLoginPassword(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error) {
	username, password, err := h.getCredentialsFromSecret(ctx, authData.SecretName, authData.SecretNamespace)
	if err != nil {
		return nil, fmt.Errorf("unable to get credentials: %w", err)
	}

	clientAdapter, err := h.CreateKeycloakClient(ctx, authData.Url, username,
		password, authData.AdminType, authData.CACert, authData.InsecureSkipVerify)
	if err != nil {
		return nil, fmt.Errorf("unable to init authData client adapter: %w", err)
	}

	jwtToken, err := clientAdapter.ExportToken()
	if err != nil {
		return nil, fmt.Errorf("unable to export authData client token: %w", err)
	}

	if err := h.saveKeycloakClientTokenSecret(ctx, tokenSecretName(authData.SecretName), authData.SecretNamespace, jwtToken); err != nil {
		return nil, fmt.Errorf("unable to save authData token to secret: %w", err)
	}

	return clientAdapter, nil
}

func (h *Helper) createKeycloakClientFromTokenSecret(ctx context.Context, authData *KeycloakAuthData) (keycloak.Client, error) {
	var tokenSecret coreV1.Secret
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      tokenSecretName(authData.KeycloakCRName),
		Namespace: authData.SecretNamespace,
	}, &tokenSecret); err != nil {
		return nil, fmt.Errorf("unable to get token secret: %w", err)
	}

	clientAdapter, err := adapter.MakeFromToken(adapter.GoCloakConfig{
		Url:                authData.Url,
		RootCertificate:    authData.CACert,
		InsecureSkipVerify: authData.InsecureSkipVerify,
	}, tokenSecret.Data[keycloakTokenSecretKey], ctrl.LoggerFrom(ctx))
	if err != nil {
		return nil, fmt.Errorf("unable to make authData client from token: %w", err)
	}

	return clientAdapter, nil
}

func (h *Helper) saveKeycloakClientTokenSecret(ctx context.Context, secretName, secretNamespace string, token []byte) error {
	var secret coreV1.Secret

	err := h.client.Get(ctx, types.NamespacedName{Namespace: secretNamespace, Name: secretName}, &secret)
	if err == nil {
		secret.Data = map[string][]byte{
			keycloakTokenSecretKey: token,
		}

		if err = h.client.Update(ctx, &secret); err != nil {
			return fmt.Errorf("unable to update token secret: %w", err)
		}

		return nil
	}

	if k8sErrors.IsNotFound(err) {
		secret = coreV1.Secret{ObjectMeta: metav1.ObjectMeta{
			Namespace: secretNamespace,
			Name:      secretName,
		}, Data: map[string][]byte{
			keycloakTokenSecretKey: token,
		}}

		if err = h.client.Create(ctx, &secret); err != nil {
			return fmt.Errorf("unable to create token secret: %w", err)
		}

		return nil
	}

	return fmt.Errorf("error during token secret retrieval: %w", err)
}

func (h *Helper) getKeycloakAuthDataFromConfigRef(object ObjectWithConfigRef) (*KeycloakAuthData, error) {
	return MakeKeycloakAuthDataFromKeycloakConfig(h.watchNamespace, object.GetConfigRef())
}

func MakeKeycloakAuthDataFromKeycloakConfig(
	ns string,
	cfgRef common.ConfigRef,
) (*KeycloakAuthData, error) {
	return &KeycloakAuthData{
		Url:             cfgRef.Url,
		SecretName:      cfgRef.Secret,
		SecretNamespace: ns,
	}, nil
}

func tokenSecretName(keycloakName string) string {
	return fmt.Sprintf("%s%s", keycloakTokenSecretPrefix, keycloakName)
}
