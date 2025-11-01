package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tkennon/ticker"
	"golang.org/x/crypto/ssh"
)

type ConfigSource string

const (
	ConfigSourceFile   ConfigSource = "file"
	ConfigSourceRemote ConfigSource = "remote"
)

type Manager struct {
	// ctx     context.Context
	source  ConfigSource
	url     string
	configs []Config
}

type ConfigState string

const (
	ConfigStateActive  ConfigState = "active"
	ConfigStateDeleted ConfigState = "deleted"
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
	connection   *TunnelForwarder
}

func NewManager(opts ...ManagerOpt) *Manager {
	conf := &Manager{
		configs: []Config{},
	}
	for _, opt := range opts {
		opt(conf)
	}
	return conf
}

func (manager *Manager) createNewConnection(config Config) *TunnelForwarder {
	return NewTunnelRemoteForwarder(
		WithConnectionID(config.ID),
		WithTunnelHost(config.TunnelHost),
		WithTunnelPort(config.TunnelPort),
		WithTunnelAuthMethod(&ssh.ClientConfig{
			Auth:            []ssh.AuthMethod{},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}),
		WithListenerHost(config.ListenerHost),
		WithListenerPort(config.ListenerPort),
		WithServiceHost(config.ServiceHost),
		WithServicePort(config.ServicePort),
	)
}

func (manager *Manager) init() error {
	if manager.source == ConfigSourceFile {
		configs, err := manager.getConfigFromFile()
		if err != nil {
			return fmt.Errorf("Error fetching config from file: %v", err)
		}
		manager.configs = configs
	} else if manager.source == ConfigSourceRemote {
		configs, err := manager.getConfigFromRemote()
		if err != nil {
			return fmt.Errorf("Error fetching config from remote: %v", err)
		}
		manager.configs = configs
	}
	return nil
}

func (manager *Manager) Start() {
	err := manager.init()
	if err != nil {
		log.Println(err)
		return
	}

	// compare new config with old config
	for _, newConfig := range manager.configs {
		found := false
		for _, oldConfig := range manager.configs {
			if newConfig.ID == oldConfig.ID {
				found = true
				break
			}
		}
		if !found {
			// create new connection
			manager.configs = append(manager.configs, newConfig)
		}
	}

	// attach connection to the config
	for index, config := range manager.configs {
		manager.configs[index].connection = manager.createNewConnection(config)
	}

	for _, config := range manager.configs {
		go func() {
			err := config.connection.ListenAndServe()
			if err != nil {
				log.Println(err)
			}
		}()
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
	file, err := os.ReadFile(manager.url)
	if err != nil {
		return nil, fmt.Errorf("Error reading file: %v", err)
	}

	var configs []Config
	if err := json.Unmarshal(file, &configs); err != nil {
		return nil, fmt.Errorf("Error unmarshalling JSON: %v", err)
	}

	return configs, nil
}

func (manager *Manager) getConfigFromRemote() ([]Config, error) {

	var configs []Config
	url := manager.url

	// Create a custom HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a timeout for the request
	}

	// Make the GET request
	resp, err := client.Get(url)
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
	log.Println("Start ticker to maintenance tunnels.....")
	t := ticker.NewConstant(5 * time.Second)

	if err := t.Start(); err != nil {
		log.Panicln("Failed to run maintainer connection", err)
	}
	defer t.Stop()

	for range t.C {
		newConfigs, err := manager.getConfigFromFile()
		if err != nil {
			log.Println(err)
		}

		// compare new config with old config
		for _, newConfig := range newConfigs {
			found := false
			for _, oldConfig := range manager.configs {
				if newConfig.ID == oldConfig.ID {
					found = true
					continue
				}
			}
			if !found {
				// add new config
				manager.configs = append(manager.configs, newConfig)
			}
		}

		for index, config := range manager.configs {
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
				if config.connection.getState() == 3 || config.connection.getState() == 0 {
					log.Printf("Connection %s is closed, try to reconnect", config.connection.getID())
					manager.configs[index].connection = manager.createNewConnection(config)

					go func() {
						// start new connection!
						err := manager.configs[index].connection.ListenAndServe()
						if err != nil {
							log.Println(err)
						}
					}()
				}
			}
		}
	}

}
