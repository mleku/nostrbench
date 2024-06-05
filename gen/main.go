package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"mleku.net/nostr/event"
)

const MaxMessageSize = 1_000_000
const GatherMessages = 10000

func main() {
	inFile, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer inFile.Close()
	outFile, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	defer outFile.Close()
	buf := make([]byte, 0, MaxMessageSize)
	scanner := bufio.NewScanner(inFile)
	scanner.Buffer(buf, 1_000_000)
	var count int
	for scanner.Scan() && count < GatherMessages {
		b := scanner.Bytes()
		rb := make([]byte, len(b))
		copy(rb, b)
		ev := &event.T{}
		if err = json.Unmarshal(b, ev); err != nil {
			continue
		}
		// ensure the event is valid
		if _, err = event.EventToBinary(ev); err != nil {
			continue
		}
		fmt.Fprintln(outFile, ev.ToObject().String())
		count++
	}

	// w, err := zstd.NewWriter(outFile)
	// if err != nil {
	// 	panic(err)
	// }
	// _, err = w.ReadFrom(outBuf)
	// if err != nil {
	// 	panic(err)
	// }
}
