package net

import (
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/spacemeshos/go-spacemesh/crypto"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/p2p/config"
	"github.com/spacemeshos/go-spacemesh/p2p/delimited"
	"github.com/spacemeshos/go-spacemesh/p2p/node"
	"github.com/spacemeshos/go-spacemesh/p2p/pb"
	"gopkg.in/op/go-logging.v1"
	"net"
	"sync"
	"time"
)

//type Net interface {
//	Dial(address string, remotePublicKey crypto.PublicKey, networkId int8) (Connection, error) // Connect to a remote node. Can send when no error.
//	SubscribeOnNewRemoteConnections() chan Connection
//	Logger() *logging.Logger
//	NetworkID() int8
//	HandlePreSessionIncomingMessage(c Connection, msg []byte) error
//	LocalNode() *node.LocalNode
//	IncomingMessages() chan IncomingMessageEvent
//	ClosingConnections() chan Connection
//	Shutdown()
//}

type IncomingMessageEvent struct {
	Conn    Connection
	Message []byte
}

type ManagedConnection interface {
	Connection
	beginEventProcessing()
}

// Net is a connection factory able to dial remote endpoints
// Net clients should register all callbacks
// Connections may be initiated by Dial() or by remote clients connecting to the listen address
// ConnManager includes a TCP server, and a TCP client
// It provides full duplex messaging functionality over the same tcp/ip connection
// Network should not know about higher-level networking types such as remoteNode, swarm and networkSession
// Network main client is the swarm
// Net has no channel events processing loops - clients are responsible for polling these channels and popping events from them
type Net struct {
	networkId int8
	localNode *node.LocalNode
	logger    *logging.Logger

	tcpListener      net.Listener
	tcpListenAddress string // Address to open connection: localhost:9999\

	isShuttingDown bool

	regNewRemoteConn []chan Connection
	regMutex         sync.RWMutex

	incomingMessages   chan IncomingMessageEvent
	closingConnections chan Connection

	config config.Config
}

// NewNet creates a new network.
// It attempts to tcp listen on address. e.g. localhost:1234 .
func NewNet(conf config.Config, localEntity *node.LocalNode) (*Net, error) {

	n := &Net{
		networkId:          conf.NetworkID,
		localNode:          localEntity,
		logger:             localEntity.Logger,
		tcpListenAddress:   localEntity.Address(),
		regNewRemoteConn:   make([]chan Connection, 0),
		incomingMessages:   make(chan IncomingMessageEvent),
		closingConnections: make(chan Connection, 20),
		config:             conf,
	}

	err := n.listen()

	if err != nil {
		return nil, err
	}

	n.logger.Debug("created network with tcp address: %s", n.tcpListenAddress)

	return n, nil
}

func (n *Net) Logger() *logging.Logger {
	return n.logger
}

func (n *Net) NetworkID() int8 {
	return n.networkId
}

func (n *Net) LocalNode() *node.LocalNode {
	return n.localNode
}

func (n *Net) IncomingMessages() chan IncomingMessageEvent {
	return n.incomingMessages
}

func (n *Net) ClosingConnections() chan Connection {
	return n.closingConnections
}

func (n *Net) createConnection(address string, remotePub crypto.PublicKey, timeOut time.Duration, keepAlive time.Duration) (ManagedConnection, error) {
	if n.isShuttingDown {
		return nil, fmt.Errorf("can't dial because the connection is shutting down")
	}
	// connect via dialer so we can set tcp network params
	dialer := &net.Dialer{}
	dialer.KeepAlive = keepAlive // drop connections after a period of inactivity
	dialer.Timeout = timeOut     // max time bef
	n.logger.Debug("TCP dialing %s ...", address)

	netConn, err := dialer.Dial("tcp", address)

	if err != nil {
		return nil, err
	}

	n.logger.Debug("Connected to %s...", address)
	formatter := delimited.NewChan(10)
	c := newConnection(netConn, n, formatter, remotePub, n.logger)

	return c, nil
}

