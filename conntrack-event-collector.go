package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/aarnaud/conntrack-event-collector/amqpProducer"
	"github.com/aarnaud/conntrack-event-collector/config"
	"github.com/aarnaud/conntrack-event-collector/conntrack"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	log "github.com/aarnaud/conntrack-event-collector/logger"
	"time"
)

var cli = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		runConntrackMonitor()
	},
}

var cliOptionVersion = &cobra.Command{
	Use:   "version",
	Short: "Print the version.",
	Long:  "The version of this program",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version 1.0.0")
	},
}

func init() {
	cli.AddCommand(cliOptionVersion)

	flags := cli.Flags()

	flags.BoolP("verbose", "v", false, "Enable verbose")
	viper.BindPFlag("verbose", flags.Lookup("verbose"))

	flags.BoolP("nat-only", "n", false, "Track nat only")
	viper.BindPFlag("nat_only", flags.Lookup("nat-only"))

	flags.String("amqp-host", "localhost", "RabbitMQ Host")
	viper.BindPFlag("amqp_host", flags.Lookup("amqp-host"))

	flags.Int("amqp-port", 5672, "RabbitMQ Port")
	viper.BindPFlag("amqp_port", flags.Lookup("amqp-port"))

	flags.String("amqp-ca", "", "CA certificate")
	viper.BindPFlag("amqp_ca", flags.Lookup("amqp-ca"))

	flags.String("amqp-crt", "", "RabbitMQ client cert")
	viper.BindPFlag("amqp_crt", flags.Lookup("amqp-crt"))

	flags.String("amqp-key", "", "RabbitMQ client key")
	viper.BindPFlag("amqp_key", flags.Lookup("amqp-key"))

	flags.String("amqp-user", "guest", "RabbitMQ user")
	viper.BindPFlag("amqp_user", flags.Lookup("amqp-user"))

	flags.String("amqp-password", "guest", "RabbitMQ password")
	viper.BindPFlag("amqp_password", flags.Lookup("amqp-password"))

	flags.String("amqp-exchange", "conntrack", "RabbitMQ Exchange")
	viper.BindPFlag("amqp_exchange", flags.Lookup("amqp-exchange"))
}

func main() {
	cli.Execute()
}

var flow_messages = make(chan conntrack.Flow, 128)

func publishFlow(flowChan <-chan conntrack.Flow, conf *config.ServiceConfig) {
	var channel *amqp.Channel
	var err error
	var connectionCloseChan chan *amqp.Error = make(chan *amqp.Error)

	channel, connectionCloseChan, err = amqpProducer.Channel(conf)
	// Retry if error
	for err != nil {
		log.Errorln(err)
		channel, connectionCloseChan, err = amqpProducer.Channel(conf)
		time.Sleep(time.Second)
	}

	routerId := config.GetId()
	for {
		select {
		case err = <-connectionCloseChan:
			// Retry if error
			for err != nil {
				log.Errorln(err)
				channel, connectionCloseChan, err = amqpProducer.Channel(conf)
				time.Sleep(time.Second)
			}
		case flow := <-flowChan:
			if flow.Type != "" {
				body, err := json.Marshal(flow)
				if err != nil {
					log.Errorln(err)
					continue
				}
				err = amqpProducer.Publish(channel, conf.AMQPExchange, body, routerId)
				if err != nil {
					log.Errorln(err)
					continue
				}
			}
		}
	}
}

func runConntrackMonitor() {
	viper.SetConfigName("conntrack-event-collector")  // name of config file (without extension)
	viper.AddConfigPath("/etc/owp")        // path to look for the config file in
	viper.AddConfigPath("/etc/config/owp") // path to look for the config file in
	viper.AddConfigPath("$HOME/.owp")      // call multiple times to add many search paths
	viper.AddConfigPath(".")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {            // Handle errors reading the config file
		log.Infoln(err)
	}

	// EXPORT OWP_AMQP_HOST=hop
	viper.SetEnvPrefix("owp")
	viper.AutomaticEnv()

	if viper.GetBool("verbose") {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	log.SetFormatter(log.GetFormater())
	log.Info("Starting...")
	log.Infof("Mac address : %s", config.GetMacAddr())
	log.Infof("Uuid : %s", config.GetId())

	conf := &config.ServiceConfig{
		AMQPHost:         viper.GetString("amqp_host"),
		AMQPPort:         viper.GetInt("amqp_port"),
		AMQPUser:         viper.GetString("amqp_user"),
		AMQPPassword:     viper.GetString("amqp_password"),
		AMQPCa:           viper.GetString("amqp_ca"),
		AMQPCrt:          viper.GetString("amqp_crt"),
		AMQPKey:          viper.GetString("amqp_key"),
		AMQPExchangeType: "direct", //Exchange type - direct|fanout|topic|x-custom
		AMQPExchange:     viper.GetString("amqp_exchange"),
		AMQPNoWait:       false,
	}

	log.Debugf("Config: %+v", conf)

	go publishFlow(flow_messages, conf)

	conntrack.Watch(flow_messages, []string{"NEW", "DESTROY"}, viper.GetBool("nat_only"))
}
