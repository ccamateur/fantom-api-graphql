/*
Repository package implements repository for handling fast and efficient access to data required
by the resolvers of the API server.

Internally it utilizes RPC to access Opera/Lachesis full node for blockchain interaction. Mongo database
for fast, robust and scalable off-chain data storage, especially for aggregated and pre-calculated data mining
results. BigCache for in-memory object storage to speed up loading of frequently accessed entities.
*/
package repository

import (
	"fantom-api-graphql/internal/config"
	"fantom-api-graphql/internal/logger"
	"fantom-api-graphql/internal/repository/cache"
	"fantom-api-graphql/internal/repository/db"
	"fantom-api-graphql/internal/repository/rpc"
	"fantom-api-graphql/internal/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"sync"
)

// Repository interface defines functions the underlying implementation provides to API resolvers.
type Repository interface {
	// Close and cleanup the repository.
	Close()

	// Account returns account at Opera blockchain for an address, nil if not found.
	Account(*common.Address) (*types.Account, error)

	// AccountBalance returns the current balance of an account at Opera blockchain.
	AccountBalance(*types.Account) (*hexutil.Big, error)

	// AccountNonce returns the current number of sent transactions of an account at Opera blockchain.
	AccountNonce(*types.Account) (*hexutil.Uint64, error)

	// AccountTransactions returns list of transaction hashes for account at Opera blockchain.
	//
	// String cursor represents cursor based on which the list is loaded. If null, it loads either from top,
	// or bottom of the list, based on the value of the integer count. The integer represents
	// the number of transaction loaded at most.
	//
	// For positive number, the list starts right after the cursor (or on top without one) and loads at most
	// defined number of transactions older than that.
	//
	// For negative number, the list starts right before the cursor (or at the bottom without one) and loads at most
	// defined number of transactions newer than that.
	//
	// Transaction are always sorted from newer to older.
	AccountTransactions(*types.Account, *string, int32) (*types.TransactionHashList, error)

	// Block returns a block at Opera blockchain represented by a number. Top block is returned if the number
	// is not provided.
	// If the block is not found, ErrBlockNotFound error is returned.
	BlockByNumber(*hexutil.Uint64) (*types.Block, error)

	// BlockHeight returns the current height of the Opera blockchain in blocks.
	BlockHeight() (*hexutil.Big, error)

	// Block returns a block at Opera blockchain represented by a hash. Top block is returned if the hash
	// is not provided.
	// If the block is not found, ErrBlockNotFound error is returned.
	BlockByHash(*types.Hash) (*types.Block, error)

	// Transaction returns a transaction at Opera blockchain by a hash, nil if not found.
	Transaction(*types.Hash) (*types.Transaction, error)

	// Transactions returns list of transaction hashes at Opera blockchain.
	Transactions(*string, int32) (*types.TransactionHashList, error)

	// Collection pulls list of blocks starting on the specified block number and going up, or down based on count number.
	Blocks(*uint64, int32) (*types.BlockList, error)
}

// Proxy represents Repository interface implementation and controls access to data
// trough several low level bridges.
type proxy struct {
	cache *cache.MemBridge
	db    *db.MongoDbBridge
	rpc   *rpc.OperaBridge
	log   logger.Logger

	// wait group allows synced wait for go routines to terminate
	waitGroup sync.WaitGroup

	// sigScannerStop is channel for signaling interrupt to blockchain scanner
	sigScannerStop chan bool
}

// New creates new instance of Repository implementation, namely proxy structure.
func New(cfg *config.Config, log logger.Logger) (Repository, error) {
	// create new in-memory cache bridge
	caBridge, err := cache.New(cfg, log)
	if err != nil {
		log.Criticalf("can not create in-memory cache bridge, %s", err.Error())
		return nil, err
	}

	// create new database connection bridge
	dbBridge, err := db.New(cfg, log)
	if err != nil {
		log.Criticalf("can not connect backend persistent storage, %s", err.Error())
		return nil, err
	}

	// create new Lachesis RPC bridge
	rpcBridge, err := rpc.New(cfg, log)
	if err != nil {
		log.Criticalf("can not connect Lachesis RPC interface, %s", err.Error())
		return nil, err
	}

	// construct the proxy instance
	p := proxy{
		cache: caBridge,
		db:    dbBridge,
		rpc:   rpcBridge,
		log:   log,
	}

	// propagate callbacks
	dbBridge.SetBalance(p.AccountBalance)

	// start blockchain scanner
	p.sigScannerStop = p.ScanBlockChain(&p.waitGroup)

	// return the proxy
	return &p, nil
}

// Close with close all connections and clean up the pending work for graceful termination.
func (p *proxy) Close() {
	// signal routines to terminate
	p.log.Debugf("sending terminate signal to the scanner")
	if p.sigScannerStop != nil {
		p.sigScannerStop <- true
	}

	// wait scanners to terminate
	p.log.Debugf("waiting for repository to finish background jobs")
	p.waitGroup.Wait()

	// close connections
	p.db.Close()
	p.rpc.Close()
}
