package gapi

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/scylladb/gocqlx/v2/qb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GateHubNet/data-api-go/internal/models"
	"github.com/GateHubNet/data-api-go/internal/pb"
)

func (server *Server) GetTransactionByHash(transactionHash string) (*pb.Transaction, error) {
	log.Info().Str("hash", transactionHash).Msg("Get transaction by hash")

	transaction := pb.Transaction{}

	stmt, names := qb.Select(models.Transactions.Name()).Where(qb.Eq("hash")).ToCql()
	log.Debug().Interface("query", stmt).Msg("Get transaction by hash from DB")

	q := server.Store.Query(stmt, names).Bind(transactionHash)
	if err := q.GetRelease(&transaction); err != nil {
		log.Error().Err(err).Msg("Failed to get transaction from the database")
		return nil, err
	}

	log.Debug().Interface("transaction", &transaction).Msg("Got transaction")

	return &transaction, nil
}

func (server *Server) GetTransactionsByLedgerIndex(ledgerIndex int64) ([]pb.Transaction, error) {
	log.Info().Int64("ledger_index", ledgerIndex).Msg("Get transaction by ledger index")

	var transactions []pb.Transaction

	stmt, names := qb.Select(models.Transactions.Name()).Where(qb.Eq("ledger_index")).ToCql()
	log.Debug().Interface("query", stmt).Msg("Get transaction by ledger_index from DB")

	q := server.Store.Query(stmt, names).Bind(ledgerIndex)
	if err := q.SelectRelease(&transactions); err != nil {
		log.Error().Err(err).Msg("Failed to get transaction by ledger_index from the database")
		return nil, err
	}

	showTransactions := transactions
	if len(showTransactions) > 3 {
		showTransactions = showTransactions[:3]
	}
	log.Debug().Interface("transaction", &showTransactions).Msg("Got transaction")

	return transactions, nil
}

func (server *Server) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.GetTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetTransaction not implemented")
}
