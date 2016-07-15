package main

import (
	"fmt"
	"github.com/aarnaud/go-conntrack-monitor/conntrack"
	"log"
	"time"
)

var count int = 0

var flow_messages = make(chan conntrack.Flow, 128)

func printFlow(flowChan <-chan conntrack.Flow) {
	for 0 == 0 {
		flow := <-flowChan
		if flow.Type != "" {
			//fmt.Printf("#+%v\n", flow)
			count++
		}

	}

}

func main() {
	go func() {
		for {
			time.Sleep(60 * time.Second)
			log.Println(fmt.Sprintf("average %d events/s", count/60))
			count = 0
		}
	}()
	go printFlow(flow_messages)
	conntrack.Watch([]string{"NEW", "DESTROY"}, flow_messages)

}
