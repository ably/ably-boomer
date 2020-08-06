package perf

import "github.com/ably-forks/boomer"

// DefaultLocustReporter provides a default client for reporting to locust.
// In this case it uses the default boomer client.
var _defaultBoomerReporter LocustReporter = LocustReporter(
	&DefaultBoomerReporter{},
)

// LocustReporter reports success and failure events to locust.
type LocustReporter interface {
	RecordSuccess(
		requestType string,
		name string,
		responseTime int64,
		responseLength int64,
	)

	RecordFailure(
		requestType string,
		name string,
		responseTime int64,
		exception string,
	)
}

// DefaultLocustReporter gets the locust reporter global default
func DefaultLocustReporter() LocustReporter {
	return LocustReporter(_defaultBoomerReporter)
}

// DefaultBoomerReporter reports to locust with the default boomer client
type DefaultBoomerReporter struct{}

// RecordSuccess reports a success event to locust.
func (*DefaultBoomerReporter) RecordSuccess(
	requestType string,
	name string,
	responseTime int64,
	responseLength int64,
) {
	boomer.RecordSuccess(
		requestType,
		name,
		responseTime,
		responseLength,
	)
}

// RecordFailure reports a failure to locust.
func (*DefaultBoomerReporter) RecordFailure(
	requestType string,
	name string,
	responseTime int64,
	exception string,
) {
	boomer.RecordFailure(
		requestType,
		name,
		responseTime,
		exception,
	)
}
