package circuitbreaker

import (
	"log/slog"
	"time"

	"github.com/sony/gobreaker/v2"
)

func CreateCircuitBreaker(toolName string) *gobreaker.CircuitBreaker[any] {

	var st gobreaker.Settings
	st.Name = toolName             // identifier of the circuit breaker
	st.MaxRequests = 1             // max no of requests in HALF OPEN state
	st.Interval = 60 * time.Second // rolling window for counting failures in the closed state
	st.Timeout = 5 * time.Second   // duration after which circuit breaker moves from OPEN to HALF OPEN state

	st.ReadyToTrip = func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= 3 // circuit breaker will be placed in OPEN state if this condition is true
	}

	st.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {

		slog.Info("circuit breaker state changed",
			"NAME", name,
			"FROM", from.String(),
			"TO", to.String(),
		)
	}

	return gobreaker.NewCircuitBreaker[any](st)
}
