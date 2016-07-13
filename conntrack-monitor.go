package main

import (
	"github.com/aarnaud/conntrack-monitor/conntrack"
	"fmt"
)

var flow_messages = make(chan conntrack.FlowRecord, 128)

func printFlow(flowChan <-chan conntrack.FlowRecord){
	for 0 == 0 {
		f := <- flowChan
		fmt.Println(f.String())
	}

}

func main(){
	go printFlow(flow_messages)
	conntrack.Watch(flow_messages)

}