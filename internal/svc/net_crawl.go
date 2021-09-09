// Package svc implements blockchain data processing services.
package svc

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"net"
	"strings"
)

// networkCrawler implements Fantom node network crawler collecting information about
// the network nodes.
type networkCrawler struct {
	cfg     *discover.Config
	db      *enode.DB
	node    *enode.LocalNode
	disc    *discover.UDPv5
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
	nec.isReady = true
}

// run starts the network crawl collecting Fantom nodes.
func (nec *networkCrawler) run() {
	// quit the service, if not ready
	if !nec.isReady {
		log.Criticalf("network crawler not ready and will not run")
		return
	}

	// make local node
	nec.node = enode.NewLocalNode(nec.db, nec.cfg.PrivateKey)
	err := nec.listen()
	if err != nil {
		log.Criticalf("network crawler not ready and will not run")
		return
	}

	// signal orchestrator we started and go
	nec.mgr.started(nec)
	go nec.crawl()
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

	// open the database
	return nec.openNodesDatabase()
}

// setPrivateKey loads and/or generate private key for local discovery node.
func (nec *networkCrawler) setPrivateKey() (err error) {
	// if we don't have the key in config, just generate a new one
	if cfg.Sig.PrivateKey == nil {
		nec.cfg.PrivateKey, err = crypto.GenerateKey()
		return
	}

	// just assign the ECDSA key given in config
	nec.cfg.PrivateKey = cfg.Sig.PrivateKey
	return nil
}

// setBootstrap prepares a list of bootstrap nodes
// used to start the scan, if we don't know any nodes yet.
func (nec *networkCrawler) setBootstrap() (err error) {
	nec.cfg.Bootnodes = make([]*enode.Node, len(cfg.LocalNode.V5Bootstrap))
	for i, url := range cfg.LocalNode.V5Bootstrap {
		nec.cfg.Bootnodes[i], err = url2node(url)
		if err != nil {
			return
		}
	}
	return nil
}

// openNodesDatabase opens the database used by the server
// to keep track of the found nodes.
func (nec *networkCrawler) openNodesDatabase() error {
	// try to open the db; if path is not provided, we use in-memory database
	db, err := enode.OpenDB(cfg.LocalNode.DbPath)
	if err != nil {
		log.Criticalf("can not open node database; %s", err.Error())
		return err
	}

	nec.db = db
	return nil
}

// listen starts UDP listener for the local node
func (nec *networkCrawler) listen() error {
	// open socket
	socket, err := net.ListenPacket("udp4", cfg.LocalNode.BindAddress)
	if err != nil {
		log.Criticalf("can not open p2p node socket at [%s]; %s", cfg.LocalNode.BindAddress, err.Error())
		return err
	}

	// check the IP address and set local node fallback
	addr := socket.LocalAddr().(*net.UDPAddr)
	if addr.IP.IsUnspecified() {
		nec.node.SetFallbackIP(net.IP{127, 0, 0, 1})
	} else {
		nec.node.SetFallbackIP(addr.IP)
	}

	// start discovery protocol
	nec.node.SetFallbackUDP(addr.Port)
	nec.disc, err = discover.ListenV5(socket.(*net.UDPConn), nec.node, *nec.cfg)
	if err != nil {
		log.Criticalf("can not start discovery protocol; %s", err.Error())
		return err
	}
	return nil
}

// crawl traverses the node network validating known nodes and registering
// newly found members; it updates repository to keep track of the network topology
func (nec *networkCrawler) crawl() {
	// make sure to close everything
	defer func() {
		// close the discovery (this also closes connection)
		nec.disc.Close()

		// signal we are done
		nec.mgr.finished(nec)
	}()
}

// url2node parses V4 enode address string into Node structure.
func url2node(url string) (*enode.Node, error) {
	if !strings.HasPrefix(url, "enode://") {
		return nil, fmt.Errorf("invalid enode url [%s]", url)
	}
	return enode.ParseV4(url)
}
