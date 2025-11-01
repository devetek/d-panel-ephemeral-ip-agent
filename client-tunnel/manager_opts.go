package main

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
