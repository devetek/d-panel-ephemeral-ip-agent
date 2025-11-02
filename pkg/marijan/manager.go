package marijan

//

/** Marijan (Managing and Routing Infrastructure for Joint Access Networks).

By using Marijan, you can manage Tukiran to connect to multiple tunnels by using remote config or local file config.
Marijan will help you to maintenance your tunnel connection always re-connect when it's disconnected.

Copyright (c) 2025 Devetek. All rights reserved.
*/

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/devetek/tukiran-dan-marijan/pkg/tukiran"
	"github.com/tkennon/ticker"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type ConfigSource string

const (
	ConfigSourceFile   ConfigSource = "file"
	ConfigSourceRemote ConfigSource = "remote"
)

type Manager struct {
	// ctx     context.Context
	debugEnabled bool
	source       ConfigSource
	url          string
	interval     time.Duration
	configs      []Config
	// wg      sync.WaitGroup
	zap *zap.Logger
}

type ConfigState string

const (
	ConfigStateActive   ConfigState = "active"
	ConfigStateInactive ConfigState = "inactive"
)

type Config struct {
	ID           string      `json:"id"`
	TunnelHost   string      `json:"tunnel_host"`
	TunnelPort   string      `json:"tunnel_port"`
	ListenerHost string      `json:"listener_host"`
	ListenerPort string      `json:"listener_port"`
	ServiceHost  string      `json:"service_host"`
	ServicePort  string      `json:"service_port"`
	State        ConfigState `json:"state,omitempty"`
	connection   *tukiran.TunnelForwarder
}

func NewManager(opts ...ManagerOpt) *Manager {
	conf := &Manager{
		debugEnabled: false,
		interval:     time.Minute,
		configs:      []Config{},
	}
	for _, opt := range opts {
		opt(conf)
	}
	return conf
}

func (manager *Manager) logger() *zap.Logger {
	if manager.zap == nil {
		return zap.NewNop()
	}

	return manager.zap.With(zap.Dict("module", zap.String("name", "marijan")))
}

func (manager *Manager) debug(message string) {
	if manager.debugEnabled {
		manager.logger().Info(message)
	}
}

func (manager *Manager) createNewConnection(config Config) *tukiran.TunnelForwarder {
	return tukiran.NewTunnelRemoteForwarder(
		tukiran.WithLogger(manager.zap),
		tukiran.WithConnectionID(config.ID),
		tukiran.WithTunnelHost(config.TunnelHost),
		tukiran.WithTunnelPort(config.TunnelPort),
		tukiran.WithTunnelAuthMethod(&ssh.ClientConfig{
			Auth:            []ssh.AuthMethod{},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}),
		tukiran.WithListenerHost(config.ListenerHost),
		tukiran.WithListenerPort(config.ListenerPort),
		tukiran.WithServiceHost(config.ServiceHost),
		tukiran.WithServicePort(config.ServicePort),
	)
}

func (manager *Manager) getNewConfig() ([]Config, error) {
	if manager.source == ConfigSourceFile {
		newConfigs, err := manager.getConfigFromFile()
		if err != nil {
			return nil, fmt.Errorf("Error fetching config from file: %v", err)
		}
		return newConfigs, nil
	} else if manager.source == ConfigSourceRemote {
		newConfigs, err := manager.getConfigFromRemote()
		if err != nil {
			return nil, fmt.Errorf("Error fetching config from remote: %v", err)
		}
		return newConfigs, nil
	}

	return nil, fmt.Errorf("Unknown config source: %s", manager.source)
}

// get current configs
func (manager *Manager) GetCurrentConfigs() []Config {
	return manager.configs
}

func (manager *Manager) Start() {
	newConfigs, err := manager.getNewConfig()
	if err != nil {
		manager.logger().Error("Error fetching config:", zap.Error(err))
		os.Exit(1)
	}

	// compare new config with old config
	for _, newConfig := range newConfigs {
		found := false
		for _, oldConfig := range manager.configs {
			if newConfig.ID == oldConfig.ID {
				found = true
				break
			}
		}
		if !found {
			// append new connection if not found and state is active
			if newConfig.State == ConfigStateActive {
				manager.configs = append(manager.configs, newConfig)
			}
		}
	}

	// attach connection to the config
	for index, config := range manager.configs {
		if config.State == ConfigStateActive {
			manager.configs[index].connection = manager.createNewConnection(config)
		}
	}

	for _, config := range manager.configs {
		if config.State == ConfigStateActive {
			go func() {
				err := config.connection.ListenAndServe()
				if err != nil {
					log.Println(err)
				}
			}()
		}
	}

	// running connection checker
	go manager.tick()
}

