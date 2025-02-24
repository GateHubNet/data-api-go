package gapi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/scylladb/gocqlx/v2/qb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	"github.com/GateHubNet/data-api-go/internal/models"
	"github.com/GateHubNet/data-api-go/internal/pb"
	"github.com/GateHubNet/data-api-go/internal/util"
)

func (server *Server) getLedgerByIndex(ledgerIndex int64) (*pb.Ledger, error) {
	l := models.LedgersStruct{
		LedgerIndex: ledgerIndex,
	}

	ledger := pb.Ledger{}

	log.Debug().Interface("LedgersStruct", l).Msg("Get ledger by ledger_index from the DB")

	startQueryTime := time.Now()
	q := server.Store.Query(models.Ledgers.Get()).BindStruct(l)
	if err := q.GetRelease(&ledger); err != nil {
		log.Warn().Err(err).Msg("Failed to get ledger_index from the database")
		return nil, err
	}

	log.Debug().Interface("ledger", &ledger).TimeDiff("fetchTime", time.Now(), startQueryTime).Msg("Got Ledger data")
	return &ledger, nil
}

func (server *Server) getLedgerByHash(ledgerHash string) (*pb.Ledger, error) {
	log.Info().Str("identifier", ledgerHash).Msg("Ledger identifier recognized as ledger hash")

	ledger := pb.Ledger{}

	stmt, names := qb.Select(models.Ledgers.Name()).Where(qb.Eq("ledger_hash")).ToCql()
	log.Debug().Interface("query", stmt).Msg("Get ledger by ledger_hash from the DB")

	q := server.Store.Query(stmt, names).Bind(ledgerHash)
	if err := q.GetRelease(&ledger); err != nil {
		log.Error().Err(err).Msg("Failed to get ledger_hash from the database")
		return nil, err
	}

	log.Debug().Interface("ledger", &ledger).Msg("Got Ledger data")

	return &ledger, nil
}

