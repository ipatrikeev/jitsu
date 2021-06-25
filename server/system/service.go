package system

import (
	"encoding/json"
	"github.com/jitsucom/jitsu/server/resources"
	"sync"
	"time"
)

const serviceName = "system"

var reloadEvery = 5 * time.Second

//Configuration is used for system endpoint in Configurator and for Server redirect when configured
//provides current authorization configuration and amount of registered users
type Configuration struct {
	Authorization          string `json:"authorization"`
	Users                  bool   `json:"users"`
	SMTP                   bool   `json:"smtp"`
	SelfHosted             bool   `json:"selfhosted"`
	SupportWidget          bool   `json:"support_widget"`
	DefaultS3Bucket        bool   `json:"default_s3_bucket"`
	SupportTrackingDomains bool   `json:"support_tracking_domains"`
	TelemetryUsageDisabled bool   `json:"telemetry_usage_disabled"`
	ShowBecomeUser         bool   `json:"show_become_user"`
	DockerHubID            string `json:"docker_hub_id"`
}

//Service is a reloadable service for keeping system configuration
type Service struct {
	mutex         *sync.RWMutex
	configuration *Configuration

	configured bool
}

//NewService returns configured Service and call resources.Watcher()
func NewService(url string) *Service {
	if url == "" {
		return &Service{}
	}

	service := &Service{configured: true, mutex: &sync.RWMutex{}}
	resources.Watch(serviceName, url, resources.LoadFromHTTP, service.reInit, reloadEvery)
	return service
}

//reInit initializes system configuration
//it is used for keeping actual configuration for configurator redirect
func (s *Service) reInit(payload []byte) {
	c := &Configuration{}
	err := json.Unmarshal(payload, c)
	if err != nil {
		return
	}

	s.mutex.Lock()
	s.configuration = c
	s.mutex.Unlock()
}

func (s *Service) ShouldBeRedirected() bool {
	return s.configured && !s.configuration.Users
}

func (s *Service) IsConfigured() bool {
	return s.configured
}
