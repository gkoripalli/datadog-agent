package netlink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerDefaultState(t *testing.T) {
	breaker := newTestBreaker(100)
	assert.False(t, breaker.IsOpen())
}

func TestCircuitBreakerRemainsClosed(t *testing.T) {
	const maxEventRate = 100
	breaker := newTestBreaker(maxEventRate)

	// The code below simulates a constant rate of events/s during 5 min
	now := time.Now()
	deadline := now.Add(5 * time.Minute)
	for now.Before(deadline) {
		breaker.Tick(maxEventRate)
		breaker.update(now)
		now = now.Add(time.Second)
	}

	//  The circuit breaker should remain closed
	assert.False(t, breaker.IsOpen())
}

func TestCircuitBreakerSupportsBursts(t *testing.T) {
	const maxEventRate = 100
	breaker := newTestBreaker(maxEventRate)

	// Let's assume the circuit-breaker has been runing with 80% of the max allowed rate
	now := time.Now()
	breaker.Tick(int(float64(maxEventRate) * 0.8))
	breaker.update(now)
	assert.False(t, breaker.IsOpen())

	// Since we smoothen the event rate using EWMA we shouldn't trip immediately after
	// going above the max rate
	now = now.Add(time.Second)
	deadline := now.Add(3 * time.Second)
	for now.Before(deadline) {
		breaker.Tick(maxEventRate + 5)
		breaker.update(now)
		now = now.Add(time.Second)
	}
	assert.False(t, breaker.IsOpen())

	// However after some time it surely should trip the circuit
	deadline = now.Add(30 * time.Second)
	for now.Before(deadline) {
		breaker.Tick(maxEventRate + 10)
		breaker.update(now)
		now = now.Add(time.Second)
	}
	assert.True(t, breaker.IsOpen())
}

func TestCircuitBreakerReset(t *testing.T) {
	const maxEventRate = 100
	breaker := newTestBreaker(maxEventRate)

	breaker.Tick(maxEventRate * 2)
	breaker.update(time.Now())
	assert.True(t, breaker.IsOpen())

	breaker.Reset()
	assert.False(t, breaker.IsOpen())
}

func TestStartAboveThreshold(t *testing.T) {
	const maxEventRate = 100
	breaker := newTestBreaker(maxEventRate)

	// If our first measurement is above threshold we don't amortize it
	// and trip the circuit.
	breaker.Tick(maxEventRate + 1)
	breaker.update(time.Now())
	assert.True(t, breaker.IsOpen())
}

func newTestBreaker(maxEventRate int) *CircuitBreaker {
	c := &CircuitBreaker{maxEventsPerSec: int64(maxEventRate)}
	c.Reset()
	return c
}
