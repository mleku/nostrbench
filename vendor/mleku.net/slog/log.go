package slog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gookit/color"
	"mleku.net/atomic"
)

var l = GetStd()

func GetStd() (ll *Log) {
	ll, _ = New(os.Stdout)
	return
}

func init() {
	switch strings.ToUpper(os.Getenv("GODEBUG")) {
	case "1", "TRUE", "ON":
		SetLogLevel(Debug)
		l.D.Ln("printing logs at this level and lower")
	case "INFO":
		SetLogLevel(Info)
	case "DEBUG":
		SetLogLevel(Debug)
		l.D.Ln("printing logs at this level and lower")
	case "TRACE":
		SetLogLevel(Trace)
		l.T.Ln("printing logs at this level and lower")
	case "WARN":
		SetLogLevel(Warn)
	case "ERROR":
		SetLogLevel(Error)
	case "FATAL":
		SetLogLevel(Fatal)
	case "0", "OFF", "FALSE":
		SetLogLevel(Off)
	default:
		SetLogLevel(Info)
	}

}

const (
	Off = iota
	Fatal
	Error
	Warn
	Info
	Debug
	Trace
)

type (
	// LevelPrinter defines a set of terminal printing primitives that output with
	// extra data, time, log logLevelList, and code location

	// Ln prints lists of interfaces with spaces in between
	Ln func(a ...interface{})
	// F prints like fmt.Println surrounded by log details
	F func(format string, a ...interface{})
	// S prints a spew.Sdump for an interface slice
	S func(a ...interface{})
	// C accepts a function so that the extra computation can be avoided if it is
	// not being viewed
	C func(closure func() string)
	// Chk is a shortcut for printing if there is an error, or returning true
	Chk func(e error) bool
	// Err is a pass-through function that uses fmt.Errorf to construct an error
	// and returns the error after printing it to the log
	Err          func(format string, a ...interface{}) error
	LevelPrinter struct {
		Ln
		F
		S
		C
		Chk
		Err
	}
	LevelSpec struct {
		ID        int
		Name      string
		Colorizer func(a ...interface{}) string
	}

	// Entry is a log entry to be printed as json to the log file
	Entry struct {
		Time         time.Time
		Level        string
		Package      string
		CodeLocation string
		Text         string
	}
)

var (
	// sep is just a convenient shortcut for this very longwinded expression
	sep          = string(os.PathSeparator)
	currentLevel = atomic.NewInt32(Info)
	// writer can be swapped out for any io.*writer* that you want to use instead of
	// stdout.
	writer io.Writer = os.Stderr
	// LevelSpecs specifies the id, string name and color-printing function
	LevelSpecs = []LevelSpec{
		{Off, "off", color.Bit24(0, 0, 0, false).Sprint},
		{Fatal, "ftl", color.Bit24(128, 0, 0, false).Sprint},
		{Error, "err", color.Bit24(255, 0, 0, false).Sprint},
		{Warn, "wrn", color.Bit24(0, 255, 0, false).Sprint},
		{Info, "inf", color.Bit24(255, 255, 0, false).Sprint},
		{Debug, "dbg", color.Bit24(0, 128, 255, false).Sprint},
		{Trace, "trc", color.Bit24(128, 0, 255, false).Sprint},
	}
)

// Log is a set of log printers for the various Level items.
type Log struct {
	F, E, W, I, D, T LevelPrinter
}

type Check struct {
	F, E, W, I, D, T Chk
}

func New(writer io.Writer) (l *Log, c *Check) {
	var nilLp = LevelPrinter{
		Ln: func(a ...interface{}) {
			fmt.Fprintln(writer, a...)
			fmt.Fprintln(writer, GetLoc(2))

		},
		F: func(format string, a ...interface{}) {
			fmt.Fprintf(writer, format, a...)
			fmt.Fprintln(writer)
			fmt.Fprintln(writer, GetLoc(2))
		},
		S: func(a ...interface{}) {
			spew.Fdump(writer, a...)
			fmt.Fprintln(writer, GetLoc(2))
		},
		C: func(closure func() string) {
			fmt.Fprintln(writer, closure())
			fmt.Fprintln(writer)
			fmt.Fprintln(writer, GetLoc(2))
		},
		Chk: func(e error) bool {
			if e != nil {
				fmt.Fprintln(writer, e.Error())
				fmt.Fprintln(writer)
				fmt.Fprintln(writer, GetLoc(2))
				return true
			}
			return false
		},
		Err: func(format string, a ...interface{}) error {
			fmt.Fprintf(writer, fmt.Sprintf(format, a...))
			fmt.Fprintln(writer)
			fmt.Fprintf(writer, GetLoc(2))
			return fmt.Errorf(format, a...)
		},
	}
	l = &Log{
		F: nilLp,
		E: nilLp,
		W: nilLp,
		I: nilLp,
		D: nilLp,
		T: nilLp,
	}
	c = &Check{
		F: l.F.Chk,
		E: l.E.Chk,
		W: l.W.Chk,
		I: l.I.Chk,
		D: l.D.Chk,
		T: l.T.Chk,
	}
	return
}

