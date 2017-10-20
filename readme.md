# conntrack-event-collector

## Building it

1. Install Go
2. Install conntrack-tools
3. Compile

```bash
CGO_ENABLED=0 go build -a
```

## Benchmark

* On ARM Allwinner A20 : `1500 events/s`
* On Intel® Core™ i5-4440 CPU: `18000 events/s`
* On MIPS1004Kc Dual-Core 880 MHz : `1100 events/s`

## Use conntrack without sudo

```
sudo setcap cap_net_admin+ep /usr/sbin/conntrack
```

## Usage

```
Usage:
   [flags]
   [command]

Available Commands:
  help        Help about any command
  version     Print the version.

Flags:
      --amqp-ca string         CA certificate
      --amqp-crt string        RabbitMQ client cert
      --amqp-exchange string   RabbitMQ Exchange (default "conntrack")
      --amqp-host string       RabbitMQ Host (default "localhost")
      --amqp-key string        RabbitMQ client key
      --amqp-password string   RabbitMQ password (default "guest")
      --amqp-port int          RabbitMQ Port (default 5672)
      --amqp-user string       RabbitMQ user (default "guest")
  -h, --help                   help for this command
  -v, --verbose                Enable verbose

```

## Example of event

### NEW

```json
{
  "Timestamp": 0,
  "Type": "NEW",
  "Id": 0,
  "Original": {
    "Layer3": {
      "Protonum": 2,
      "Protoname": "ipv4",
      "Src": "192.168.1.xxx",
      "Dst": "xxx.xxx.xxx.xxx"
    },
    "Layer4": {
      "Protonum": 0,
      "Protoname": "tcp",
      "Sport": 42216,
      "Dport": 80
    },
    "Counter": {
      "Packets": 0,
      "Bytes": 0
    }
  },
  "Reply": {
    "Layer3": {
      "Protonum": 2,
      "Protoname": "ipv4",
      "Src": "xxx.xxx.xxx.xxx",
      "Dst": "192.168.0.xxx"
    },
    "Layer4": {
      "Protonum": 0,
      "Protoname": "tcp",
      "Sport": 80,
      "Dport": 42216
    },
    "Counter": {
      "Packets": 0,
      "Bytes": 0
    }
  },
  "UNREPLIED": false,
  "ASSURED": false
}
```

### DESTROY

```json
{
  "Timestamp": 0,
  "Type": "DESTROY",
  "Id": 0,
  "Original": {
    "Layer3": {
      "Protonum": 2,
      "Protoname": "ipv4",
      "Src": "192.168.1.xxx",
      "Dst": "xxx.xxx.xxx.xxx"
    },
    "Layer4": {
      "Protonum": 0,
      "Protoname": "tcp",
      "Sport": 34277,
      "Dport": 80
    },
    "Counter": {
      "Packets": 4,
      "Bytes": 305
    }
  },
  "Reply": {
    "Layer3": {
      "Protonum": 2,
      "Protoname": "ipv4",
      "Src": "xxx.xxx.xxx.xxx",
      "Dst": "192.168.0.xxx"
    },
    "Layer4": {
      "Protonum": 0,
      "Protoname": "tcp",
      "Sport": 80,
      "Dport": 34277
    },
    "Counter": {
      "Packets": 3,
      "Bytes": 291
    }
  },
  "UNREPLIED": false,
  "ASSURED": false
}
```