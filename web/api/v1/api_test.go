package v1

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/SotirisAlfonsos/chaos-bot/common/service"
	"github.com/SotirisAlfonsos/chaos-bot/proto"
	"github.com/SotirisAlfonsos/chaos-master/chaoslogger"
	"github.com/go-kit/kit/log"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheckService_Check(t *testing.T) {
	hcs := &HealthCheckService{}
	resp, err := hcs.Check(context.TODO(), &proto.HealthCheckRequest{})

	if err != nil {
		t.Fatalf("Error on Health Check request. err=%s", err)
	}

	assert.Equal(t, proto.HealthCheckResponse_SERVING, resp.Status)
}

// === End to end testing ===
func TestServiceManager_e2e(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}

	logger := getLogger()
	myCache := cache.New(0, 0)
	serviceName := "simple"
	hostname, _ := os.Hostname()

	sm := &ServiceManager{Cache: myCache, Logger: logger}
	stratM := &StrategyManager{myCache, logger}

	startService(sm, serviceName, t, hostname)
	startServiceFail(sm, serviceName, t)
	recoverServiceEmpty(stratM, serviceName, t)
	stopService(sm, serviceName, t, hostname)
	recoverService(stratM, serviceName, t, hostname)
	stopService(sm, serviceName, t, hostname)
}

func startService(sm *ServiceManager, serviceName string, t *testing.T, hostname string) {
	resp, err := sm.Start(context.TODO(), &proto.ServiceRequest{Name: serviceName})
	if err != nil {
		t.Fatalf("Error in Service Start request. err=%s", err)
	}

	expectedMessage := fmt.Sprintf("Bot %s started service %s", hostname, serviceName)
	_, ok := sm.Cache.Get(serviceName)

	assert.Equal(t, proto.StatusResponse_SUCCESS, resp.Status)
	assert.Equal(t, expectedMessage, resp.Message)
	assert.False(t, ok)
	assert.Equal(t, 0, sm.Cache.ItemCount())
}

func startServiceFail(sm *ServiceManager, serviceName string, t *testing.T) {
	resp, err := sm.Start(context.TODO(), &proto.ServiceRequest{Name: serviceName})

	expectedMessage := fmt.Sprintf("Could not start service %s", serviceName)
	_, ok := sm.Cache.Get(serviceName)

	assert.Error(t, err)
	assert.Equal(t, proto.StatusResponse_FAIL, resp.Status)
	assert.Equal(t, expectedMessage, resp.Message)
	assert.False(t, ok)
	assert.Equal(t, 0, sm.Cache.ItemCount())
}

func stopService(sm *ServiceManager, serviceName string, t *testing.T, hostname string) {
	resp, err := sm.Stop(context.TODO(), &proto.ServiceRequest{JobName: serviceName, Name: serviceName})
	if err != nil {
		t.Fatalf("Error in Service Stop request. err=%s", err)
	}

	expectedMessage := fmt.Sprintf("Bot %s stopped service %s", hostname, serviceName)
	serviceObj, ok := sm.Cache.Get(serviceName)

	if !ok {
		t.Fatalf(fmt.Sprintf("Could not retrieve item %s from cache", serviceName))
	}

	assert.Equal(t, proto.StatusResponse_SUCCESS, resp.Status)
	assert.Equal(t, expectedMessage, resp.Message)
	assert.Equal(t, serviceName, serviceObj.(*service.Service).Name)
	assert.Equal(t, 1, sm.Cache.ItemCount())
}

func recoverService(sm *StrategyManager, serviceName string, t *testing.T, hostname string) {
	resp, err := sm.Recover(context.TODO(), &proto.RecoverRequest{})
	if err != nil {
		t.Fatalf("Error in Service Recover request. err=%s", err)
	}

	expectedMessage := fmt.Sprintf("Bot %s started service %s", hostname, serviceName)
	_, ok := sm.Cache.Get(serviceName)

	assert.Equal(t, proto.StatusResponse_SUCCESS, resp.Response[0].Status)
	assert.Equal(t, expectedMessage, resp.Response[0].Message)
	assert.False(t, ok)
	assert.Equal(t, 0, sm.Cache.ItemCount())
}