// New returns a set of LevelPrinter with their subsystem preloaded
//
// this copies the interface of stdlib log but we don't respect the settings
// because a logger without timestamps is retarded
func Newp(wr io.Writer) (l *Log, c *Check) {
	writer = wr
	l = &Log{
		F: _getOnePrinter(Fatal),
		E: _getOnePrinter(Error),
		W: _getOnePrinter(Warn),
		I: _getOnePrinter(Info),
		D: _getOnePrinter(Debug),
		T: _getOnePrinter(Trace),
	}
	c = &Check{
		F: l.F.Chk,
		E: l.E.Chk,
		W: l.W.Chk,
		I: l.I.Chk,
		D: l.D.Chk,
		T: l.T.Chk,
	}
	return
}

func _getOnePrinter(level int32) LevelPrinter {
	return LevelPrinter{
		Ln:  _ln(level),
		F:   _f(level),
		S:   _s(level),
		C:   _c(level),
		Chk: _chk(level),
		Err: _err(level),
	}
}

// SetLogLevel sets the log level via a string, which can be truncated down to
// one character, similar to nmcli's argument processor, as the first letter is
// unique. This could be used with a linter to make larger command sets.
func SetLogLevel(l int) {
	currentLevel.Store(int32(l))
}

func GetLogLevel() (l int) {
	return int(currentLevel.Load())
}

// UnixNanoAsFloat e
func UnixNanoAsFloat() (s string) {
	timeText := fmt.Sprint(time.Now().UnixNano())
	lt := len(timeText)
	lb := lt + 1
	var timeBytes = make([]byte, lb)
	copy(timeBytes[lb-9:lb], timeText[lt-9:lt])
	timeBytes[lb-10] = '.'
	lb -= 10
	lt -= 9
	copy(timeBytes[:lb], timeText[:lt])
	return string(timeBytes)
}

func _ln(level int32) func(a ...interface{}) {
	return func(a ...interface{}) {
		if level <= currentLevel.Load() {
			printer := fmt.Sprintf
			fmt.Fprint(
				writer,
				printer(
					"%s %s %s %v\n",
					color.Bit24(0, 128, 255, false).Sprint(
						UnixNanoAsFloat(),
					),
					LevelSpecs[level].Colorizer(
						LevelSpecs[level].Name,
					),
					joinStrings(" ", a...),
					GetLoc(2),
				),
			)
		}
	}
}

func _f(level int32) func(format string, a ...interface{}) {
	return func(format string, a ...interface{}) {
		if level <= currentLevel.Load() {
			printer := fmt.Sprintf
			fmt.Fprint(
				writer,
				printer(
					"%s %s %s %v\n",
					color.Bit24(0, 128, 255, false).Sprint(
						UnixNanoAsFloat(),
					),
					LevelSpecs[level].Colorizer(
						LevelSpecs[level].Name,
					),
					fmt.Sprintf(format, a...),
					GetLoc(2),
				),
			)
		}
	}
}

func _s(level int32) func(a ...interface{}) {
	return func(a ...interface{}) {
		if level <= currentLevel.Load() {
			printer := fmt.Sprintf
			fmt.Fprint(
				writer,
				printer(
					"%s %s %s %s\n",
					color.Bit24(0, 128, 255, false).Sprint(
						UnixNanoAsFloat(),
					),
					LevelSpecs[level].Colorizer(
						LevelSpecs[level].Name,
					),
					fmt.Sprint(
						"\n\n"+spew.Sdump(a),
						"\n",
					),
					GetLoc(2),
				),
			)
		}
	}
}

func _c(level int32) func(closure func() string) {
	return func(closure func() string) {
		if level <= currentLevel.Load() {
			printer := fmt.Sprintf
			fmt.Fprint(
				writer,
				printer(
					"%s %s %s %v\n",
					color.Bit24(0, 128, 255, false).Sprint(
						UnixNanoAsFloat(),
					),
					LevelSpecs[level].Colorizer(
						LevelSpecs[level].Name,
					),
					closure(),
					GetLoc(2),
				),
			)
		}
	}
}

func _chk(level int32) func(e error) bool {
	return func(e error) bool {
		if level <= currentLevel.Load() {
			loc := GetLoc(2)
			if e != nil {
				printer := fmt.Sprintf
				fmt.Fprint(
					writer,
					printer(
						"%s %s %s %s\n",
						color.Bit24(0, 128, 255, false).Sprint(
							UnixNanoAsFloat(),
						),
						LevelSpecs[level].Colorizer(
							LevelSpecs[level].Name,
						),
						e.Error()+" ",
						loc,
					),
				)
			}
		}
		return e != nil
	}
}

func _err(level int32) func(format string, a ...interface{}) error {
	return func(format string, a ...interface{}) (err error) {
		err = fmt.Errorf(format, a...)
		if level <= currentLevel.Load() {
			loc := GetLoc(2)
			printer := fmt.Sprintf
			fmt.Fprint(
				writer,
				printer(
					"%s %s %s %s\n",
					color.Bit24(0, 128, 255, false).Sprint(
						UnixNanoAsFloat(),
					),
					LevelSpecs[level].Colorizer(
						LevelSpecs[level].Name,
					),
					err.Error()+" ",
					loc,
				),
			)
		}
		return
	}
}

// joinStrings constructs a string from an slice of interface same as Println but
// without the terminal newline
func joinStrings(sep string, a ...interface{}) (o string) {
	for i := range a {
		o += fmt.Sprint(a[i])
		if i < len(a)-1 {
			o += sep
		}
	}
	return
}

func GetLoc(skip int) (output string) {
	_, file, line, _ := runtime.Caller(skip)
	output = color.Bit24(0, 128, 255, false).Sprint(
		file, ":", line,
	)
	return
}
