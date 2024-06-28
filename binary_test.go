package nostrbench

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/gob"
	"encoding/json"
	"testing"

	"github.com/mailru/easyjson"
	event3 "github.com/mleku/nodl/pkg/event"
	"github.com/nbd-wtf/go-nostr"
	. "github.com/nbd-wtf/go-nostr/binary"
	event2 "mleku.net/nostr/event"
)

//go:embed out.jsonl
var rawEvents []byte

var normalEvents []string

var evts = make([]*event3.T, 0, 10000)

func init() {
	// decompress embedded rawEvents
	buf := bytes.NewBuffer(rawEvents)
	scanner := bufio.NewScanner(buf)
	scanBuffer := make([]byte, 0, 1_000_000)
	scanner.Buffer(scanBuffer, 999_999)
	for scanner.Scan() {
		s := string(scanner.Bytes())
		normalEvents = append(normalEvents, s)
	}
	scanner = bufio.NewScanner(bytes.NewBuffer(rawEvents))
	var ev *event3.T
	var err error
	for scanner.Scan() {
		b := scanner.Bytes()
		if ev, _, err = event3.Unmarshal(b); err != nil {
			panic(err)
		}
		evts = append(evts, ev)
	}

}

func BenchmarkEncoding(b *testing.B) {
	events := make([]*nostr.Event, len(normalEvents))
	events2 := make([]*event2.T, len(normalEvents))
	binaryEvents := make([]*Event, len(normalEvents))
	for i, jevt := range normalEvents {
		evt := &nostr.Event{}
		json.Unmarshal([]byte(jevt), evt)
		events[i] = evt
		binaryEvents[i] = BinaryEvent(evt)
		evt2 := &event2.T{}
		if err := json.Unmarshal([]byte(normalEvents[i]), evt2); err != nil {
			panic(err)
		}
		events2[i] = evt2
	}

	b.Run("nodl.Marshal", func(bb *testing.B) {
		var i int
		var out []byte
		bb.ReportAllocs()
		var counter int
		for i = 0; i < bb.N; i++ {
			out = evts[counter].Marshal(out)
			out = out[:0]
			counter++
			if counter != len(evts) {
				counter = 0
			}
		}
	})

	b.Run("event2.MarshalJSON", func(b *testing.B) {
		b.ReportAllocs()
		var counter int
		for i := 0; i < b.N; i++ {
			_, _ = events2[counter].MarshalJSON()
			counter++
			if counter == len(events2) {
				counter = 0
			}
		}
	})

	b.Run("event2.EventToBinary", func(b *testing.B) {
		b.ReportAllocs()
		var maxSize int
		for _, evt := range events2 {
			m := event2.EstimateSize(evt)
			if m > maxSize {
				maxSize = m
			}
		}
		evtBuf := event2.NewWriteBuffer(maxSize)
		var counter int
		for i := 0; i < b.N; i++ {
			_ = evtBuf.WriteEvent(events2[counter])
			evtBuf.Buf = evtBuf.Buf[:0]
			counter++
			if counter == len(events2) {
				counter = 0
			}
		}
	})

	b.Run("easyjson.Marshal", func(b *testing.B) {
		b.ReportAllocs()
		var counter int
		for i := 0; i < b.N; i++ {
			easyjson.Marshal(events[counter])
			counter++
			if counter == len(events) {
				counter = 0
			}
		}
	})

	b.Run("gob.Encode", func(b *testing.B) {
		b.ReportAllocs()
		var counter int
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			gob.NewEncoder(&buf).Encode(events[counter])
			counter++
			if counter == len(events) {
				counter = 0
			}
			_ = buf.Bytes()
		}
	})

	b.Run("binary.Marshal", func(b *testing.B) {
		b.ReportAllocs()
		var counter int
		for i := 0; i < b.N; i++ {
			Marshal(events[counter])
			counter++
			if counter == len(events) {
				counter = 0
			}
		}
	})

	// this does not comprehend rawEvents over 64kb in size
	// b.Run("binary.MarshalBinary", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		for _, bevt := range binaryEvents {
	// 			MarshalBinary(bevt)
	// 		}
	// 	}
	// })

}

