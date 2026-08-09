package dispatcher

import (
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/features/stats"
	"github.com/xtls/xray-core/transport/internet/stat"
)

// combinedCounter forwards Add to a primary counter and an optional extra
// counter. It is used to make an inbound connection's wire-level byte counter
// (Counted by CounterConnection at the outermost layer, including TLS / vision
// overhead, and kept accurate even when splice or vision direct-copy bypass the
// wrapper) also increment a per-user counter, so user traffic can be billed on
// the actual bytes the server transfers on the link.
type combinedCounter struct {
	primary stats.Counter
	extra   stats.Counter
}

func (c *combinedCounter) Add(v int64) int64 {
	if c.primary != nil {
		c.primary.Add(v)
	}
	if c.extra != nil {
		c.extra.Add(v)
	}
	return v
}

// Value/Set delegate to primary so that the inbound named counter semantics
// (the original CounterConnection counter) stay intact; only Add is mirrored to
// the per-user extra counter.
func (c *combinedCounter) Value() int64 {
	if c.primary != nil {
		return c.primary.Value()
	}
	if c.extra != nil {
		return c.extra.Value()
	}
	return 0
}

func (c *combinedCounter) Set(v int64) int64 {
	if c.primary != nil {
		return c.primary.Set(v)
	}
	return 0
}

// attachUserWireStats wires the inbound connection's wire-level byte counters
// (the CounterConnection at the outermost layer, which counts every byte
// actually transferred on the link, including TLS / vision overhead, and which
// splice / vision direct-copy keep accurate by compensating with Add) to also
// increment per-user "wire" traffic counters. This lets user traffic stats
// reflect the actual bytes the server consumes for that connection.
//
// Only bytes transferred after this call are attributed to the user, so the
// small TLS / protocol handshake traffic is excluded. It is a no-op when user
// stats are disabled or the inbound conn is not stat-wrapped.
func attachUserWireStats(sessionInbound *session.Inbound, sm stats.Manager, email string, uplink, downlink bool) {
	if sessionInbound == nil || sessionInbound.Conn == nil {
		return
	}
	cc, ok := sessionInbound.Conn.(*stat.CounterConnection)
	if !ok {
		return
	}
	var readExtra, writeExtra stats.Counter
	if uplink {
		if c, _ := sm.GetOrRegisterCounter("user>>>" + email + ">>>traffic>>>wire>>>uplink"); c != nil {
			readExtra = c
		}
	}
	if downlink {
		if c, _ := sm.GetOrRegisterCounter("user>>>" + email + ">>>traffic>>>wire>>>downlink"); c != nil {
			writeExtra = c
		}
	}
	if readExtra == nil && writeExtra == nil {
		return
	}
	cc.ReadCounter = &combinedCounter{primary: cc.ReadCounter, extra: readExtra}
	cc.WriteCounter = &combinedCounter{primary: cc.WriteCounter, extra: writeExtra}
}
