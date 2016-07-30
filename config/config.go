package config

//ServerConfig is the server config struct
type ServiceConfig struct {
	AMQPHost         string
	AMQPPort         int
	AMQPUser         string
	AMQPPassword     string
	AMQPCa           string
	AMQPCrt          string
	AMQPKey          string
	AMQPExchangeType string
	AMQPExchange     string
	AMQPNoWait       bool
}
