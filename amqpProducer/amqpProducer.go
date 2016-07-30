package amqpProducer

import (
	"fmt"
	log "github.com/Sirupsen/logrus"

	"crypto/tls"
	"crypto/x509"
	"github.com/aarnaud/go-conntrack-monitor/config"
	"github.com/streadway/amqp"
	"io/ioutil"
	"time"
)

func getUrl(config *config.ServiceConfig, isTLS bool) string {
	if isTLS {
		return fmt.Sprintf("amqps://%s:%s@%s:%d/", config.AMQPUser, config.AMQPPassword, config.AMQPHost, config.AMQPPort)
	} else {
		return fmt.Sprintf("amqp://%s:%s@%s:%d/", config.AMQPUser, config.AMQPPassword, config.AMQPHost, config.AMQPPort)
	}
}

func getConnection(config *config.ServiceConfig) (*amqp.Connection, error) {
	if config.AMQPKey != "" && config.AMQPCrt != "" && config.AMQPCa != "" {
		var isTLS = true
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

		if ca, err := ioutil.ReadFile(config.AMQPCa); err == nil {
			tlsCfg.RootCAs.AppendCertsFromPEM(ca)
		}

		// Move the client cert and key to a location specific to your application
		// and load them here.

		if cert, err := tls.LoadX509KeyPair(config.AMQPCrt, config.AMQPKey); err == nil {
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
		return amqp.DialTLS(getUrl(config, isTLS), tlsCfg)
	} else {
		var isTLS = false
		return amqp.Dial(getUrl(config, isTLS))
	}
}

func Channel(config *config.ServiceConfig) (*amqp.Channel, error) {

	connection, err := getConnection(config)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	log.Infoln("got AMQP Connection, getting Channel...")

	channel, err := connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	log.Infof("got Channel, declaring %q Exchange (%q)", config.AMQPExchangeType, config.AMQPExchange)

	if err := channel.ExchangeDeclare(
		config.AMQPExchange,     // name
		config.AMQPExchangeType, // type
		true,              // durable
		false,             // auto-deleted
		false,             // internal
		config.AMQPNoWait, // noWait
		nil,               // arguments
	); err != nil {
		return nil, fmt.Errorf("Exchange Declare: %s", err)
	}

	// Reliable publisher confirms require confirm.select support from the
	// connection.
	if !config.AMQPNoWait {
		log.Infoln("enabling publishing confirms.")

		confirms := channel.NotifyPublish(make(chan amqp.Confirmation))

		if err := channel.Confirm(false); err != nil {
			return nil, fmt.Errorf("Channel could not be put into confirm mode: %s", err)
		}

		go confirmRoutine(confirms)
	}
	log.Infoln("declared Exchange")

	return channel, nil
}

func Publish(channel *amqp.Channel, exchange string, body []byte) error {

	err := channel.Publish(
		exchange, // publish to an exchange
		"",       // routing to 0 or more queues
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			Timestamp:    time.Now(),
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		},
	)

	if err != nil {
		return fmt.Errorf("Exchange Publish: %s", err)
	}

	return nil
}

// One would typically keep a channel of publishings, a sequence number, and a
// set of unacknowledged sequence numbers and loop until the publishing channel
// is closed.
func confirmRoutine(confirms <-chan amqp.Confirmation) {
	for {
		if confirmed := <-confirms; confirmed.Ack {
			log.Debugf("confirmed delivery with delivery tag: %d", confirmed.DeliveryTag)
		} else {
			log.Errorf("failed delivery of delivery tag: %d", confirmed.DeliveryTag)
		}
	}

}
