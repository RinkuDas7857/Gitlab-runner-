package vault

import (
	"fmt"

	"gitlab.com/gitlab-org/gitlab-runner/common"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/secrets"
	"gitlab.com/gitlab-org/gitlab-runner/helpers/vault/service"
)

const (
	resolverName = "vault"
)

var newVaultService = service.NewVault

type resolver struct {
	secret common.Secret
}

func newResolver(secret common.Secret) common.SecretResolver {
	return &resolver{
		secret: secret,
	}
}

func (v *resolver) Name() string {
	return resolverName
}

func (v *resolver) IsSupported() bool {
	return v.secret.Vault != nil
}

func (v *resolver) Resolve() (map[string]string, error) {
	if !v.IsSupported() {
		return nil, secrets.NewResolvingUnsupportedSecretError(resolverName)
	}

	secret := v.secret.Vault

	url := secret.Server.URL
	namespace := secret.Server.Namespace

	s, err := newVaultService(url, namespace, secret)
	if err != nil {
		return nil, err
	}

	data, err := s.GetFields(secret, secret)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, common.ErrSecretNotFound
	}

	resolved := make(map[string]string)
	for key, value := range data {
		resolved[key] = fmt.Sprintf("%v", value)
	}

	return resolved, nil
}

func init() {
	common.GetSecretResolverRegistry().Register(newResolver)
}
