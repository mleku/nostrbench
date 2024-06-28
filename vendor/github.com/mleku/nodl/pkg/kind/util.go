package kind

import (
	"os"

	"github.com/mleku/nodl/pkg/lol"
)

type (
	B = []byte
	S = string
)

var (
	log, chk, errorf = lol.New(os.Stderr)
)
