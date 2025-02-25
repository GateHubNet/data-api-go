package gapi

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/scylladb/gocqlx/v2/qb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/GateHubNet/data-api-go/internal/models"
	"github.com/GateHubNet/data-api-go/internal/pb"
	"github.com/GateHubNet/data-api-go/internal/util"
)

func (server *Server) accountBalanceChangesResponseHelper(ctx context.Context, balanceChanges []*pb.BalanceChanges) *pb.GetAccountBalanceChangesResponse {
	_ = grpc.SetHeader(ctx, metadata.Pairs(util.HttpCode, "200"))

	return &pb.GetAccountBalanceChangesResponse{
		Result:         "success",
		BalanceChanges: balanceChanges,
		Counter:        uint64(len(balanceChanges)),
	}
}

func (server *Server) GetAccountBalanceChanges(ctx context.Context, req *pb.GetAccountBalanceChangesRequest) (*pb.GetAccountBalanceChangesResponse, error) {
	address := req.GetAddress()

	log.Info().Str("account", address).Msg("Get account balance changes")

	// Initialize a slice with an estimated capacity if known, or leave it as zero.
	var balanceChanges []pb.BalanceChanges

	// Generate the CQL statement and names.
	queryStatement := qb.Select(models.BalanceChanges.Name()).Where(qb.Eq("account")).Limit(200)
	//TODO: check other fields
	stmt, names := queryStatement.ToCql()
	log.Debug().Str("query", stmt).Msg("Get balance changes for address from DB")

	// Execute the query and populate balanceChanges.
	if err := server.Store.Query(stmt, names).Bind(address).SelectRelease(&balanceChanges); err != nil {
		log.Error().Err(err).Msg("Failed to get account balance changes from the database")
		return nil, err
	}

	// Convert to a slice of pointers in a single pass
	balanceChangePointers := make([]*pb.BalanceChanges, len(balanceChanges))
	for i := range balanceChanges {
		balanceChangePointers[i] = &balanceChanges[i]
	}

	// Log the first few balance changes
	log.Debug().Interface("balanceChanges", balanceChangePointers[:min(3, len(balanceChangePointers))]).Msg("Got balance changes")

	// Return the response
	return server.accountBalanceChangesResponseHelper(ctx, balanceChangePointers), nil
}

// Helper function to calculate the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
