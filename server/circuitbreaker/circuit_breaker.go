package circuitbreaker

import (
	"log/slog"
	"time"

	"github.com/sony/gobreaker/v2"
)

func CreateCircuitBreaker(toolName string) *gobreaker.CircuitBreaker[any] {

	var st gobreaker.Settings
	st.Name = toolName
	st.MaxRequests = 8
	st.Interval = 10 * time.Second

	st.ReadyToTrip = func(counts gobreaker.Counts) bool {
		failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
		return counts.ConsecutiveFailures >= 3 && failureRatio >= 0.4
	}

	st.Timeout = time.Millisecond

	st.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {

		if to == gobreaker.StateOpen {
			slog.Error("State Open")
		}

		if from == gobreaker.StateOpen && to == gobreaker.StateHalfOpen {
			slog.Info("Going from Open to Half Open state")
		}

		if from == gobreaker.StateHalfOpen && to == gobreaker.StateClosed {
			slog.Info("Going from Half Open to Closed state")
		}
	}

	cb := gobreaker.NewCircuitBreaker[any](st)

	return cb
}
