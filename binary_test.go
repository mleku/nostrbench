package nostrbench

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/gob"
	"encoding/json"
	"os"
	"testing"

	"github.com/mailru/easyjson"
	mleku "github.com/mleku/nodl/pkg/codec/event"
	"github.com/mleku/nodl/pkg/util/lol"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/binary"
)

//go:embed out.jsonl
var rawEvents []byte

func BenchmarkEncodingEasyJSON(b *testing.B) {
	b.StopTimer()
	evts := make([]*nostr.Event, 0, 9999)
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	scanBuf := make([]byte, 1_000_000)
	scanner.Buffer(scanBuf, len(scanBuf))
	for scanner.Scan() {
		jevt := scanner.Bytes()
		evt := &nostr.Event{}
		_ = json.Unmarshal(jevt, evt)
		evts = append(evts, evt)
	}
	b.ReportAllocs()
	b.StartTimer()
	var counter int
	for i := 0; i < b.N; i++ {
		easyjson.Marshal(evts[counter])
		counter++
		if counter == len(evts) {
			counter = 0
		}
	}
}

func BenchmarkDecodingEasyJSON(b *testing.B) {
	b.StopTimer()
	normalEvents := make([][]byte, 0, 9999)
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	scanBuf := make([]byte, 1_000_000)
	scanner.Buffer(scanBuf, len(scanBuf))
	for scanner.Scan() {
		sb := scanner.Bytes()
		eb := make([]byte, len(sb))
		copy(eb, sb)
		normalEvents = append(normalEvents, eb)
	}
	b.ReportAllocs()
	b.StartTimer()
	var counter int
	for i := 0; i < b.N; i++ {
		evt := &nostr.Event{}
		err := easyjson.Unmarshal(normalEvents[counter], evt)
		if err != nil {
			b.Fatalf("failed to unmarshal: %s", err)
		}
		counter++
		if counter == len(normalEvents) {
			counter = 0
		}
	}
}

func BenchmarkEncodingGob(b *testing.B) {
	b.StopTimer()
	evts := make([]*nostr.Event, 0, 9999)
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	scanBuf := make([]byte, 1_000_000)
	scanner.Buffer(scanBuf, len(scanBuf))
	for scanner.Scan() {
		jevt := scanner.Bytes()
		evt := &nostr.Event{}
		json.Unmarshal(jevt, evt)
		evts = append(evts, evt)
	}
	b.ReportAllocs()
	b.StartTimer()
	var counter int
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		gob.NewEncoder(&buf).Encode(evts[counter])
		counter++
		if counter == len(evts) {
			counter = 0
		}
		_ = buf.Bytes()
	}
}

func BenchmarkDecodingGob(b *testing.B) {
	b.StopTimer()
	normalEvents := make([][]byte, 0, 9999)
	gevents := make([][]byte, 0, cap(normalEvents))
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	scanBuf := make([]byte, 1_000_000)
	scanner.Buffer(scanBuf, len(scanBuf))
	for scanner.Scan() {
		sb := scanner.Bytes()
		evt := &nostr.Event{}
		_ = json.Unmarshal(sb, evt)
		var buf bytes.Buffer
		_ = gob.NewEncoder(&buf).Encode(evt)
		gevents = append(gevents, buf.Bytes())
	}
	b.ReportAllocs()
	b.StartTimer()
	var counter int
	evt := &nostr.Event{}
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(gevents[counter])
		_ = gob.NewDecoder(buf).Decode(evt)
		counter++
		if counter == len(gevents) {
			counter = 0
		}
	}
}

func BenchmarkEncodingFiatjafBinary(b *testing.B) {
	b.StopTimer()
	evts := make([]*nostr.Event, 0, 9999)
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	scanBuf := make([]byte, 1_000_000)
	scanner.Buffer(scanBuf, len(scanBuf))
	for scanner.Scan() {
		jevt := scanner.Bytes()
		evt := &nostr.Event{}
		_ = json.Unmarshal(jevt, evt)
		evts = append(evts, evt)
	}
	b.ReportAllocs()
	b.StartTimer()
	var counter int
	for i := 0; i < b.N; i++ {
		_, _ = binary.Marshal(evts[counter])
		counter++
		if counter == len(evts) {
			counter = 0
		}
	}
}

