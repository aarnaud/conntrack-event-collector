package clientAMQP

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/streadway/amqp"
	log "gitlab.com/OpenWifiPortal/conntrack-event-collector/logger"
	"io/ioutil"
	"time"
)

type ClientConfig struct {
	AMQPHost         string
	AMQPPort         int
	AMQPUser         string
	AMQPPassword     string
	AMQPCa           string
	AMQPCrt          string
	AMQPKey          string
	AMQPExchangeType string
	AMQPExchange     string
	AMQPRoutingKey   string
	AMQPNoWait       bool
}

type ClientWrapper struct {
	Connection          *amqp.Connection
	Channel             *amqp.Channel
	Config              ClientConfig
	ConnectionCloseChan chan *amqp.Error
	isOk				bool
}

func New(config ClientConfig) (*ClientWrapper, error) {
	amqpClient := &ClientWrapper{
		Config: config,
		isOk: false,
	}

	for {
		time.Sleep(time.Second)
		if err := amqpClient.connect(); err != nil {
			log.Errorln(err)
			continue
		}
		if err := amqpClient.startChannel(); err != nil {
			log.Errorln(err)
			continue
		}
		break
	}

	amqpClient.isOk = true

	go amqpClient.watchConnection()

	return amqpClient, nil
}

func (amqpClient *ClientWrapper) getUrl(isTLS bool) string {
	if isTLS {
		return fmt.Sprintf("amqps://%s:%s@%s:%d/", amqpClient.Config.AMQPUser, amqpClient.Config.AMQPPassword, amqpClient.Config.AMQPHost, amqpClient.Config.AMQPPort)
	} else {
		return fmt.Sprintf("amqp://%s:%s@%s:%d/", amqpClient.Config.AMQPUser, amqpClient.Config.AMQPPassword, amqpClient.Config.AMQPHost, amqpClient.Config.AMQPPort)
	}
}

func (amqpClient *ClientWrapper) connect() error {
	var err error
	if amqpClient.Config.AMQPKey != "" && amqpClient.Config.AMQPCrt != "" && amqpClient.Config.AMQPCa != "" {
		tlsCfg := new(tls.Config)

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

		if ca, err := ioutil.ReadFile(amqpClient.Config.AMQPCa); err == nil {
			tlsCfg.RootCAs.AppendCertsFromPEM(ca)
		}

		// Move the client cert and key to a location specific to your application
		// and load them here.

		if cert, err := tls.LoadX509KeyPair(amqpClient.Config.AMQPCrt, amqpClient.Config.AMQPKey); err == nil {
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
		if amqpClient.Connection, err = amqp.DialTLS(amqpClient.getUrl(true), tlsCfg); err != nil {
			return fmt.Errorf("Dial: %s", err)
		}
	} else {
		if amqpClient.Connection, err = amqp.Dial(amqpClient.getUrl(false)); err != nil {
			return fmt.Errorf("Dial: %s", err)
		}
	}
	return nil
}

func (amqpClient *ClientWrapper) startChannel() error {
	var err error

	//Make a Go channel for connection error
	amqpClient.ConnectionCloseChan = make(chan *amqp.Error)
	//Attach this Go channel
	amqpClient.Connection.NotifyClose(amqpClient.ConnectionCloseChan)

	log.Infoln("got AMQP Connection, getting Channel...")

	amqpClient.Channel, err = amqpClient.Connection.Channel()
	if err != nil {
		return fmt.Errorf("Channel: %s", err)
	}

	log.Infof("got Channel, declaring %q Exchange (%q)", amqpClient.Config.AMQPExchangeType, amqpClient.Config.AMQPExchange)

	if err := amqpClient.Channel.ExchangeDeclare(
		amqpClient.Config.AMQPExchange,     // name
		amqpClient.Config.AMQPExchangeType, // type
		true,  // durable
		false, // auto-deleted
		false, // internal
		amqpClient.Config.AMQPNoWait, // noWait
		nil, // arguments
	); err != nil {
		return fmt.Errorf("Exchange Declare: %s", err)
	}

	// Reliable publisher confirms require confirm.select support from the
	// connection.
	if !amqpClient.Config.AMQPNoWait {
		log.Infoln("enabling publishing confirms.")

		confirms := amqpClient.Channel.NotifyPublish(make(chan amqp.Confirmation, 128))

		if err := amqpClient.Channel.Confirm(false); err != nil {
			return fmt.Errorf("Channel could not be put into confirm mode: %s", err)
		}

		// Use a Go channel to stop confirmRoutine when AMQP channel is closed
		channelCloseChan := make(chan *amqp.Error)
		amqpClient.Channel.NotifyClose(channelCloseChan)
		go confirmRoutine(confirms, channelCloseChan)
	}
	log.Infoln("declared Exchange")

	return nil
}

func (amqpClient *ClientWrapper) watchConnection() {
	var err error
	for err = range amqpClient.ConnectionCloseChan {
		for err != nil {
			amqpClient.isOk = false
			log.Errorln(err)
			time.Sleep(time.Second)
			if err = amqpClient.connect(); err != nil {
				continue
			}
			if err = amqpClient.startChannel(); err != nil {
				continue
			}
		}
		amqpClient.isOk = true
	}
}

func (amqpClient *ClientWrapper) WaitConnection() {
	for !amqpClient.isOk {
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
			log.Infoln("Closing confirmRoutine")
			return err
		case confirmed := <-confirms:
			if confirmed.Ack {
				log.Debugf("confirmed delivery with delivery tag: %d", confirmed.DeliveryTag)
			} else {
				log.Debugf("failed delivery of delivery tag: %d", confirmed.DeliveryTag)
			}
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (amqpClient *ClientWrapper) Publish(exchange string, routingKey string, body []byte, replyTo string, headers amqp.Table) error {

	err := amqpClient.Channel.Publish(
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
		return fmt.Errorf("Exchange Publish: %s", err)
	}

	return nil
}

func (amqpClient *ClientWrapper) Consume(args amqp.Table) (<-chan amqp.Delivery, error) {
	q, err := amqpClient.Channel.QueueDeclare(
		amqpClient.Config.AMQPRoutingKey, // queue name
		false, // durable
		true,  // auto-deleted
		false, // internal
		amqpClient.Config.AMQPNoWait, // noWait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Declare: %s", err)
	}

	log.Printf("Binding queue %s to exchange %s", amqpClient.Config.AMQPRoutingKey, amqpClient.Config.AMQPExchange)
	if err := amqpClient.Channel.QueueBind(
		q.Name, // queue name
		"",     // routing key
		amqpClient.Config.AMQPExchange, // exchange
		false,
		args,
	); err != nil {
		return nil, fmt.Errorf("Exchange Declare: %s", err)
	}

	msgs, err := amqpClient.Channel.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)

	if err != nil {
		return nil, fmt.Errorf("Consume Declare: %s", err)
	}

	return msgs, nil
}
