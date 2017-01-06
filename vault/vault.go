package vault

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"crypto/x509"

	vaultapi "github.com/hashicorp/vault/api"
)

type Vault struct {
	*vaultapi.Client
}

func (v *Vault) Secret() *Secret {
	return &Secret{vault: v}
}

func New(cfg Config) (*Vault, error) {
	if cfg.AuthType == AuthNone {
		return nil, errors.New("you have to set the auth type when using the vault backend")
	}

	var (
		vcfg   = vaultapi.DefaultConfig()
		tlsCfg = &tls.Config{}
	)

	vcfg.Address = cfg.Address

	if cfg.Cert != "" && cfg.Key != "" {
		certificate, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{certificate}
		tlsCfg.BuildNameToCertificate()
	}

	if cfg.CAcert != "" {
		cacert, err := ioutil.ReadFile(cfg.CAcert)
		if err != nil {
			return nil, err
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(cacert)
		tlsCfg.RootCAs = certPool
	}

	vcfg.HttpClient.Transport = &http.Transport{TLSClientConfig: tlsCfg}

	client, err := vaultapi.NewClient(vcfg)
	if err != nil {
		return nil, err
	}

	// auth typep
	var secret *vaultapi.Secret

	switch cfg.AuthType {
	case AuthAppID:
		secret, err = client.Logical().Write("/auth/app-id/login", map[string]interface{}{
			"app_id":  cfg.AppID,
			"user_id": cfg.UserID,
		})
	case AuthGithub:
		secret, err = client.Logical().Write("/auth/github/login", map[string]interface{}{
			"token": cfg.Token,
		})
	case AuthToken:
		client.SetToken(cfg.Token)
		secret, err = client.Logical().Read("/auth/token/lookup-self")
	case AuthPass:
		secret, err = client.Logical().Write(fmt.Sprintf("/auth/userpass/login/%s", cfg.Username), map[string]interface{}{
			"password": cfg.Password,
		})
	}

	if err != nil {
		return nil, err
	}

	if client.Token() == "" {
		client.SetToken(secret.Auth.ClientToken)
	}

	return &Vault{client}, nil
}