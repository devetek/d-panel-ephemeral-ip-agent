package main

import "time"

type ManagerOpt func(*Manager)

func WithSource(source ConfigSource) func(*Manager) {
	return func(conf *Manager) {
		conf.source = source
	}
}

func WithURL(url string) func(*Manager) {
	return func(conf *Manager) {
		conf.url = url
	}
}

func WithInterval(interval time.Duration) func(*Manager) {
	return func(conf *Manager) {
		conf.interval = interval
	}
}
