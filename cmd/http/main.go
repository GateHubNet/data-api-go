package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

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

	grpcMux := runtime.NewServeMux(runtime.WithForwardResponseOption(func(ctx context.Context, w http.ResponseWriter, m proto.Message) error {
		md, ok := runtime.ServerMetadataFromContext(ctx)
		if !ok {
			return nil
		}

		if vals := md.HeaderMD.Get(util.HttpCode); len(vals) > 0 {
			code, err := strconv.Atoi(vals[0])
			if err != nil {
				return err
			}
			// delete the headers to not expose any grpc-metadata in http response
			delete(md.HeaderMD, util.HttpCode)
			delete(w.Header(), "Grpc-Metadata-X-Http-Code")
			w.WriteHeader(code)
		}

		return nil
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = pb.RegisterGateHubDataAPIHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot register handler server")
	}

	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create HTTP listener")
	}

	log.Info().Msgf("HTTP server listening on %s", listener.Addr().String())

	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start HTTP server")
	}
}
