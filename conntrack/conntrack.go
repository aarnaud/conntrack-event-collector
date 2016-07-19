package conntrack

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const ConntrackBufferSize int = 30000000
const conntrackFlowRegex = `\[(?P<timestamp>\d+\.\d+)(?:\s+)?\]\s+\[(?P<type>\w+)\]\s+(?P<protoname3>\w+)\s+(?P<protonum3>\d+)\s+(?P<protoname4>\w+)\s+`
const conntrackOriginalRegex = `(?:.+)src=(?P<originalSrc>\S+)\s+dst=(?P<originalDst>\S+)\s+(?:sport=(?P<originalSport>\d+)\s+dport=(?P<originalDport>\d+)\s+)?(?:packets=(?P<originalPackets>\d+)\s+bytes=(?P<originalBytes>\d+))?`
const conntrackReplyRegex = `(?:.+)src=(?P<replySrc>\S+)\s+dst=(?P<replyDst>\S+)\s+(?:sport=(?P<replySport>\d+)\s+dport=(?P<replyDport>\d+)\s+)?(?:packets=(?P<replyPackets>\d+)\s+bytes=(?P<replyBytes>\d+))?`

var conntrackRegexCompiled *regexp.Regexp

func Watch(eventType []string, flowChan chan Flow) {
	regexBuffer := bytes.Buffer{}
	regexBuffer.WriteString(conntrackFlowRegex)
	regexBuffer.WriteString(conntrackOriginalRegex)
	regexBuffer.WriteString(conntrackReplyRegex)
	conntrackRegexCompiled = regexp.MustCompile(regexBuffer.String())

	for {
		runConntrack(eventType, flowChan)
	}
}

func runConntrack(eventType []string, flowChan chan Flow) {
	cmd := exec.Command("conntrack", "--buffer-size", strconv.Itoa(ConntrackBufferSize), "-E", "-e", strings.Join(eventType, ","), "-o", "timestamp,extended,id")
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Print("Error conntrack:", err)
	}

	go func() {
		stderr := bufio.NewReader(stderrPipe)
		for {
			line, _, err := stderr.ReadLine()
			if err != nil {
				log.Print("Error stderr readline:", err)
				break
			}
			log.Println(string(line))
		}
	}()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Print("Error conntrack:", err)
	}
	stdout := bufio.NewReader(stdoutPipe)
	log.Println("Starting conntrack...")
	cmd.Start()

	var buffer bytes.Buffer
	for {
		frag, isPrefix, err := stdout.ReadLine()
		if err != nil {
			log.Print("Error stdout readline:", err)
			break
		}
		buffer.Write(frag)
		if !isPrefix {
			line := buffer.String()
			go func() {
				flow := flowParse(line)
				flowChan <- flow
			}()
			buffer.Reset()
		}

	}
}

func flowParse(str string) Flow {
	var flow Flow = Flow{}
	flow.Original = Meta{}
	flow.Original.Layer3 = Layer3{}
	flow.Original.Layer4 = Layer4{}
	flow.Original.Counter = Counter{}
	flow.Reply = Meta{}
	flow.Reply.Layer3 = Layer3{}
	flow.Reply.Layer4 = Layer4{}
	flow.Reply.Counter = Counter{}

	result := conntrackRegexCompiled.FindStringSubmatch(str)
	names := conntrackRegexCompiled.SubexpNames()
	for i, match := range result {
		if i != 0 {
			switch names[i] {
			case "timestamp":
				flow.Timestamp, _ = strconv.Atoi(match)
				break
			case "type":
				flow.Type = match
				break
			case "protoname3":
				flow.Original.Layer3.Protoname = match
				flow.Reply.Layer3.Protoname = match
				break
			case "protonum3":
				flow.Original.Layer3.Protonum, _ = strconv.Atoi(match)
				flow.Reply.Layer3.Protonum, _ = strconv.Atoi(match)
				break
			case "protoname4":
				flow.Original.Layer4.Protoname = match
				flow.Reply.Layer4.Protoname = match
				break
			case "protonum4":
				flow.Original.Layer4.Protonum, _ = strconv.Atoi(match)
				flow.Reply.Layer4.Protonum, _ = strconv.Atoi(match)
				break
			case "originalSrc":
				flow.Original.Layer3.Src = net.ParseIP(match)
				break
			case "originalDst":
				flow.Original.Layer3.Dst = net.ParseIP(match)
				break
			case "originalSport":
				flow.Original.Layer4.Sport, _ = strconv.Atoi(match)
				break
			case "originalDport":
				flow.Original.Layer4.Dport, _ = strconv.Atoi(match)
				break
			case "originalPackets":
				flow.Original.Counter.Packets, _ = strconv.Atoi(match)
				break
			case "originalBytes":
				flow.Original.Counter.Bytes, _ = strconv.Atoi(match)
				break
			case "replySrc":
				flow.Reply.Layer3.Src = net.ParseIP(match)
				break
			case "replyDst":
				flow.Reply.Layer3.Dst = net.ParseIP(match)
				break
			case "replySport":
				flow.Reply.Layer4.Sport, _ = strconv.Atoi(match)
				break
			case "replyDport":
				flow.Reply.Layer4.Dport, _ = strconv.Atoi(match)
				break
			case "replyPackets":
				flow.Reply.Counter.Packets, _ = strconv.Atoi(match)
				break
			case "replyBytes":
				flow.Reply.Counter.Bytes, _ = strconv.Atoi(match)
				break
			}

		}
	}
	if len(result) == 0 {
		log.Print("parse error of: %s", str)
	}

	return flow
}
