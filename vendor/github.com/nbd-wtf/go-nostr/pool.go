package nostr

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
)

const (
	seenAlreadyDropTick = time.Minute
)

type SimplePool struct {
	Relays  *xsync.MapOf[string, *Relay]
	Context context.Context

	authHandler func(*Event) error
	cancel      context.CancelFunc

	// custom things not often used
	SignatureChecker func(Event) bool
}

type DirectedFilters struct {
	Filters
	Relay string
}

type IncomingEvent struct {
	*Event
	Relay *Relay
}

func (ie IncomingEvent) String() string {
	return fmt.Sprintf("[%s] >> %s", ie.Relay.URL, ie.Event)
}

type PoolOption interface {
	IsPoolOption()
	Apply(*SimplePool)
}

func NewSimplePool(ctx context.Context, opts ...PoolOption) *SimplePool {
	ctx, cancel := context.WithCancel(ctx)

	pool := &SimplePool{
		Relays: xsync.NewMapOf[string, *Relay](),

		Context: ctx,
		cancel:  cancel,
	}

	for _, opt := range opts {
		opt.Apply(pool)
	}

	return pool
}

// WithAuthHandler must be a function that signs the auth event when called.
// it will be called whenever any relay in the pool returns a `CLOSED` message
// with the "auth-required:" prefix, only once for each relay
type WithAuthHandler func(authEvent *Event) error

func (_ WithAuthHandler) IsPoolOption() {}
func (h WithAuthHandler) Apply(pool *SimplePool) {
	pool.authHandler = h
}

var _ PoolOption = (WithAuthHandler)(nil)

func (pool *SimplePool) EnsureRelay(url string) (*Relay, error) {
	nm := NormalizeURL(url)
	defer namedLock(nm)()

	relay, ok := pool.Relays.Load(nm)
	if ok && relay.IsConnected() {
		// already connected, unlock and return
		return relay, nil
	} else {
		var err error
		// we use this ctx here so when the pool dies everything dies
		ctx, cancel := context.WithTimeout(pool.Context, time.Second*15)
		defer cancel()

		opts := make([]RelayOption, 0, 1)
		if pool.SignatureChecker != nil {
			opts = append(opts, WithSignatureChecker(pool.SignatureChecker))
		}

		if relay, err = RelayConnect(ctx, nm, opts...); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		pool.Relays.Store(nm, relay)
		return relay, nil
	}
}

// SubMany opens a subscription with the given filters to multiple relays
// the subscriptions only end when the context is canceled
func (pool *SimplePool) SubMany(ctx context.Context, urls []string, filters Filters) chan IncomingEvent {
	return pool.subMany(ctx, urls, filters, true)
}

// SubManyNonUnique is like SubMany, but returns duplicate events if they come from different relays
func (pool *SimplePool) SubManyNonUnique(ctx context.Context, urls []string, filters Filters) chan IncomingEvent {
	return pool.subMany(ctx, urls, filters, false)
}

func (pool *SimplePool) subMany(ctx context.Context, urls []string, filters Filters, unique bool) chan IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // do this so `go vet` will stop complaining
	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[string, Timestamp]()
	ticker := time.NewTicker(seenAlreadyDropTick)

	eose := false

	pending := xsync.NewCounter()
	pending.Add(int64(len(urls)))
	for i, url := range urls {
		url = NormalizeURL(url)
		urls[i] = url
		if idx := slices.Index(urls, url); idx != i {
			// skip duplicate relays in the list
			continue
		}

		go func(nm string) {
			defer func() {
				pending.Dec()
				if pending.Value() == 0 {
					close(events)
				}
				cancel()
			}()

			hasAuthed := false
			interval := 3 * time.Second
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var sub *Subscription

				relay, err := pool.EnsureRelay(nm)
				if err != nil {
					goto reconnect
				}
				hasAuthed = false

			subscribe:
				sub, err = relay.Subscribe(ctx, filters)
				if err != nil {
					goto reconnect
				}

				go func() {
					<-sub.EndOfStoredEvents
					eose = true
				}()

				// reset interval when we get a good subscription
				interval = 3 * time.Second

				for {
					select {
					case evt, more := <-sub.Events:
						if !more {
							// this means the connection was closed for weird reasons, like the server shut down
							// so we will update the filters here to include only events seem from now on
							// and try to reconnect until we succeed
							now := Now()
							for i := range filters {
								filters[i].Since = &now
							}
							goto reconnect
						}
						if unique {
							if _, seen := seenAlready.LoadOrStore(evt.ID, evt.CreatedAt); seen {
								continue
							}
						}
						select {
						case events <- IncomingEvent{Event: evt, Relay: relay}:
						case <-ctx.Done():
						}
					case <-ticker.C:
						if eose {
							old := Timestamp(time.Now().Add(-seenAlreadyDropTick).Unix())
							seenAlready.Range(func(id string, value Timestamp) bool {
								if value < old {
									seenAlready.Delete(id)
								}
								return true
							})
						}
					case reason := <-sub.ClosedReason:
						if strings.HasPrefix(reason, "auth-required:") && pool.authHandler != nil && !hasAuthed {
							// relay is requesting auth. if we can we will perform auth and try again
							if err := relay.Auth(ctx, pool.authHandler); err == nil {
								hasAuthed = true // so we don't keep doing AUTH again and again
								goto subscribe
							}
						} else {
							log.Printf("CLOSED from %s: '%s'\n", nm, reason)
						}
						return
					case <-ctx.Done():
						return
					}
				}

			reconnect:
				// we will go back to the beginning of the loop and try to connect again and again
				// until the context is canceled
				time.Sleep(interval)
				interval = interval * 17 / 10 // the next time we try we will wait longer
			}
		}(url)
	}

	return events
}