func BenchmarkDecoding(b *testing.B) {
	events := make([][]byte, len(normalEvents))
	gevents := make([][]byte, len(normalEvents))
	bevents := make([][]byte, len(normalEvents))
	for i, jevt := range normalEvents {
		evt := &nostr.Event{}
		json.Unmarshal([]byte(jevt), evt)
		bevt, _ := Marshal(evt)
		events[i] = bevt

		var buf bytes.Buffer
		gob.NewEncoder(&buf).Encode(evt)
		gevents[i] = buf.Bytes()

		evt2 := &event2.T{}
		json.Unmarshal([]byte(jevt), evt2)
		bevt2, _ := event2.EventToBinary(evt2)
		bevents[i] = bevt2
	}

	b.Run("nodl.Unmarshal", func(bb *testing.B) {
		bb.ReportAllocs()
		scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
		var ev *event3.T
		var err error
		for i := 0; i < bb.N; i++ {
			if !scanner.Scan() {
				scanner = bufio.NewScanner(bytes.NewBuffer(rawEvents))
				scanner.Scan()
			}
			b := scanner.Bytes()
			if ev, _, err = event3.Unmarshal(b); err != nil {
				bb.Fatal(err)
			}
			evts = append(evts, ev)
		}
	})

	b.Run("event2.UnmarshalJSON", func(b *testing.B) {
		b.ReportAllocs()
		var ev event2.T
		var counter int
		for i := 0; i < b.N; i++ {
			if err := json.Unmarshal([]byte(normalEvents[counter]),
				&ev); err != nil {
			}
			counter++
			if counter == len(normalEvents) {
				counter = 0
			}
		}
	})

	b.Run("event2.BinaryToEvent", func(b *testing.B) {
		b.ReportAllocs()
		var counter int
		for i := 0; i < b.N; i++ {
			event2.BinaryToEvent(bevents[counter])
			counter++
			if counter == len(bevents) {
				counter = 0
			}
		}
	})

	b.Run("easyjson.Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		var counter int
		for i := 0; i < b.N; i++ {
			evt := &nostr.Event{}
			err := easyjson.Unmarshal([]byte(normalEvents[counter]), evt)
			if err != nil {
				b.Fatalf("failed to unmarshal: %s", err)
			}
			counter++
			if counter == len(normalEvents) {
				counter = 0
			}
		}
	})

	b.Run("gob.Decode", func(b *testing.B) {
		b.ReportAllocs()
		var counter int
		for i := 0; i < b.N; i++ {
			evt := &nostr.Event{}
			buf := bytes.NewBuffer(gevents[counter])
			evt = &nostr.Event{}
			gob.NewDecoder(buf).Decode(evt)
			counter++
			if counter == len(gevents) {
				counter = 0
			}
		}
	})

	b.Run("binary.Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		var counter int
		for i := 0; i < b.N; i++ {
			evt := &nostr.Event{}
			err := Unmarshal(events[counter], evt)
			if err != nil {
				b.Fatalf("failed to unmarshal: %s", err)
			}
			counter++
			if counter == len(events) {
				counter = 0
			}
		}
	})

	// b.Run("binary.UnmarshalBinary", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		for _, bevt := range rawEvents {
	// 			evt := &Event{}
	// 			err := UnmarshalBinary(bevt, evt)
	// 			if err != nil {
	// 				b.Fatalf("failed to unmarshal: %s", err)
	// 			}
	// 		}
	// 	}
	// })

	// b.Run("easyjson.Unmarshal+sig", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		for _, nevt := range normalEvents {
	// 			evt := &nostr.Event{}
	// 			err := easyjson.Unmarshal([]byte(nevt), evt)
	// 			if err != nil {
	// 				b.Fatalf("failed to unmarshal: %s", err)
	// 			}
	// 			evt.CheckSignature()
	// 		}
	// 	}
	// })

	// b.Run("binary.Unmarshal+sig", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		for _, bevt := range rawEvents {
	// 			evt := &nostr.Event{}
	// 			err := Unmarshal(bevt, evt)
	// 			if err != nil {
	// 				b.Fatalf("failed to unmarshal: %s", err)
	// 			}
	// 			evt.CheckSignature()
	// 		}
	// 	}
	// })
}
