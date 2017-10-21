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
  "timestamp": 1508566165785,
  "type": "NEW",
  "id": 0,
  "original": {
    "layer3": {
      "protonum": 2,
      "protoname": "ipv4",
      "src": "192.168.1.xxx",
      "dst": "xxx.xxx.xxx.xxx"
    },
    "layer4": {
      "protonum": 0,
      "protoname": "tcp",
      "sport": 42216,
      "dport": 80
    },
    "counter": {
      "packets": 0,
      "bytes": 0
    }
  },
  "reply": {
    "layer3": {
      "protonum": 2,
      "protoname": "ipv4",
      "src": "xxx.xxx.xxx.xxx",
      "dst": "192.168.0.xxx"
    },
    "layer4": {
      "protonum": 0,
      "protoname": "tcp",
      "sport": 80,
      "dport": 42216
    },
    "counter": {
      "packets": 0,
      "bytes": 0
    }
  },
  "UNREPLIED": false,
  "ASSURED": false
}
```

### DESTROY

```json
{
  "timestamp": 1508566186345,
  "type": "DESTROY",
  "id": 0,
  "original": {
    "layer3": {
      "protonum": 2,
      "protoname": "ipv4",
      "src": "192.168.1.xxx",
      "dst": "xxx.xxx.xxx.xxx"
    },
    "layer4": {
      "protonum": 0,
      "protoname": "tcp",
      "sport": 34277,
      "dport": 80
    },
    "counter": {
      "packets": 4,
      "bytes": 305
    }
  },
  "reply": {
    "layer3": {
      "protonum": 2,
      "protoname": "ipv4",
      "src": "xxx.xxx.xxx.xxx",
      "dst": "192.168.0.xxx"
    },
    "layer4": {
      "protonum": 0,
      "protoname": "tcp",
      "sport": 80,
      "dport": 34277
    },
    "counter": {
      "packets": 3,
      "bytes": 291
    }
  },
  "UNREPLIED": false,
  "ASSURED": false
}
```