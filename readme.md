# conntrack-monitor

## Building it

1. Install Go
2. Install conntrack-tools
3. Compile

```bash
go build conntrack-monitor.go
```

## benchmark

* On ARM Allwinner A20 : `1500 events/s`
* On Intel® Core™ i5-4440 CPU: `18000 events/s`

## Use conntrack without sudo

```
sudo setcap cap_net_admin+ep /usr/sbin/conntrack
```