func (manager *Manager) StopAll() {
	for _, config := range manager.configs {
		if config.connection != nil {
			config.connection.Close()
		}
	}
}

func (manager *Manager) getConfigFromFile() ([]Config, error) {
	var configs []Config

	file, err := os.ReadFile(manager.url)
	if err != nil {
		return configs, fmt.Errorf("Error reading file: %v", err)
	}

	if err := json.Unmarshal(file, &configs); err != nil {
		return configs, fmt.Errorf("Error unmarshalling JSON: %v", err)
	}

	return configs, nil
}

func (manager *Manager) getConfigFromRemote() ([]Config, error) {
	var configs []Config

	// Create a custom HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a timeout for the request
	}

	// Make the GET request
	resp, err := client.Get(manager.url)
	if err != nil {
		return configs, fmt.Errorf("Error making GET request: %v", err)
	}
	defer resp.Body.Close() // Ensure the response body is closed

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return configs, fmt.Errorf("Received non-OK HTTP status: %s", resp.Status)
	}

	// Decode the JSON response into a struct
	if err := json.NewDecoder(resp.Body).Decode(&configs); err != nil {
		return configs, fmt.Errorf("Error decoding JSON response: %v", err)
	}

	return configs, nil
}

// Do check member is still connected or not
func (manager *Manager) tick() {
	manager.debug("Start ticker to maintenance tunnels connection.....")

	t := ticker.NewConstant(manager.interval)

	if err := t.Start(); err != nil {
		manager.logger().Panic("Failed to run maintainer connection", zap.Error(err))
	}
	defer t.Stop()

	for range t.C {
		newConfigs, err := manager.getNewConfig()
		if err != nil {
			manager.logger().Error("Error fetching config from remote", zap.Error(err))
		}

		// compare new config with old config
		for _, newConfig := range newConfigs {
			found := false
			for index, oldConfig := range manager.configs {
				if newConfig.ID == oldConfig.ID {
					found = true

					// update config based on remote config
					manager.configs[index].TunnelHost = newConfig.TunnelHost
					manager.configs[index].TunnelPort = newConfig.TunnelPort
					manager.configs[index].ListenerHost = newConfig.ListenerHost
					manager.configs[index].ListenerPort = newConfig.ListenerPort
					manager.configs[index].ServiceHost = newConfig.ServiceHost
					manager.configs[index].ServicePort = newConfig.ServicePort
					manager.configs[index].State = newConfig.State

					continue
				}
			}
			if !found {
				// add new config if state is active
				if newConfig.State == ConfigStateActive {
					manager.configs = append(manager.configs, newConfig)
				}
			}
		}

		// temporary array to store new config, copy array to tempArray
		for index, config := range manager.configs {
			// reconfigure connection if remote config is active
			if config.State == ConfigStateActive {
				if config.connection == nil {
					// set new connection
					manager.configs[index].connection = manager.createNewConnection(config)

					go func() {
						// start new connection!
						err := manager.configs[index].connection.ListenAndServe()
						if err != nil {
							log.Println(err)
						}
					}()
				}
				if config.connection != nil {
					manager.debug(fmt.Sprintf("Connection ID %s %s", config.connection.GetID(), config.connection.GetStateString()))

					if config.connection.GetState() == tukiran.Closed || config.connection.GetState() == tukiran.Idle {
						manager.debug(fmt.Sprintf("Connection ID %s is closed, try to reconnect", config.connection.GetID()))
						manager.configs[index].connection = manager.createNewConnection(config)

						go func() {
							// start new connection!
							err := manager.configs[index].connection.ListenAndServe()
							if err != nil {
								manager.logger().Error("Error reconnecting connection", zap.Error(err))
							}
						}()
					}
				}
			}

			// delete connection if remote config is inactive
			if config.State == ConfigStateInactive {
				if config.connection != nil {
					config.connection.Close()
					manager.configs[index].connection = nil
				}

				// delete index from new array
				manager.configs = append(manager.configs[:index], manager.configs[index+1:]...)
			}
		}
	}
}
