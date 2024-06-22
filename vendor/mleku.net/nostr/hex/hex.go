package hex

import "encoding/hex"

var Enc = hex.EncodeToString
var EncBytes = hex.Encode
var Dec = hex.DecodeString
var DecBytes = hex.Decode
var EncAppend = hex.AppendEncode
var DecAppend = hex.AppendDecode
