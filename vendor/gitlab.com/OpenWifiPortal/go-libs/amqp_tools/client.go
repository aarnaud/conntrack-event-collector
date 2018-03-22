package amqp_tools

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	vault_api "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
	"github.com/streadway/amqp"
	log "gitlab.com/OpenWifiPortal/go-libs/logger"
	"gitlab.com/OpenWifiPortal/go-libs/vault_tools"
	"io/ioutil"
	"time"
)

type ClientConfig struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Ca              string
	Crt             string
	Key             string
	Vhost           string
	ExchangeType    string
	Exchange        string
	RoutingKey      string
	NoWait          bool
	VaultAddr       string
	VaultToken      string
	VaultPathConfig string
	VaultPathCreds  string
}

type ClientWrapper struct {
	Connection          *amqp.Connection
	Channel             *amqp.Channel
	Config              *ClientConfig
	ConnectionCloseChan chan *amqp.Error
	isOk                bool
}

func New(config *ClientConfig) (*ClientWrapper, error) {
	var err error
	var client *ClientWrapper

	//Use vault data if configured
	if config.VaultAddr != "" && config.VaultToken != "" {
		vault := vault_tools.ClientWrapper{}
		vault.Init(config.VaultAddr, config.VaultToken)

		// Get credentials
		err = vault.GetSecret(config.VaultPathCreds, func(secret *vault_api.Secret) {
			err := mapstructure.WeakDecode(secret.Data, config)
			if err != nil {
				log.Errorln("[amqp] ", err)
			}
		})
		if err != nil {
			log.Fatalln("[amqp] ", err)
		}

		// Get configuration
		err = vault.GetSecret(config.VaultPathConfig, func(secret *vault_api.Secret) {
			err := mapstructure.WeakDecode(secret.Data, config)
			if err != nil {
				log.Errorln("[amqp] ", err)
			}
		})
		if err != nil {
			log.Fatalln("[amqp] ", err)
		}
	}

	client = &ClientWrapper{
		Config: config,
		isOk:   false,
	}

	for {
		time.Sleep(time.Second)
		if err := client.connect(); err != nil {
			log.Errorln("[amqp] ", err)
			continue
		}
		if err := client.startChannel(); err != nil {
			log.Errorln("[amqp] ", err)
			continue
		}
		break
	}

	client.isOk = true

	go client.watchConnection()

	return client, nil
}

func (client *ClientWrapper) getUrl(isTLS bool) string {
	conf := client.Config
	if isTLS {
		return fmt.Sprintf("amqps://%s:%s@%s:%d/", conf.Username, conf.Password, conf.Host, conf.Port)
	} else {
		return fmt.Sprintf("amqp://%s:%s@%s:%d/", conf.Username, conf.Password, conf.Host, conf.Port)
	}
}

func (client *ClientWrapper) connect() error {
	var err error
	var tlsCfg *tls.Config
	if client.Config.Key != "" && client.Config.Crt != "" && client.Config.Ca != "" {
		tlsCfg = new(tls.Config)

		// The self-signing certificate authority's certificate must be included in
		// the RootCAs to be trusted so that the server certificate can be verified.
		//
		// Alternatively to adding it to the tls.Config you can add the CA's cert to
		// your system's root CAs.  The tls package will use the system roots
		// specific to each support OS.  Under OS X, add (drag/drop) your cacert.pem
		// file to the 'Certificates' section of KeyChain.app to add and always
		// trust.
		//
		// Or with the command line add and trust the DER encoded certificate:
		//
		//   security add-certificate testca/cacert.cer
		//   security add-trusted-cert testca/cacert.cer
		//
		// If you depend on the system root CAs, then use nil for the RootCAs field
		// so the system roots will be loaded.

		tlsCfg.RootCAs = x509.NewCertPool()

		if ca, err := ioutil.ReadFile(client.Config.Ca); err == nil {
			tlsCfg.RootCAs.AppendCertsFromPEM(ca)
		}

		// Move the client cert and key to a location specific to your application
		// and load them here.

		if cert, err := tls.LoadX509KeyPair(client.Config.Crt, client.Config.Key); err == nil {
			tlsCfg.Certificates = append(tlsCfg.Certificates, cert)
		}

		// Server names are validated by the crypto/tls package, so the server
		// certificate must be made for the hostname in the URL.  Find the commonName
		// (CN) and make sure the hostname in the URL matches this common name.  Per
		// the RabbitMQ instructions for a self-signed cert, this defautls to the
		// current hostname.
		//
		//   openssl x509 -noout -in server/cert.pem -subject
		//
		// If your server name in your certificate is different than the host you are
		// connecting to, set the hostname used for verification in
		// ServerName field of the tls.Config struct.
	}

	client.Connection, err = amqp.DialConfig(client.getUrl(false), amqp.Config{
		Heartbeat:       10 * time.Second,
		Locale:          "en_US",
		Vhost:           client.Config.Vhost,
		TLSClientConfig: tlsCfg,
	})
	if err != nil {
		return fmt.Errorf("Dial: %s", err)
	}
	return nil
}

