package main

import (
	"chaos-slave/chaoslogger"
	"chaos-slave/web"
	"flag"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/patrickmn/go-cache"
)

func main() {
	debugLevel := flag.String("debug.level", "info", "the debug level for the chaos slave. Can be one of debug, info, warn, error.")
	port := flag.String("port", "8080", "the port used by the grpc server.")
	flag.Parse()

	logger := createLogger(*debugLevel)
	myCache := cache.New(0, 0)

	grpcHandler := web.NewGrpcHandler(*port, logger, myCache)
	if err := grpcHandler.Run(); err != nil {
		_ = level.Error(logger).Log("msg", "Failed to start Grpc server on port "+*port, "err", err)
	}
}

func createLogger(debugLevel string) log.Logger {
	allowLevel := &chaoslogger.AllowedLevel{}
	if err := allowLevel.Set(debugLevel); err != nil {
		fmt.Printf("%v", err)
	}
	
	return chaoslogger.New(allowLevel)
}
