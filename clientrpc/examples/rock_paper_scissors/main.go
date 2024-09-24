package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/jsonrpc"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/decred/slog"
	"golang.org/x/sync/errgroup"
)

const (
	Rock     = "rock"
	Paper    = "paper"
	Scissors = "scissors"
)

type pluginServer struct {
	plugin   types.PluginServiceClient
	gameLock sync.Mutex
	players  map[string]Player
}

func sendLoop(ctx context.Context, rps types.PluginServiceClient, log slog.Logger) error {
	r := bufio.NewScanner(os.Stdin)
	for r.Scan() {
		line := strings.TrimSpace(r.Text())
		if len(line) < 0 {
			continue
		}

		tokens := strings.SplitN(line, " ", 2)
		if len(tokens) != 2 {
			log.Warn("Input should be in format: <user> <move>")
			continue
		}

		player, move := tokens[0], tokens[1]
		if move != Rock && move != Paper && move != Scissors {
			log.Warn("Invalid move, only 'rock', 'paper', or 'scissors' are allowed")
			continue
		}

		// Create the request to send the move
		req := &types.PluginCallActionStreamRequest{
			ClientId: player,
			Action:   move,
		}
		var res types.PluginCallActionStreamResponse
		actionRespStream, err := rps.CallAction(ctx, req)
		actionRespStream.Recv(&res)
		if errors.Is(err, context.Canceled) {
			// Program is done.
			return err
		}
		if err != nil {
			log.Warnf("Unable to send last move: %v", err)
			continue
		}

		fmt.Printf("-> %v played %v\n", player, move)
	}
	return r.Err()
}

func receiveLoop(ctx context.Context, rps types.PluginServiceClient, log slog.Logger) error {
	for {
		// Keep requesting new streams for updates from the server
		streamReq := &types.PluginCallActionStreamRequest{}
		stream, err := rps.CallAction(ctx, streamReq)
		if errors.Is(err, context.Canceled) {
			// Program is done.
			return err
		}
		if err != nil {
			log.Warnf("Error while obtaining game state stream: %v", err)
			time.Sleep(time.Second) // Wait to try again.
			continue
		}

		for {
			var res types.PluginCallActionStreamResponse
			err := stream.Recv(&res)
			if errors.Is(err, context.Canceled) {
				// Program is done.
				return err
			}
			if err != nil {
				log.Warnf("Error while receiving stream: %v", err)
				break
			}

			// Display the result or state received from the server
			fmt.Printf("<- Result: %v\n", string(res.Response))
		}

		time.Sleep(time.Second)
	}
}

var (
	flagURL            = flag.String("url", "wss://127.0.0.1:7676/ws", "URL of the websocket endpoint")
	flagServerCertPath = flag.String("servercert", "~/.brclient/rpc.cert", "Path to rpc.cert file")
	flagClientCertPath = flag.String("clientcert", "~/.brclient/rpc-client.cert", "Path to rpc-client.cert file")
	flagClientKeyPath  = flag.String("clientkey", "~/.brclient/rpc-client.key", "Path to rpc-client.key file")
)

func realMain() error {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, gctx := errgroup.WithContext(ctx)

	bknd := slog.NewBackend(os.Stderr)
	log := bknd.Logger("RPS")
	log.SetLevel(slog.LevelInfo)

	// Create the WebSocket client
	c, err := jsonrpc.NewWSClient(
		jsonrpc.WithWebsocketURL(*flagURL),
		jsonrpc.WithServerTLSCertPath(*flagServerCertPath),
		jsonrpc.WithClientTLSCert(*flagClientCertPath, *flagClientKeyPath),
		jsonrpc.WithClientLog(log),
	)
	if err != nil {
		return err
	}

	// Initialize the RPS game client
	rps := types.NewPluginServiceClient(c)

	// Start the WebSocket client and communication loops
	g.Go(func() error { return c.Run(gctx) })
	g.Go(func() error { return sendLoop(gctx, rps, log) })
	g.Go(func() error { return receiveLoop(gctx, rps, log) })

	// Wait for completion
	return g.Wait()
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