func (client *ClientWrapper) startChannel() error {
	var err error

	//Make a Go channel for connection error
	client.ConnectionCloseChan = make(chan *amqp.Error)
	//Attach this Go channel
	client.Connection.NotifyClose(client.ConnectionCloseChan)

	log.Infoln("[amqp] got AMQP connection, getting Channel...")

	client.Channel, err = client.Connection.Channel()
	if err != nil {
		return fmt.Errorf("[amqp] channel: %s", err)
	}

	log.Infof("[amqp] got channel, declaring %q exchange (%q)", client.Config.Exchange, client.Config.ExchangeType)

	if err := client.Channel.ExchangeDeclare(
		client.Config.Exchange,     // name
		client.Config.ExchangeType, // type
		true,                 // durable
		false,                // auto-deleted
		false,                // internal
		client.Config.NoWait, // noWait
		nil,                  // arguments
	); err != nil {
		return fmt.Errorf("[amqp] exchange declare: %s", err)
	}

	// Reliable publisher confirms require confirm.select support from the
	// connection.
	if !client.Config.NoWait {
		log.Infoln("[amqp] enabling publishing confirms.")

		confirms := client.Channel.NotifyPublish(make(chan amqp.Confirmation, 128))

		if err := client.Channel.Confirm(false); err != nil {
			return fmt.Errorf("[amqp] channel could not be put into confirm mode: %s", err)
		}

		// Use a Go channel to stop confirmRoutine when AMQP channel is closed
		channelCloseChan := make(chan *amqp.Error)
		client.Channel.NotifyClose(channelCloseChan)
		go confirmRoutine(confirms, channelCloseChan)
	}
	log.Infoln("[amqp] declared exchange")

	return nil
}

func (client *ClientWrapper) watchConnection() {
	var err error
	for err = range client.ConnectionCloseChan {
		for err != nil {
			client.isOk = false
			log.Errorln("[amqp] ", err)
			time.Sleep(time.Second)
			if err = client.connect(); err != nil {
				continue
			}
			if err = client.startChannel(); err != nil {
				continue
			}
		}
		client.isOk = true
	}
}

func (client *ClientWrapper) WaitConnection() {
	for !client.isOk {
		time.Sleep(time.Second)
	}
}

// One would typically keep a channel of publishings, a sequence number, and a
// set of unacknowledged sequence numbers and loop until the publishing channel
// is closed.
func confirmRoutine(confirms <-chan amqp.Confirmation, channelCloseChan <-chan *amqp.Error) error {
	for {
		select {
		case err := <-channelCloseChan:
			log.Infoln("[amqp] closing confirmRoutine")
			return err
		case confirmed := <-confirms:
			if confirmed.Ack {
				log.Debugf("[amqp] confirmed delivery with delivery tag: %d", confirmed.DeliveryTag)
			} else {
				log.Debugf("[amqp] failed delivery of delivery tag: %d", confirmed.DeliveryTag)
			}
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (client *ClientWrapper) Publish(exchange string, routingKey string, body []byte, replyTo string, headers amqp.Table) error {

	err := client.Channel.Publish(
		exchange,   // publish to an exchange
		routingKey, // routing to 0 or more queues
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			Timestamp:    time.Now(),
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			ReplyTo:      replyTo,
			Headers:      headers,
		},
	)

	if err != nil {
		return fmt.Errorf("[amqp] exchange publish: %s", err)
	}

	return nil
}

func (client *ClientWrapper) Consume(args amqp.Table) (<-chan amqp.Delivery, error) {
	q, err := client.Channel.QueueDeclare(
		client.Config.RoutingKey, // queue name
		false,                // durable
		true,                 // auto-deleted
		false,                // internal
		client.Config.NoWait, // noWait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("[amqp] queue Declare: %s", err)
	}

	log.Printf("[amqp] binding queue %s to exchange %s", client.Config.RoutingKey, client.Config.Exchange)
	if err := client.Channel.QueueBind(
		q.Name, // queue name
		"",     // routing key
		client.Config.Exchange, // exchange
		false,
		args,
	); err != nil {
		return nil, fmt.Errorf("[amqp] exchange declare: %s", err)
	}

	msgs, err := client.Channel.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)

	if err != nil {
		return nil, fmt.Errorf("[amqp] consume Declare: %s", err)
	}

	return msgs, nil
}