func BenchmarkDecodingFiatjafBinary(b *testing.B) {
	b.StopTimer()
	events := make([][]byte, 0, 9999)
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	scanBuf := make([]byte, 1_000_000)
	scanner.Buffer(scanBuf, len(scanBuf))
	for scanner.Scan() {
		sb := scanner.Bytes()
		evt := &nostr.Event{}
		_ = json.Unmarshal(sb, evt)
		bevt, _ := binary.Marshal(evt)
		events = append(events, bevt)
	}
	b.ReportAllocs()
	b.StartTimer()
	var counter int
	for i := 0; i < b.N; i++ {
		evt := &nostr.Event{}
		err := binary.Unmarshal(events[counter], evt)
		if err != nil {
			b.Fatalf("failed to unmarshal: %s", err)
		}
		counter++
		if counter == len(events) {
			counter = 0
		}
	}
}

type (
	B = []byte
	S = string
	I = int
	E = error
)

var (
	log, chk, errorf = lol.New(os.Stderr)
	equals           = bytes.Equal
)

func BenchmarkMlekuMarshalJSON(bb *testing.B) {
	bb.StopTimer()
	var i int
	var out B
	var err error
	evts := make([]*mleku.T, 0, 9999)
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	buf := make(B, 1_000_000)
	scanner.Buffer(buf, len(buf))
	for scanner.Scan() {
		b := scanner.Bytes()
		ea := mleku.New()
		b, err = ea.UnmarshalJSON(b)
		if err != nil {
			continue
		}
		evts = append(evts, ea)
	}
	bb.ReportAllocs()
	var counter int
	out = out[:0]
	bb.StartTimer()
	for i = 0; i < bb.N; i++ {
		out, _ = evts[counter].MarshalJSON(out)
		out = out[:0]
		counter++
		if counter != len(evts) {
			counter = 0
		}
	}
}

func BenchmarkMlekuUnmarshalJSON(bb *testing.B) {
	var i int
	var err error
	evts := make([]*mleku.T, 9999)
	bb.ReportAllocs()
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	buf := make(B, 1_000_000)
	scanner.Buffer(buf, len(buf))
	var counter I
	for i = 0; i < bb.N; i++ {
		if !scanner.Scan() {
			scanner = bufio.NewScanner(bytes.NewBuffer(rawEvents))
			scanner.Scan()
		}
		b := scanner.Bytes()
		ea := mleku.New()
		b, err = ea.UnmarshalJSON(b)
		if err != nil {
			continue
		}
		evts[counter] = ea
		b = b[:0]
		if counter > 9999 {
			counter = 0
		}
	}
}

func BenchmarkMlekuMarshalBinary(bb *testing.B) {
	bb.StopTimer()
	var i int
	var out B
	var err error
	evts := make([]*mleku.T, 0, 9999)
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	buf := make(B, 1_000_000)
	scanner.Buffer(buf, len(buf))
	for scanner.Scan() {
		b := scanner.Bytes()
		ea := mleku.New()
		b, err = ea.UnmarshalJSON(b)
		if err != nil {
			continue
		}
		evts = append(evts, ea)
	}
	var counter int
	out = out[:0]
	bb.ReportAllocs()
	bb.StartTimer()
	for i = 0; i < bb.N; i++ {
		out, _ = evts[counter].MarshalBinary(out)
		out = out[:0]
		counter++
		if counter != len(evts) {
			counter = 0
		}
	}
}

func BenchmarkMlekuUnmarshalBinary(bb *testing.B) {
	bb.StopTimer()
	var i int
	var out B
	var err error
	evts := make([]B, 0, 9999)
	scanner := bufio.NewScanner(bytes.NewBuffer(rawEvents))
	buf := make(B, 1_000_000)
	scanner.Buffer(buf, len(buf))
	for scanner.Scan() {
		b := scanner.Bytes()
		ea := mleku.New()
		b, err = ea.UnmarshalJSON(b)
		if err != nil {
			continue
		}
		out = make(B, len(b))
		out, _ = ea.MarshalBinary(out)
		evts = append(evts, out)
	}
	bb.ReportAllocs()
	var counter int
	bb.StartTimer()
	ev := mleku.New()
	for i = 0; i < bb.N; i++ {
		l := len(evts[counter])
		b := make(B, l)
		copy(b, evts[counter])
		b, _ = ev.UnmarshalBinary(b)
		out = out[:0]
		counter++
		if counter != len(evts) {
			counter = 0
		}
	}
}
