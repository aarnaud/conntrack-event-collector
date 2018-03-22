package vault_tools

import (
	"fmt"
	vault_api "github.com/hashicorp/vault/api"
	log "gitlab.com/OpenWifiPortal/go-libs/logger"
	"time"
)

type secretCallback func(secret *vault_api.Secret)
type ClientWrapper struct {
	Client *vault_api.Client
}

func (vault *ClientWrapper) Init(vaultAddr, token string) {
	var err error
	vault.Client, err = vault_api.NewClient(&vault_api.Config{
		Address:    vaultAddr,
		MaxRetries: 10,
	})
	if err != nil {
		log.Fatal("[vault] ", err)
	}

	vault.Client.SetToken(token)

	token_secret, err := vault.Client.Auth().Token().LookupSelf()
	if err != nil {
		log.Fatal("[vault] ", err)
	}

	isRenewable, _ := token_secret.TokenIsRenewable()
	if isRenewable {
		// Get a renewed token
		secret, err := vault.Client.Auth().Token().RenewTokenAsSelf(token, 0)
		if err != nil {
			log.Fatal("[vault] ", err)
		}

		token_renewer, err := vault.Client.NewRenewer(&vault_api.RenewerInput{
			Secret: secret,
		})
		if err != nil {
			log.Fatal("[vault] ", err)
		}

		watch_renewer_vault(token_renewer)
	} else {
		ttl, _ := token_secret.TokenTTL()
		log.Infof("[vault] token is not renewable, ttl: %d", int32(ttl))
	}
}

func (vault *ClientWrapper) GetSecret(path string, fn secretCallback) error {
	var secret *vault_api.Secret
	var err error
	secret, err = vault.Client.Logical().Read(path)
	if err != nil {
		return err
	}
	if secret == nil {
		return fmt.Errorf("secret not found : %s", path)
	}

	// return the secret
	fn(secret)

	if secret.Renewable {
		renewer, err := vault.Client.NewRenewer(&vault_api.RenewerInput{
			Secret: secret,
		})
		if err != nil {
			log.Fatal("[vault] ", err)
		}

		watch_renewer_vault(renewer)
	} else {
		log.Infof("[vault] secret is not renewable, use TTL to refresh secret : %s", path)
		// Refresh secret at the end of Lease
		if secret.LeaseDuration > 0 {
			go func() {
				for {
					time.Sleep(time.Duration(secret.LeaseDuration) * time.Second)
					secret, err = vault.Client.Logical().Read(path)
					if err != nil {
						log.Errorln("[vault]", err)
						continue
					}
					if secret == nil {
						log.Errorln("[vault] secret not found : %s", path)
						continue
					}
					fn(secret)
					log.Infof("[vault] successfully refreshed : %s", path)
				}
			}()
		}
	}
	return nil
}

func watch_renewer_vault(renewer *vault_api.Renewer) {
	go func() {
		for {
			select {
			case err := <-renewer.DoneCh():
				if err != nil {
					log.Fatal("[vault]", err)
				}

				// Renewal is now over
			case renewal := <-renewer.RenewCh():
				var flag string
				flag = renewal.Secret.LeaseID
				if flag == "" {
					flag = "token"
				}
				log.Printf("[vault] successfully renewed: %s", flag)
			}
		}
	}()
	go func() {
		for {
			renewer.Renew()
		}
	}()
}
