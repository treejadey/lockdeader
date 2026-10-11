package install

import "time"

// Pace sets how a download's progress bar shine moves: slowly with pauses, at
// a normal speed with pauses, or quickly without stopping.
type Pace int

const (
	PaceNormal Pace = iota
	PaceSlow        // under about 17 Mbps
	PaceFast        // over about 60 Mbps
)

func (p Pace) String() string {
	switch p {
	case PaceSlow:
		return "slow"
	case PaceFast:
		return "fast"
	default:
		return "normal"
	}
}

// In bytes per second.
const (
	slowBelow = 17e6 / 8
	fastAbove = 60e6 / 8
	// A download has to get this much past a boundary to cross back over
	// it, so a speed that hovers around one doesn't flip the pace back and
	// forth.
	paceMargin = 0.1
)

const (
	speedWindow = 2 * time.Second
	// A download keeps the normal pace until it's been going this long,
	// rather than being judged on its first few packets.
	speedWarmUp = 500 * time.Millisecond
)

// The zero value is ready to use.
type SpeedMeter struct {
	samples []speedSample
	pace    Pace
}

type speedSample struct {
	at       time.Time
	received int64
}

// received is the total that has arrived by then, not what arrived since the
// last call.
func (m *SpeedMeter) Add(at time.Time, received int64) {
	m.samples = append(m.samples, speedSample{at, received})

	// Keep one sample from before the window, so the window is always covered.
	drop := 0

	for drop+1 < len(m.samples) && at.Sub(m.samples[drop+1].at) >= speedWindow {
		drop++
	}

	m.samples = m.samples[drop:]

	if at.Sub(m.samples[0].at) >= speedWarmUp {
		m.pace = m.paceAt(m.BytesPerSecond())
	}
}

// BytesPerSecond is 0 until there's enough to tell.
func (m *SpeedMeter) BytesPerSecond() float64 {
	if len(m.samples) < 2 {
		return 0
	}

	first, last := m.samples[0], m.samples[len(m.samples)-1]
	elapsed := last.at.Sub(first.at).Seconds()
	if elapsed <= 0 {
		return 0
	}

	return float64(last.received-first.received) / elapsed
}

func (m *SpeedMeter) Pace() Pace {
	return m.pace
}

func (m *SpeedMeter) paceAt(speed float64) Pace {
	slow, fast := float64(slowBelow), float64(fastAbove)

	switch m.pace {
	case PaceSlow:
		slow *= 1 + paceMargin
	case PaceFast:
		fast *= 1 - paceMargin
	}

	switch {
	case speed < slow:
		return PaceSlow
	case speed > fast:
		return PaceFast
	default:
		return PaceNormal
	}
}
