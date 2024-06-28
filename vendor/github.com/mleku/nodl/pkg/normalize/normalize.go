package normalize

import (
	"bytes"
	"net/url"
	"strings"

	"github.com/mleku/nodl/pkg/ints"
)

// URL normalizes the URL
//
// - Adds wss:// to addresses without a port, or with 443 that have no protocol
// prefix
//
// - Adds ws:// to addresses with any other port
//
// - Converts http/s to ws/s
func URL(u B) (b B) {
	if len(u) == 0 {
		return nil
	}
	u = bytes.TrimSpace(u)
	u = bytes.ToLower(u)

	// if address has a port number, we can probably assume it is insecure
	// websocket as most public or production relays have a domain name and a
	// well known port 80 or 443 and thus no port number.
	//
	// if a protocol prefix is present, we assume it is already complete.
	// Converting http/s to websocket equivalent will be done later anyway.
	if bytes.Contains(u, []byte(":")) &&
		!(bytes.HasPrefix(u, B("http://")) ||
			bytes.HasPrefix(u, B("https://")) ||
			bytes.HasPrefix(u, B("ws://")) ||
			bytes.HasPrefix(u, B("wss://"))) {
		split := bytes.Split(u, B(":"))
		if len(split) != 2 {
			log.D.F("Error: more than one ':' in URL: '%s'", u)
			// this is a malformed URL if it has more than one ":", return empty
			// since this function does not return an error explicitly.
			return
		}

		port, _, err := ints.ExtractInt64FromByteString(split[1])
		if chk.E(err) {
			log.D.F("Error normalizing URL '%s': %s", u, err)
			// again, without an error we must return nil
			return
		}
		if port > 65535 {
			log.D.F("Port on address %d: greater than maximum 65535", port)
			return
		}
		// if the port is explicitly set to 443 we assume it is wss:// and drop
		// the port.
		if port == 443 {
			u = append(B("wss://"), split[0]...)
		} else {
			u = append(B("ws://"), u...)
		}
	}

	// if prefix isn't specified as http/s or websocket, assume secure websocket
	// and add wss prefix (this is the most common).
	if !(bytes.HasPrefix(u, B("http://")) ||
		bytes.HasPrefix(u, B("https://")) ||
		bytes.HasPrefix(u, B("ws://")) ||
		bytes.HasPrefix(u, B("wss://"))) {
		u = append(B("wss://"), u...)
	}
	var err error
	var p *url.URL
	p, err = url.Parse(string(u))
	if err != nil {
		return
	}
	// convert http/s to ws/s
	switch p.Scheme {
	case "https":
		p.Scheme = "wss"
	case "http":
		p.Scheme = "ws"
	}
	// remove trailing path slash
	p.Path = S(bytes.TrimRight(B(p.Path), "/"))
	return B(p.String())
}

// Reason takes a string message that is to be sent in an `OK` or `CLOSED`
// command and prefixes it with "<prefix>: " if it doesn't already have an
// acceptable prefix.
func Reason(reason string, prefix string) string {
	if idx := strings.Index(reason,
		": "); idx == -1 || strings.IndexByte(reason[0:idx], ' ') != -1 {
		return prefix + ": " + reason
	}
	return reason
}
