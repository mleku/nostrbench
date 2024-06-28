# nostrbench

benchmarks of Go implementations of nostr things

```
goos: linux
goarch: amd64
pkg: github.com/mleku/nostrbench
cpu: AMD Ryzen 5 PRO 4650G with Radeon Graphics
BenchmarkEncoding/nodl.Marshal-12                1203306             992.3 ns/op               0 B/op          0 allocs/op
BenchmarkEncoding/event2.MarshalJSON-12           142533              7135 ns/op            3581 B/op         40 allocs/op
BenchmarkEncoding/event2.EventToBinary-12        1528389             807.0 ns/op             350 B/op          4 allocs/op
BenchmarkEncoding/easyjson.Marshal-12             642684              1849 ns/op            1628 B/op          6 allocs/op
BenchmarkEncoding/gob.Encode-12                   181173              6117 ns/op            4867 B/op         43 allocs/op
BenchmarkEncoding/binary.Marshal-12                98074             10816 ns/op           73789 B/op          1 allocs/op
BenchmarkDecoding/nodl.Unmarshal-12               359467              2847 ns/op            1359 B/op         15 allocs/op
BenchmarkDecoding/event2.UnmarshalJSON-12         153836              8195 ns/op            2290 B/op         22 allocs/op
BenchmarkDecoding/event2.BinaryToEvent-12         882427              1416 ns/op            1308 B/op         16 allocs/op
BenchmarkDecoding/easyjson.Unmarshal-12           567627              2292 ns/op            2322 B/op         17 allocs/op
BenchmarkDecoding/gob.Decode-12                    54482             23727 ns/op           10643 B/op        248 allocs/op
BenchmarkDecoding/binary.Unmarshal-12            1453920             820.3 ns/op             769 B/op         10 allocs/op
```