// FindFirstGreater performs binary search - O(log n)
func (server *Server) findFirstGreater(records []pb.DailyLedger, findTimestamp int64) (*pb.DailyLedger, error) {
	if len(records) <= 0 {
		return nil, errors.New("empty daily ledgers")
	}

	left, right := 0, len(records)-1
	result := -1 // Default value if not found

	for left <= right {
		mid := left + (right-left)/2

		if records[mid].CloseTime > findTimestamp {
			result = mid
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	if result >= 0 {
		foundLedger := &records[result]
		if foundLedger != nil {
			return foundLedger, nil
		}

		lastDailyLedger := &records[len(records)-1]
		return lastDailyLedger, nil
	}

	return &records[0], nil
}

func (server *Server) getLedgerByTimestamp(timestamp time.Time) (*pb.Ledger, error) {
	log.Info().Str("identifier", timestamp.String()).Msg("Ledger identifier recognized as timestamp")

	//ok, so identifier here is timestamp...we need to get day from the string
	ledgerDate := timestamp.Format(time.DateOnly)
	var dailyLedgers []pb.DailyLedger

	log.Debug().Interface("ledgerDate", ledgerDate).Msg("Getting Daily Ledger with following date")
	stmt, names := qb.Select(models.DailyLedgers.Name()).Where(qb.Eq("ledger_close_day")).ToCql()
	log.Debug().Interface("query", stmt).Msg("Get ledger by timestamp from the DB")

	queryStartTime := time.Now()
	q := server.Store.Query(stmt, names).Bind(ledgerDate)
	if err := q.SelectRelease(&dailyLedgers); err != nil {
		log.Error().Err(err).Msg("Failed to get daily ledger by date from the database")
		return nil, err
	}

	showDailyLedgers := dailyLedgers
	if len(showDailyLedgers) > 3 {
		showDailyLedgers = showDailyLedgers[:3]
	}
	log.Debug().Interface("daily_ledgers", showDailyLedgers).TimeDiff("fetchTime", time.Now(), queryStartTime).Msg("Got DailyLedgers")

	findFirstGreaterStartTime := time.Now()
	foundDailyLedger, foundDailyLedgerErr := server.findFirstGreater(dailyLedgers, timestamp.Unix())
	if foundDailyLedgerErr != nil {
		return nil, foundDailyLedgerErr
	}

	log.Debug().Interface("daily_ledger", foundDailyLedger).TimeDiff("findTime", time.Now(), findFirstGreaterStartTime).Msg("Found DailyLedger")

	return server.getLedgerByIndex(foundDailyLedger.LedgerIndex)
}

func (server *Server) findLedger(identifier string) (*pb.Ledger, error) {
	if parsed, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		return server.getLedgerByIndex(parsed)
	} else if parsedTimestamp, err := time.Parse(time.RFC3339, identifier); err == nil {
		return server.getLedgerByTimestamp(parsedTimestamp)
	}
	return server.getLedgerByHash(identifier)
}

func (server *Server) ledgerResponseHelper(ctx context.Context, ledger *pb.Ledger) *pb.GetLedgerResponse {
	_ = grpc.SetHeader(ctx, metadata.Pairs(util.HttpCode, "200"))

	return &pb.GetLedgerResponse{
		Result: "success",
		Ledger: ledger,
	}
}

func (server *Server) ledgerResponseErrorHelper(ctx context.Context, err error) *pb.GetLedgerResponse {
	_ = grpc.SetHeader(ctx, metadata.Pairs(util.HttpCode, "400"))

	log.Warn().Err(err).Msg("Failed to find ledger")

	message := err.Error()
	return &pb.GetLedgerResponse{
		Result:  "error",
		Message: &message,
	}
}

func (server *Server) saveCache(key string, value proto.Message, expiration time.Duration) error {
	// Serialize the Protobuf struct to a byte slice
	dataBytes, err := proto.Marshal(value)
	if err != nil {
		return err
	}

	// Save the byte slice to Redis
	return server.Cache.Set(key, dataBytes, expiration).Err()
}

func (server *Server) getCache(key string, target proto.Message) error {
	// Retrieve the byte slice from Redis
	val, err := server.Cache.Get(key).Bytes()
	if err != nil {
		return err
	}

	// Deserialize the byte slice back into the Protobuf struct
	return proto.Unmarshal(val, target)
}

func (server *Server) GetLastValidatedLedger(ctx context.Context, req *pb.GetLastValidatedLedgerRequest) (*pb.GetLedgerResponse, error) {
	identifier := time.Now()

	ledger, err := server.getLedgerByTimestamp(identifier)

	if err != nil {
		return server.ledgerResponseErrorHelper(ctx, err), nil
	}

	return server.ledgerResponseHelper(ctx, ledger), nil
}

func (server *Server) GetLedger(ctx context.Context, req *pb.GetLedgerRequest) (*pb.GetLedgerResponse, error) {
	ledgerIdentifier := req.GetIdentifier()

	ledger := &pb.Ledger{}

	// Attempt to retrieve the ledger from the cache.
	cacheKey := fmt.Sprintf(server.ledgerCacheKey, ledgerIdentifier)
	if err := server.getCache(cacheKey, ledger); err != nil {
		// If an error occurs (cache miss or cache service issue), log or handle the error accordingly.
		log.Warn().Str("ledgerIdentifier", ledgerIdentifier).Err(err).Msg("Cache miss or error for ledger")

		// Since the cache didn't have the data, query it from the source.
		var findErr error
		ledger, findErr = server.findLedger(ledgerIdentifier)
		if findErr != nil {
			return server.ledgerResponseErrorHelper(ctx, findErr), nil
		}

		// This step is crucial for cache population and to avoid future cache misses for this item.
		if cacheUpdateErr := server.saveCache(cacheKey, ledger, time.Hour*24); cacheUpdateErr != nil {
			// Log or handle the error during cache update.
			// We will just write to log, since this is not really a crucial operation
			log.Warn().Str("ledgerIdentifier", ledgerIdentifier).Err(cacheUpdateErr).Msg("Error updating cache for ledger")
		}
	} else {
		// Cache hit, use the ledger from cache.
		log.Info().Str("ledgerIdentifier", ledgerIdentifier).Msg("Successfully retrieved ledger from cache")
	}

	// check if we need to get transactions also
	if req.GetExpand() {

	} else if req.GetBinary() {
		transactions, err := server.GetTransactionsByLedgerIndex(ledger.LedgerIndex)

		if err != nil {
			log.Warn().Err(err).Msg("Failed to get transactions")
		} else {
			var transactionHashes []string
			for _, tx := range transactions {
				transactionHashes = append(transactionHashes, tx.Tx)
			}
			ledger.Transactions = &pb.Ledger_TransactionHashes{
				TransactionHashes: &pb.HashList{
					Hashes: transactionHashes,
				},
			}
		}
	} else if req.GetTransactions() {
		transactions, err := server.GetTransactionsByLedgerIndex(ledger.LedgerIndex)

		if err != nil {
			log.Warn().Err(err).Msg("Failed to get transactions")
		} else {
			var transactionHashes []string
			for _, tx := range transactions {
				transactionHashes = append(transactionHashes, tx.Hash)
			}
			ledger.Transactions = &pb.Ledger_TransactionHashes{
				TransactionHashes: &pb.HashList{
					Hashes: transactionHashes,
				},
			}
		}
	}

	return server.ledgerResponseHelper(ctx, ledger), nil
}