func (n *Net) createSecuredConnection(address string, remotePublicKey crypto.PublicKey, networkId int8, timeOut time.Duration, keepAlive time.Duration) (ManagedConnection, error) {
	errMsg := "failed to establish secured connection."
	conn, err := n.createConnection(address, remotePublicKey, timeOut, keepAlive)
	if err != nil {
		return nil, err
	}
	data, session, err := GenerateHandshakeRequestData(n.localNode.PublicKey(), n.localNode.PrivateKey(), remotePublicKey, networkId)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("%s err: %v", errMsg, err)
	}
	n.logger.Debug("Creating session handshake request session id: %s", session)
	payload, err := proto.Marshal(data)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("%s err: %v", errMsg, err)
	}

	err = conn.Send(payload)
	if err != nil {
		conn.Close()
		return nil, err
	}

	var msg []byte
	var ok bool
	timer := time.NewTimer(n.config.ResponseTimeout)
	select {
	case msg, ok = <-conn.IncomingChannel():
		if !ok {
			conn.Close()
			return nil, fmt.Errorf("%s err: incoming channel got closed", errMsg)
		}
	case <-timer.C:
		n.logger.Info("waiting for HS response timed-out. remoteKey=%v", remotePublicKey)
		conn.Close()
		return nil, fmt.Errorf("%s err: HS response timed-out", errMsg)
	}

	respData := &pb.HandshakeData{}
	err = proto.Unmarshal(msg, respData)
	if err != nil {
		//n.logger.Warning("invalid incoming handshake resp bin data", err)
		conn.Close()
		return nil, fmt.Errorf("%s err: %v", errMsg, err)
	}

	err = ProcessHandshakeResponse(remotePublicKey, session, respData)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("%s err: %v", errMsg, err)
	}

	conn.SetSession(session)
	return conn, nil
}

// Dial a remote server with provided time out
// address:: ip:port
// Returns established connection that local clients can send messages to or error if failed
// to establish a connection, currently only secured connections are supported
func (n *Net) Dial(address string, remotePublicKey crypto.PublicKey, networkId int8) (Connection, error) {
	conn, err := n.createSecuredConnection(address, remotePublicKey, networkId, n.config.DialTimeout, n.config.ConnKeepAlive)
	if err != nil {
		return nil, fmt.Errorf("failed to Dail. err: %v", err)
	}
	go conn.beginEventProcessing()
	return conn, nil
}

func (n *Net) Shutdown() {
	n.isShuttingDown = true
	n.tcpListener.Close()
}

// Start network server
func (n *Net) listen() error {
	n.logger.Info("Starting to listen...")
	tcpListener, err := net.Listen("tcp", n.tcpListenAddress)
	if err != nil {
		return err
	}
	n.tcpListener = tcpListener
	go n.acceptTCP()
	return nil
}

func (n *Net) acceptTCP() {
	for {
		n.logger.Debug("Waiting for incoming connections...")
		netConn, err := n.tcpListener.Accept()
		if err != nil {

			if !n.isShuttingDown {
				log.Error("Failed to accept connection request", err)
				//TODO only print to log and return? The node will continue running without the listener, doesn't sound healthy
			}
			return
		}

		n.logger.Debug("Got new connection... Remote Address: %s", netConn.RemoteAddr())
		formatter := delimited.NewChan(10)
		c := newConnection(netConn, n, formatter, nil, n.logger)

		go c.beginEventProcessing()
		// network won't publish the connection before it the remote node had established a session
	}
}

func (n *Net) SubscribeOnNewRemoteConnections() chan Connection {
	n.regMutex.Lock()
	ch := make(chan Connection, 20)
	n.regNewRemoteConn = append(n.regNewRemoteConn, ch)
	n.regMutex.Unlock()
	return ch
}

func (n *Net) publishNewRemoteConnection(conn Connection) {
	n.regMutex.RLock()
	for _, c := range n.regNewRemoteConn {
		c <- conn
	}
	n.regMutex.RUnlock()
}

func (n *Net) HandlePreSessionIncomingMessage(c Connection, message []byte) error {
	//TODO replace the next few lines with a way to validate that the message is a handshake request based on the message metadata
	errMsg := "failed to handle handshake request"
	data := &pb.HandshakeData{}

	err := proto.Unmarshal(message, data)
	if err != nil {
		return fmt.Errorf("%s. err: %v", errMsg, err)
	}

	// new remote connection doesn't hold the remote public key until it gets the handshake request
	if c.RemotePublicKey() == nil {
		rPub, err := crypto.NewPublicKey(data.GetNodePubKey())
		n.Logger().Info("DEBUG: handling HS req from %v", rPub)
		if err != nil {
			return fmt.Errorf("%s. err: %v", errMsg, err)
		}
		c.SetRemotePublicKey(rPub)

	}
	respData, session, err := ProcessHandshakeRequest(n.NetworkID(), n.localNode.PublicKey(), n.localNode.PrivateKey(), c.RemotePublicKey(), data)
	payload, err := proto.Marshal(respData)
	if err != nil {
		return fmt.Errorf("%s. err: %v", errMsg, err)
	}

	err = c.Send(payload)
	if err != nil {
		return err
	}

	c.SetSession(session)
	// update on new connection
	n.publishNewRemoteConnection(c)
	return nil
}