// SubManyEose is like SubMany, but it stops subscriptions and closes the channel when gets a EOSE
func (pool *SimplePool) SubManyEose(ctx context.Context, urls []string, filters Filters) chan IncomingEvent {
	return pool.subManyEose(ctx, urls, filters, true)
}

// SubManyEoseNonUnique is like SubManyEose, but returns duplicate events if they come from different relays
func (pool *SimplePool) SubManyEoseNonUnique(ctx context.Context, urls []string, filters Filters) chan IncomingEvent {
	return pool.subManyEose(ctx, urls, filters, false)
}

func (pool *SimplePool) subManyEose(ctx context.Context, urls []string, filters Filters, unique bool) chan IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)

	events := make(chan IncomingEvent)
	seenAlready := xsync.NewMapOf[string, bool]()
	wg := sync.WaitGroup{}
	wg.Add(len(urls))

	go func() {
		// this will happen when all subscriptions get an eose (or when they die)
		wg.Wait()
		cancel()
		close(events)
	}()

	for _, url := range urls {
		go func(nm string) {
			defer wg.Done()

			relay, err := pool.EnsureRelay(nm)
			if err != nil {
				return
			}

			hasAuthed := false

		subscribe:
			sub, err := relay.Subscribe(ctx, filters)
			if sub == nil {
				debugLogf("error subscribing to %s with %v: %s", relay, filters, err)
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case <-sub.EndOfStoredEvents:
					return
				case reason := <-sub.ClosedReason:
					if strings.HasPrefix(reason, "auth-required:") && pool.authHandler != nil && !hasAuthed {
						// relay is requesting auth. if we can we will perform auth and try again
						err := relay.Auth(ctx, pool.authHandler)
						if err == nil {
							hasAuthed = true // so we don't keep doing AUTH again and again
							goto subscribe
						}
					}
					log.Printf("CLOSED from %s: '%s'\n", nm, reason)
					return
				case evt, more := <-sub.Events:
					if !more {
						return
					}

					if unique {
						if _, seen := seenAlready.LoadOrStore(evt.ID, true); seen {
							continue
						}
					}

					select {
					case events <- IncomingEvent{Event: evt, Relay: relay}:
					case <-ctx.Done():
						return
					}
				}
			}
		}(NormalizeURL(url))
	}

	return events
}

// QuerySingle returns the first event returned by the first relay, cancels everything else.
func (pool *SimplePool) QuerySingle(ctx context.Context, urls []string, filter Filter) *IncomingEvent {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for ievt := range pool.SubManyEose(ctx, urls, Filters{filter}) {
		return &ievt
	}
	return nil
}

func (pool *SimplePool) batchedSubMany(
	ctx context.Context,
	dfs []DirectedFilters,
	subFn func(context.Context, []string, Filters, bool) chan IncomingEvent,
) chan IncomingEvent {
	res := make(chan IncomingEvent)

	for _, df := range dfs {
		go func(df DirectedFilters) {
			for ie := range subFn(ctx, []string{df.Relay}, df.Filters, true) {
				res <- ie
			}
		}(df)
	}

	return res
}

// BatchedSubMany fires subscriptions only to specific relays, but batches them when they are the same.
func (pool *SimplePool) BatchedSubMany(ctx context.Context, dfs []DirectedFilters) chan IncomingEvent {
	return pool.batchedSubMany(ctx, dfs, pool.subMany)
}

// BatchedSubManyEose is like BatchedSubMany, but ends upon receiving EOSE from all relays.
func (pool *SimplePool) BatchedSubManyEose(ctx context.Context, dfs []DirectedFilters) chan IncomingEvent {
	return pool.batchedSubMany(ctx, dfs, pool.subManyEose)
}
