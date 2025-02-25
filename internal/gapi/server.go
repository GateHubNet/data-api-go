package gapi

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/gocql/gocql"
	"github.com/rs/zerolog/log"
	"github.com/scylladb/gocqlx/v2"

	"github.com/GateHubNet/data-api-go/internal/pb"
	"github.com/GateHubNet/data-api-go/internal/util"
)

// Server serves gRPC requests for our GateHub Data Api service.
type Server struct {
	pb.UnimplementedGateHubDataAPIServer
	config util.Config
	Store  gocqlx.Session
	Cache  *redis.ClusterClient

	ledgerCacheKey string
}

// NewServer creates a new gRPC server
func NewServer(config util.Config) (*Server, error) {
	parsedScyllaPort, err := strconv.Atoi(config.ScyllaPort)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot convert scylla port to number")
	}

	scyllaHosts := strings.Split(config.ScyllaHosts, ",")

	cluster := gocql.NewCluster(scyllaHosts...)
	cluster.Keyspace = config.ScyllaKeyspace
	cluster.Port = parsedScyllaPort
	cluster.Timeout = 10 * time.Second
	session, err := gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create ScyllaDB session")
	}

	cache := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:          []string{config.RedisAddress},
		Password:       config.RedisPassword,
		RouteByLatency: true,
	})

	server := &Server{
		config: config,
		Store:  session,
		Cache:  cache,
	}

	server.ledgerCacheKey = "ledgers:%s"

	return server, nil
}
