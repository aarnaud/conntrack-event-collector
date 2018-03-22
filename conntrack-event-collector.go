package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"gitlab.com/OpenWifiPortal/conntrack-event-collector/config"
	"gitlab.com/OpenWifiPortal/conntrack-event-collector/conntrack"
	"gitlab.com/OpenWifiPortal/go-libs/amqp_tools"
	log "gitlab.com/OpenWifiPortal/go-libs/logger"
)

var amqpClient *amqp_tools.ClientWrapper
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
		fmt.Printf("Version 1.1.0")
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

	flags.String("vault-addr", "http://127.0.0.1:8200", "Vault address")
	viper.BindPFlag("vault_addr", flags.Lookup("vault-addr"))

	flags.String("vault-token", "", "Vault Token")
	viper.BindPFlag("vault_token", flags.Lookup("vault-token"))

	flags.String("vault-path-creds", "rabbitmq/creds/owp", "Vault Credentials Path for rabbitmq")
	viper.BindPFlag("vault_path_creds", flags.Lookup("vault-path-creds"))

	flags.String("vault-path-config", "secret/owp/conntrack-event-collector", "Vault Config Path for rabbitmq")
	viper.BindPFlag("vault_path_config", flags.Lookup("vault-path-config"))
}

func main() {
	cli.Execute()
}

var flowMessages = make(chan conntrack.Flow, 128)

func publishFlow(flowChan <-chan conntrack.Flow) {
	routerId := config.GetId()
	for flow := range flowChan {
		if flow.Type != "" {
			body, err := json.Marshal(flow)
			if err != nil {
				log.Errorln(err)
				continue
			}
			err = amqpClient.Publish(amqpClient.Config.Exchange, "", body, "", amqp.Table{
				"router_id": routerId,
			})
			if err != nil {
				log.Errorln(err)
				amqpClient.WaitConnection()
				continue
			}
		}
	}
}

func runConntrackMonitor() {
	viper.SetConfigName("conntrack-event-collector") // name of config file (without extension)
	viper.AddConfigPath("/etc/owp")                  // path to look for the config file in
	viper.AddConfigPath("/etc/config/owp")           // path to look for the config file in
	viper.AddConfigPath("$HOME/.owp")                // call multiple times to add many search paths
	viper.AddConfigPath(".")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
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
	log.Info("starting...")
	log.Infof("mac address : %s", config.GetMacAddr())
	log.Infof("uuid : %s", config.GetId())

	config.Config = &config.ServiceConfig{
		ClientAMQPConfig: amqp_tools.ClientConfig{
			Host:            viper.GetString("amqp_host"),
			Port:            viper.GetInt("amqp_port"),
			Username:        viper.GetString("amqp_user"),
			Password:        viper.GetString("amqp_password"),
			Ca:              viper.GetString("amqp_ca"),
			Crt:             viper.GetString("amqp_crt"),
			Key:             viper.GetString("amqp_key"),
			ExchangeType:    "direct", //Exchange type - direct|fanout|topic|x-custom
			Exchange:        viper.GetString("amqp_exchange"),
			NoWait:          false,
			VaultAddr:       viper.GetString("vault_addr"),
			VaultToken:      viper.GetString("vault_token"),
			VaultPathCreds:  viper.GetString("vault_path_creds"),
			VaultPathConfig: viper.GetString("vault_path_config"),
		},
		NatOnly: viper.GetBool("nat_only"),
	}

	log.Debugf("config: %+v", config.Config)

	amqpClient, err = amqp_tools.New(&config.Config.ClientAMQPConfig)

	go publishFlow(flowMessages)

	conntrack.Watch(flowMessages, []string{"NEW", "DESTROY"}, config.Config.NatOnly)
}
