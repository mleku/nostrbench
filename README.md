# nostrbench

benchmarks of Go implementations of nostr things

```
goos: linux
goarch: amd64
pkg: github.com/mleku/nostrbench
cpu: AMD Ryzen 5 PRO 4650G with Radeon Graphics
BenchmarkEncodingEasyJSON-12              687991              1790 ns/op            1631 B/op          6 allocs/op
BenchmarkDecodingEasyJSON-12              676521              1777 ns/op            1413 B/op         16 allocs/op
BenchmarkEncodingGob-12                   184216              5982 ns/op            4872 B/op         43 allocs/op
BenchmarkDecodingGob-12                    55344             21500 ns/op           10061 B/op        236 allocs/op
BenchmarkEncodingFiatjafBinary-12          99108             11383 ns/op           73789 B/op          1 allocs/op
BenchmarkDecodingFiatjafBinary-12        1370816               863.9 ns/op           769 B/op         10 allocs/op
BenchmarkMlekuMarshalJSON-12             1000000              1149 ns/op               0 B/op          0 allocs/op
BenchmarkMlekuUnmarshalJSON-12            609270              1951 ns/op             684 B/op         13 allocs/op
BenchmarkMlekuMarshalBinary-12           2982892               402.2 ns/op           600 B/op          2 allocs/op
BenchmarkMlekuUnmarshalBinary-12         1571326               767.5 ns/op          1096 B/op         14 allocs/op
```