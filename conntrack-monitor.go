package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aarnaud/go-conntrack-monitor/conntrack"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		fmt.Printf("Version 0.0.1")
	},
}

func init() {
	cli.AddCommand(cliOptionVersion)

	flags := cli.Flags()

	flags.BoolP("verbose", "v", false, "Enable verbose")
	viper.BindPFlag("verbose", flags.Lookup("verbose"))

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

	flags.String("amqp-user", "admin", "RabbitMQ user")
	viper.BindPFlag("amqp_user", flags.Lookup("amqp-user"))

	flags.String("amqp-password", "admin", "RabbitMQ password")
	viper.BindPFlag("amqp_password", flags.Lookup("amqp-password"))
}

func main() {
	cli.Execute()
}

var count int = 0

var flow_messages = make(chan conntrack.Flow, 128)

func printFlow(flowChan <-chan conntrack.Flow) {
	for {
		flow := <-flowChan
		if flow.Type != "" {
			//log.Debugf("#+%v\n", flow)
			count++
		}

	}

}

func runConntrackMonitor() {
	// EXPORT OWP_AMQP_HOST=hop
	viper.SetEnvPrefix("owp")
	viper.AutomaticEnv()

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetFormatter(&log.TextFormatter{})
	log.Debugln("Config:", viper.AllSettings())
	log.Info("Starting...")

	go func() {
		for {
			time.Sleep(60 * time.Second)
			log.Infoln(fmt.Sprintf("average %d events/s", count/60))
			count = 0
		}
	}()
	go printFlow(flow_messages)
	conntrack.Watch(flow_messages, []string{"NEW", "DESTROY"}, false)
}
