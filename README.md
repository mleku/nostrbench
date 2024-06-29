# nostrbench

benchmarks of Go implementations of nostr things

```
goos: linux
goarch: amd64
pkg: github.com/mleku/nostrbench
cpu: AMD Ryzen 5 PRO 4650G with Radeon Graphics
BenchmarkEncoding/nodl.Marshal-12                1000000              1163 ns/op               0 B/op          0 allocs/op
BenchmarkEncoding/event2.MarshalJSON-12           140084              7148 ns/op            3631 B/op         41 allocs/op
BenchmarkEncoding/event2.EventToBinary-12        1784491             659.0 ns/op             286 B/op          3 allocs/op
BenchmarkEncoding/easyjson.Marshal-12             594602              2021 ns/op            1629 B/op          6 allocs/op
BenchmarkEncoding/gob.Encode-12                   185842              6141 ns/op            4877 B/op         43 allocs/op
BenchmarkEncoding/binary.Marshal-12                94168             10931 ns/op           73789 B/op          1 allocs/op
BenchmarkDecoding/nodl.Unmarshal-12               476563              2939 ns/op            1357 B/op         15 allocs/op
BenchmarkDecoding/event2.UnmarshalJSON-12         152085              8196 ns/op            2257 B/op         22 allocs/op
BenchmarkDecoding/event2.BinaryToEvent-12         856695              1382 ns/op            1205 B/op         15 allocs/op
BenchmarkDecoding/easyjson.Unmarshal-12           578265              2313 ns/op            2322 B/op         17 allocs/op
BenchmarkDecoding/gob.Decode-12                    54600             23911 ns/op           10642 B/op        248 allocs/op
BenchmarkDecoding/binary.Unmarshal-12            1484220             813.4 ns/op             769 B/op         10 allocs/op
```