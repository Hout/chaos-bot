package v1

import (
	"context"
	"fmt"

	"github.com/SotirisAlfonsos/chaos-bot/common"
	"github.com/SotirisAlfonsos/chaos-bot/common/docker"
	"github.com/SotirisAlfonsos/chaos-bot/common/service"
	"github.com/SotirisAlfonsos/chaos-bot/proto"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HealthCheckService is the rpc
type HealthCheckService struct {
}

// Check the health of the chaos bot
func (hcs *HealthCheckService) Check(ctx context.Context,
	req *proto.HealthCheckRequest) (*proto.HealthCheckResponse, error) {
	return &proto.HealthCheckResponse{Status: proto.HealthCheckResponse_SERVING}, nil
}

// Watch is not used at the moment
func (hcs *HealthCheckService) Watch(req *proto.HealthCheckRequest, srv proto.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "method Watch not implemented")
}

// ServiceManager is the rpc for services management
type ServiceManager struct {
	Cache  *cache.Cache
	Logger log.Logger
}

// Start a service based on the name. Delete the item from the cache if it had been cached previously
func (sm *ServiceManager) Start(ctx context.Context, req *proto.ServiceRequest) (*proto.StatusResponse, error) {
	serviceManage := &service.Service{JobName: req.JobName, Name: req.Name, Logger: sm.Logger}

	message, err := startTarget(serviceManage, sm.Cache, req.Name)

	return prepareResponse(message, err)
}

// Stop a service based on the name. Cache it if the service is stopped successfully
func (sm *ServiceManager) Stop(ctx context.Context, req *proto.ServiceRequest) (*proto.StatusResponse, error) {
	serviceManage := &service.Service{JobName: req.JobName, Name: req.Name, Logger: sm.Logger}

	message, err := stopTarget(serviceManage, sm.Cache, req.Name, sm.Logger)

	return prepareResponse(message, err)
}

// DockerManager is the rpc for docker management
type DockerManager struct {
	Cache  *cache.Cache
	Logger log.Logger
}

// Start a docker container based on the name. Delete the item from the cache if it had been cached previously
func (dm *DockerManager) Start(ctx context.Context, req *proto.DockerRequest) (*proto.StatusResponse, error) {
	dockerManage := &docker.Docker{JobName: req.JobName, Name: req.Name, Logger: dm.Logger}

	message, err := startTarget(dockerManage, dm.Cache, req.Name)

	return prepareResponse(message, err)
}

// Stop a docker container based on the name. Cache it if the docker container is stopped successfully
func (dm *DockerManager) Stop(ctx context.Context, req *proto.DockerRequest) (*proto.StatusResponse, error) {
	dockerManage := &docker.Docker{JobName: req.JobName, Name: req.Name, Logger: dm.Logger}

	message, err := stopTarget(dockerManage, dm.Cache, req.Name, dm.Logger)

	return prepareResponse(message, err)
}

func startTarget(target common.Target, cache *cache.Cache, name string) (string, error) {
	message, err := target.Start()
	if err == nil {
		cache.Delete(name)
	}

	return message, err
}

func stopTarget(target common.Target, cache *cache.Cache, name string, logger log.Logger) (string, error) {
	message, err := target.Stop()
	if err == nil {
		if cacheErr := cache.Add(name, target, 0); cacheErr != nil {
			_ = level.Error(logger).Log("msg",
				fmt.Sprintf("Could not update cache after stopping target %s", name), "err", cacheErr)
		}
	}

	return message, err
}

// StrategyManager handles recovery of services
type StrategyManager struct {
	Cache  *cache.Cache
	Logger log.Logger
}

// Recover all services that are in the cache (have been stopped). Clean cache for every successful recovery
func (sm *StrategyManager) Recover(ctx context.Context, req *proto.RecoverRequest) (*proto.ResolveResponse, error) {
	responses := make([]*proto.StatusResponse, 0)

	var err error

	for item := range sm.Cache.Items() {
		target, ok := sm.Cache.Get(item)
		if !ok {
			_ = level.Error(sm.Logger).Log("err", fmt.Sprintf("Could not find item %s in cache", item))
		}

		message, startErr := target.(common.Target).Start()
		if startErr == nil {
			sm.Cache.Delete(item)
			_ = level.Info(sm.Logger).Log("err", fmt.Sprintf("Started and removed item %s from cache", item))
		} else {
			err = errors.Wrap(err, startErr.Error())
		}

		resp, respErr := prepareResponse(message, err)
		if respErr != nil {
			err = errors.Wrap(err, respErr.Error())
		}

		responses = append(responses, resp)
	}

	return &proto.ResolveResponse{Response: responses}, err
}

func prepareResponse(message string, err error) (*proto.StatusResponse, error) {
	if err != nil {
		return respFail(message, err)
	}

	return respSuccess(message)
}

func respSuccess(message string) (*proto.StatusResponse, error) {
	return &proto.StatusResponse{
		Status:  proto.StatusResponse_SUCCESS,
		Message: message,
	}, nil
}

func respFail(message string, err error) (*proto.StatusResponse, error) {
	return &proto.StatusResponse{
		Status:  proto.StatusResponse_FAIL,
		Message: message,
	}, err
}
