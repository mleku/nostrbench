package nostrbench

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"

	_ "embed"

	"github.com/mailru/easyjson"
	"github.com/nbd-wtf/go-nostr"
	. "github.com/nbd-wtf/go-nostr/binary"
	event2 "mleku.net/nostr/event"
)

//go:embed out.jsonl
var events []byte
var normalEvents []string

func init() {
	// decompress embedded events
	buf := bytes.NewBuffer(events)
	scanner := bufio.NewScanner(buf)
	scanBuffer := make([]byte, 0, 1_000_000)
	scanner.Buffer(scanBuffer, 999_999)
	for scanner.Scan() {
		s := string(scanner.Bytes())
		normalEvents = append(normalEvents, s)
	}
}

func BenchmarkBinaryEncoding(b *testing.B) {
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

	b.Run("event2.MarshalJSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events2 {
				_, _ = evt.MarshalJSON()
			}
		}
	})

	b.Run("event2.EventToBinary", func(b *testing.B) {
		var maxSize int
		for _, evt := range events2 {
			m := event2.EstimateSize(evt)
			if m > maxSize {
				maxSize = m
			}
		}
		evtBuf := event2.NewWriteBuffer(maxSize)
		for i := 0; i < b.N; i++ {
			for _, evt := range events2 {
				_ = evtBuf.WriteEvent(evt)
				evtBuf.Buf = evtBuf.Buf[:0]
			}
		}
	})

	b.Run("easyjson.Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				easyjson.Marshal(evt)
			}
		}
	})

	b.Run("gob.Encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				var buf bytes.Buffer
				gob.NewEncoder(&buf).Encode(evt)
				_ = buf.Bytes()
			}
		}
	})

	b.Run("binary.Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, evt := range events {
				Marshal(evt)
			}
		}
	})

	// this does not comprehend events over 64kb in size
	// b.Run("binary.MarshalBinary", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		for _, bevt := range binaryEvents {
	// 			MarshalBinary(bevt)
	// 		}
	// 	}
	// })

}

func BenchmarkBinaryDecoding(b *testing.B) {
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

	b.Run("event2.BinaryToEvent", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range bevents {
				event2.BinaryToEvent(bevt)
			}
		}
	})

	b.Run("easyjson.Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, jevt := range normalEvents {
				evt := &nostr.Event{}
				err := easyjson.Unmarshal([]byte(jevt), evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
			}
		}
	})

	b.Run("gob.Decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, gevt := range gevents {
				evt := &nostr.Event{}
				buf := bytes.NewBuffer(gevt)
				evt = &nostr.Event{}
				gob.NewDecoder(buf).Decode(evt)
			}
		}
	})

	b.Run("binary.Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &nostr.Event{}
				err := Unmarshal(bevt, evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
			}
		}
	})

	b.Run("binary.UnmarshalBinary", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &Event{}
				err := UnmarshalBinary(bevt, evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
			}
		}
	})

	b.Run("easyjson.Unmarshal+sig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, nevt := range normalEvents {
				evt := &nostr.Event{}
				err := easyjson.Unmarshal([]byte(nevt), evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
				evt.CheckSignature()
			}
		}
	})

	b.Run("binary.Unmarshal+sig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, bevt := range events {
				evt := &nostr.Event{}
				err := Unmarshal(bevt, evt)
				if err != nil {
					b.Fatalf("failed to unmarshal: %s", err)
				}
				evt.CheckSignature()
			}
		}
	})
}
