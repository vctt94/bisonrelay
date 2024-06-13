package client

import (
	"context"
	"fmt"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/slog"
	"github.com/vctt94/pong-bisonrelay/pongrpc/grpc/pong"

	"google.golang.org/grpc"
)

type PongClientCfg struct {
	TLSCertPath string
	Address     string
	Log         slog.Logger
}

// DcrlnPaymentClient implements the PaymentClient interface for servers that
// offer the "dcrln" payment scheme.
type PongClient struct {
	ID           string
	playerNumber int32
	pongClient   pong.PongGameClient
	updatesCh    chan interface{}
	log          slog.Logger
	chainParams  *chaincfg.Params
	notifier     pong.PongGame_InitClient
	stream       pong.PongGame_SignalReadyClient
}

// NewDcrlndPaymentClient creates a new payment client that can send payments
// through dcrlnd.
func NewPongClient(ctx context.Context, cfg *PongClientCfg) (*PongClient, error) {
	// First attempt to establish a connection to lnd's RPC sever.
	// _, err := credentials.NewClientTLSFromFile(cfg.TLSCertPath, "")
	// if err != nil {
	// 	fmt.Printf("cfg Address: %+v\n\n", cfg.Address)
	// 	return nil, fmt.Errorf("unable to read cert file: %v", err)
	// }
	// opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	conn, err := grpc.Dial(cfg.Address, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("unable to dial to dcrlnd's gRPC server: %v", err)
	}

	// Start RPCs.
	pc := pong.NewPongGameClient(conn)

	log := slog.Disabled
	if cfg.Log != nil {
		log = cfg.Log
	}

	return &PongClient{
		updatesCh:  make(chan interface{}),
		pongClient: pc,
		log:        log,
	}, nil
}

type GameStartedMsg struct {
	Started      bool
	PlayerNumber int32
}
type GameUpdateMsg *pong.GameUpdateBytes

func (pc *PongClient) Init(ctx context.Context, req *pong.GameStartedStreamRequest, cb func(pong.PongGame_InitClient)) error {

	gameStartedStream, err := pc.pongClient.Init(context.Background(), req)
	if err != nil {
		return fmt.Errorf("error creating game started stream: %w", err)
	}
	pc.notifier = gameStartedStream
	pc.ID = req.ClientId
	cb(pc.notifier)

	return nil
}

func (pc *PongClient) SendInput(ctx context.Context, input string) error {
	_, err := pc.pongClient.SendInput(ctx, &pong.PlayerInput{
		Input:        input,
		PlayerId:     pc.ID,
		PlayerNumber: pc.playerNumber,
	})
	if err != nil {
		return fmt.Errorf("error sending input: %w", err)
	}
	return nil
}

func (pc *PongClient) SignalReady(ctx context.Context, req *pong.SignalReadyRequest, cb func(pong.PongGame_SignalReadyClient)) error {
	if pc.ID == "" {
		return fmt.Errorf("pongClient is nil")
	}

	req.ClientId = pc.ID

	// Signal readiness after stream is initialized
	stream, err := pc.pongClient.SignalReady(context.Background(), req)
	if err != nil {
		return fmt.Errorf("error signaling readiness: %w", err)
	}
	pc.stream = stream
	cb(pc.stream)

	return nil
}
