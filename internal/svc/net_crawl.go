// Package svc implements blockchain data processing services.
package svc

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"strings"
)

// networkCrawler implements Fantom node network crawler collecting information about
// the network nodes.
type networkCrawler struct {
	cfg     *discover.Config
	isReady bool
	service
}

// name returns the name of the service used by orchestrator.
func (nec *networkCrawler) name() string {
	return "network crawler"
}

// init prepares the network crawler for collecting node data.
func (nec *networkCrawler) init() {
	nec.sigStop = make(chan bool, 1)

	// make the config
	if err := nec.configure(); err != nil {
		log.Criticalf("can not run local discovery; %s", err.Error())
		return
	}

	// all ready
	nec.isReady = true
}

// run starts the network crawl collecting Fantom nodes.
func (nec *networkCrawler) run() {
	// quit the service, if not ready
	if !nec.isReady {
		log.Criticalf("network crawler not ready and will not run")
		return
	}
}

// configure prepares the local node to operate.
func (nec *networkCrawler) configure() error {
	nec.cfg = new(discover.Config)

	// prep private key for the local node
	if err := nec.setPrivateKey(); err != nil {
		return err
	}

	// setup bootstrap nodes to give server something to start from
	if err := nec.setBootstrap(); err != nil {
		return err
	}

	// nec.openNodesDatabase()
	return nil
}

// setPrivateKey loads and/or generate private key for local discovery node.
func (nec *networkCrawler) setPrivateKey() (err error) {
	// if we don't have the key in config, just generate a new one
	if cfg.Lachesis.NodeKey == "" {
		nec.cfg.PrivateKey, err = crypto.GenerateKey()
		return
	}

	// decode ECDSA key given
	nec.cfg.PrivateKey, err = crypto.HexToECDSA(cfg.Lachesis.NodeKey)
	return err
}

// setBootstrap prepares a list of bootstrap nodes
// used to start the scan, if we don't know any nodes yet.
func (nec *networkCrawler) setBootstrap() (err error) {
	nec.cfg.Bootnodes = make([]*enode.Node, len(cfg.Lachesis.V5Bootstrap))
	for i, url := range cfg.Lachesis.V5Bootstrap {
		nec.cfg.Bootnodes[i], err = url2node(url)
		if err != nil {
			return
		}
	}
	return nil
}

// url2node parses V4 enode address string into Node structure.
func url2node(url string) (*enode.Node, error) {
	if !strings.HasPrefix(url, "enode://") {
		return nil, fmt.Errorf("invalid enode url [%s]", url)
	}
	return enode.ParseV4(url)
}