func recoverServiceEmpty(sm *StrategyManager, serviceName string, t *testing.T) {
	resp, err := sm.Recover(context.TODO(), &proto.RecoverRequest{})
	if err != nil {
		t.Fatalf("Error in Service Recover request. err=%s", err)
	}

	_, ok := sm.Cache.Get(serviceName)

	assert.Equal(t, 0, len(resp.Response))
	assert.False(t, ok)
	assert.Equal(t, 0, sm.Cache.ItemCount())
}

// === End to end testing ===
func TestDockerManager_e2e(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}

	logger := getLogger()
	myCache := cache.New(0, 0)
	dockerName := "zookeeper"
	hostname, _ := os.Hostname()

	dm := &DockerManager{Cache: myCache, Logger: logger}
	stratM := &StrategyManager{myCache, logger}

	startDocker(dm, dockerName, t, hostname)
	recoverDockerEmpty(stratM, dockerName, t)
	stopDocker(dm, dockerName, t, hostname)
	recoverDocker(stratM, dockerName, t, hostname)
	stopDocker(dm, dockerName, t, hostname)
}

func startDocker(dm *DockerManager, dockerName string, t *testing.T, hostname string) {
	resp, err := dm.Start(context.TODO(), &proto.DockerRequest{Name: dockerName})
	if err != nil {
		t.Fatalf("Error in Docker Start request. err=%s", err)
	}

	expectedMessage := fmt.Sprintf("Bot %s started docker container %s", hostname, dockerName)

	assert.Equal(t, proto.StatusResponse_SUCCESS, resp.Status)
	assert.Equal(t, expectedMessage, resp.Message)
}

func recoverDockerEmpty(sm *StrategyManager, dockerName string, t *testing.T) {
	resp, err := sm.Recover(context.TODO(), &proto.RecoverRequest{})
	if err != nil {
		t.Fatalf("Error in Docker recover Start request. err=%s", err)
	}

	_, ok := sm.Cache.Get(dockerName)

	assert.Equal(t, 0, len(resp.Response))
	assert.False(t, ok)
	assert.Equal(t, 0, sm.Cache.ItemCount())
}

func stopDocker(dm *DockerManager, dockerName string, t *testing.T, hostname string) {
	resp, err := dm.Stop(context.TODO(), &proto.DockerRequest{Name: dockerName})
	if err != nil {
		t.Fatalf("Error in Docker Stop request. err=%s", err)
	}

	expectedMessage := fmt.Sprintf("Bot %s stopped docker container %s", hostname, dockerName)

	assert.Equal(t, proto.StatusResponse_SUCCESS, resp.Status)
	assert.Equal(t, expectedMessage, resp.Message)
}

func recoverDocker(sm *StrategyManager, dockerName string, t *testing.T, hostname string) {
	resp, err := sm.Recover(context.TODO(), &proto.RecoverRequest{})
	if err != nil {
		t.Fatalf("Error in Docker recover Start request. err=%s", err)
	}

	expectedMessage := fmt.Sprintf("Bot %s started docker container %s", hostname, dockerName)
	_, ok := sm.Cache.Get(dockerName)

	assert.Equal(t, proto.StatusResponse_SUCCESS, resp.Response[0].Status)
	assert.Equal(t, expectedMessage, resp.Response[0].Message)
	assert.False(t, ok)
	assert.Equal(t, 0, sm.Cache.ItemCount())
}

func getLogger() log.Logger {
	allowLevel := &chaoslogger.AllowedLevel{}
	if err := allowLevel.Set("debug"); err != nil {
		fmt.Printf("%v", err)
	}

	return chaoslogger.New(allowLevel)
}
