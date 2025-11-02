package marijan

import (
	"time"

	"go.uber.org/zap"
)

type ManagerOpt func(*Manager)

// set config source
func WithSource(source ConfigSource) func(*Manager) {
	return func(conf *Manager) {
		conf.source = source
	}
}

// set config url
func WithURL(url string) func(*Manager) {
	return func(conf *Manager) {
		conf.url = url
	}
}

// set interval
func WithInterval(interval time.Duration) func(*Manager) {
	return func(conf *Manager) {
		conf.interval = interval
	}
}

// set debug enabled
func WithDebug(enabled bool) func(*Manager) {
	return func(conf *Manager) {
		conf.debugEnabled = enabled
	}
}

// set logger
func WithLogger(logger *zap.Logger) func(*Manager) {
	return func(conf *Manager) {
		conf.zap = logger
	}
}
