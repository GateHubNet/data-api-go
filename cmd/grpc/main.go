package main

import (
	"net"
	"os"

	"github.com/go-redis/redis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/GateHubNet/data-api-go/internal/gapi"
	"github.com/GateHubNet/data-api-go/internal/pb"
	"github.com/GateHubNet/data-api-go/internal/util"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	server, err := gapi.NewServer(config)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	defer server.Store.Close()
	defer func(Cache *redis.ClusterClient) {
		if err := Cache.Close(); err != nil {
			log.Error().Err(err).Msg("cannot close cache")
		}
	}(server.Cache)

	grpcLogger := grpc.UnaryInterceptor(gapi.GrpcLogger)
	grpcServer := grpc.NewServer(grpcLogger)
	pb.RegisterGateHubDataAPIServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create listener")
	}

	log.Info().Msgf("gRPC server listening on %v", listener.Addr())

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start gRPC server")
	}
}
