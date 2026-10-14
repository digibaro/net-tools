package mtr

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Hop accumulates statistics for one TTL from `mtr --raw` output.
type Hop struct {
	Hop    int       `json:"hop"`
	IP     string    `json:"ip"`
	Host   string    `json:"host"`
	Sent   int       `json:"sent"`
	Recv   int       `json:"recv"`
	Loss   float64   `json:"loss"`
	LastMs float64   `json:"lastMs"`
	AvgMs  float64   `json:"avgMs"`
	BestMs float64   `json:"bestMs"`
	Worst  float64   `json:"worstMs"`
	StdDev float64   `json:"stddevMs"`
	RTTMs  []float64 `json:"rttMs"`
	// ASN is filled in when the AS lookup is available.
	ASN *uint32 `json:"asn"`
}

// Parser consumes `mtr --raw` lines:
//
//	x <pos> <seq>      probe sent
//	h <pos> <ip>       host at hop
//	d <pos> <name>     reverse DNS name
//	p <pos> <usec> <seq> reply
//
// Positions are 0-based; other record types are ignored.
type Parser struct {
	hops map[int]*Hop
	// MaxRTTs bounds kept samples per hop.
	MaxRTTs int
}

// NewParser creates an empty parser.
func NewParser() *Parser { return &Parser{hops: map[int]*Hop{}, MaxRTTs: 50} }

func (p *Parser) hop(pos int) *Hop {
	h, ok := p.hops[pos]
	if !ok {
		h = &Hop{Hop: pos + 1, RTTMs: []float64{}}
		p.hops[pos] = h
	}
	return h
}

// Line parses one line; it returns the affected hop number (1-based) or 0.
func (p *Parser) Line(line string) int {
	f := strings.Fields(line)
	if len(f) < 3 {
		return 0
	}
	pos, err := strconv.Atoi(f[1])
	if err != nil || pos < 0 || pos > 255 {
		return 0
	}
	h := p.hop(pos)
	switch f[0] {
	case "x":
		h.Sent++
	case "h":
		if h.IP == "" {
			h.IP = f[2]
		}
	case "d":
		if h.Host == "" {
			h.Host = f[2]
		}
	case "p":
		usec, err := strconv.ParseFloat(f[2], 64)
		if err != nil {
			return 0
		}
		h.Recv++
		ms := math.Round(usec) / 1000
		h.LastMs = ms
		if len(h.RTTMs) < p.MaxRTTs {
			h.RTTMs = append(h.RTTMs, ms)
		}
	default:
		return 0
	}
	h.recompute()
	return h.Hop
}

func (h *Hop) recompute() {
	sent := max(h.Sent, h.Recv)
	if sent > 0 {
		h.Loss = math.Round(float64(sent-h.Recv)/float64(sent)*1000) / 10
	}
	if len(h.RTTMs) == 0 {
		return
	}
	sum := 0.0
	for _, v := range h.RTTMs {
		sum += v
	}
	avg := sum / float64(len(h.RTTMs))
	variance := 0.0
	for _, v := range h.RTTMs {
		variance += (v - avg) * (v - avg)
	}
	h.AvgMs = math.Round(avg*1000) / 1000
	h.BestMs, h.Worst = slices.Min(h.RTTMs), slices.Max(h.RTTMs)
	h.StdDev = math.Round(math.Sqrt(variance/float64(len(h.RTTMs)))*1000) / 1000
}

// Hops returns hops in order, trimmed after the last hop that answered (mtr probes beyond the
// destination are dropped).
func (p *Parser) Hops() []Hop {
	last := -1
	for pos, h := range p.hops {
		if h.IP != "" && pos > last {
			last = pos
		}
	}
	out := make([]Hop, 0, last+1)
	for pos := 0; pos <= last; pos++ {
		if h, ok := p.hops[pos]; ok {
			out = append(out, *h)
		} else {
			out = append(out, Hop{Hop: pos + 1, RTTMs: []float64{}})
		}
	}
	return out
}

// Get returns a copy of one hop (1-based).
func (p *Parser) Get(hop int) Hop {
	if h, ok := p.hops[hop-1]; ok {
		return *h
	}
	return Hop{Hop: hop, RTTMs: []float64{}}
}

// throttle limits per-hop update events.
type throttle struct {
	last  map[int]time.Time
	every time.Duration
}

func (t *throttle) allow(hop int, now time.Time) bool {
	if now.Sub(t.last[hop]) < t.every {
		return false
	}
	t.last[hop] = now
	return true
}
