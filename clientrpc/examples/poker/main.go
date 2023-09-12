// chat is an example showing how to use the clientrpc to send and receive
// messages.
//
// Messages to send are read from stdin, one message per line, in a <user> <msg>
// format.
//
// Messages received are sent to stdout, in the same format.

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/jsonrpc"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/decred/slog"
	"golang.org/x/sync/errgroup"
)

const ()

func sendLoop(ctx context.Context, chat types.ChatServiceClient, log slog.Logger) error {
	r := bufio.NewScanner(os.Stdin)
	for r.Scan() {
		line := strings.TrimSpace(r.Text())
		if len(line) < 0 {
			continue
		}

		tokens := strings.SplitN(line, " ", 2)
		if len(tokens) != 2 {
			log.Warn("Read line from stdin without 2 tokens")
			continue
		}

		user, msg := tokens[0], tokens[1]
		req := &types.PMRequest{
			User: user,
			Msg: &types.RMPrivateMessage{
				Message: msg,
			},
		}
		var res types.PMResponse
		err := chat.PM(ctx, req, &res)
		if errors.Is(err, context.Canceled) {
			// Program is done.
			return err
		}
		if err != nil {
			// Decide on whether to retry, give up, warn operator,
			// etc.
			log.Warnf("Unable to send last message: %v", err)
			continue
		}

		fmt.Printf("-> %v %v\n", user, msg)
	}
	return r.Err()
}

func receiveLoop(ctx context.Context, chat types.ChatServiceClient, log slog.Logger) error {
	var ackRes types.AckResponse
	var ackReq types.AckRequest
	for {
		// Keep requesting a new stream if the connection breaks. Also
		// request any messages received since the last one we acked.
		streamReq := types.PMStreamRequest{UnackedFrom: ackReq.SequenceId}
		stream, err := chat.PMStream(ctx, &streamReq)
		if errors.Is(err, context.Canceled) {
			// Program is done.
			return err
		}
		if err != nil {
			log.Warn("Error while obtaining PM stream: %v", err)
			time.Sleep(time.Second) // Wait to try again.
			continue
		}

		for {
			var pm types.ReceivedPM
			err := stream.Recv(&pm)
			if errors.Is(err, context.Canceled) {
				// Program is done.
				return err
			}
			if err != nil {
				log.Warnf("Error while receiving stream: %v", err)
				break
			}

			// Escape content before sending it to the terminal.
			nick := escapeNick(pm.Nick)
			var msg string
			if pm.Msg != nil {
				msg = escapeContent(pm.Msg.Message)
			}

			log.Debugf("Received PM from '%s' with len %d and sequence %s",
				nick, len(msg), types.DebugSequenceID(pm.SequenceId))

			fmt.Printf("<- %v %v\n", nick, msg)

			// Ack to client that message is processed.
			ackReq.SequenceId = pm.SequenceId
			err = chat.AckReceivedPM(ctx, &ackReq, &ackRes)
			if err != nil {
				log.Warnf("Error while ack'ing received pm: %v", err)
				break
			}
		}

		time.Sleep(time.Second)
	}
}

func gameLoop(ctx context.Context, poker types.PokerServiceClient, game *Game, log slog.Logger) error {
	var ackRes types.AckResponse
	var ackReq types.AckRequest
	for {
		// Keep requesting a new stream if the connection breaks. Also
		// request any action received since the last one we acked.
		streamReq := types.TAStreamRequest{UnackedFrom: ackReq.SequenceId}
		stream, err := poker.TAStream(ctx, &streamReq)
		if errors.Is(err, context.Canceled) {
			// Program is done.
			return err
		}
		if err != nil {
			log.Warn("Error while obtaining PM stream: %v", err)
			time.Sleep(time.Second) // Wait to try again.
			continue
		}
		for {
			var gcm types.ReceivedTA
			err = stream.Recv(&gcm)
			if errors.Is(err, context.Canceled) {
				// Program is done.
				return err
			}
			nick := escapeNick(gcm.Nick)
			currentPlayer := string(gcm.Msg.CurrentPlayer)

			var msg string
			if gcm.Msg != nil {
				msg = escapeContent(gcm.Msg.Action)
				fmt.Printf("Nick: %v %+v\n\n", nick, gcm.Msg)
			}

			// Handle player actions
			if game.Players[game.CurrentPlayer].ID == currentPlayer {
				if strings.HasPrefix(msg, "bet") {
					chips, err := strconv.Atoi(strings.TrimSpace(msg[len("/bet "):]))
					if err != nil {
						log.Warnf("Invalid bet amount: %v", err)
						continue
					}

					game.Players[game.CurrentPlayer].Chips -= chips
					game.Pot += chips
					fmt.Printf("%s has bet %d chips\n", nick, chips)
				}
				if strings.HasPrefix(msg, "call") {
					fmt.Printf("%s has called", nick)
				}
				if strings.HasPrefix(msg, "fold") {
					fmt.Printf("%s has Folled", nick)
				}

				game.ProgressGame()
				// XXX Make grpc method to update progressed game

				// Ack to client that message is processed.
				ackReq.SequenceId = gcm.SequenceId

				err = poker.AckReceivedTA(ctx, &ackReq, &ackRes)
				if err != nil {
					log.Warnf("Error while ack'ing received pm: %v", err)
					break
				}
			}

		}
	}
}

var (
	flagURL            = flag.String("url", "wss://127.0.0.1:7676/ws", "URL of the websocket endpoint")
	flagServerCertPath = flag.String("servercert", "/home/pokerbot/rpc.cert", "Path to rpc.cert file")
	flagClientCertPath = flag.String("clientcert", "/home/pokerbot/rpc-client.cert", "Path to rpc-client.cert file")
	flagClientKeyPath  = flag.String("clientkey", "/home/pokerbot/rpc-client.key", "Path to rpc-client.key file")
)

func realMain() error {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, gctx := errgroup.WithContext(ctx)

	bknd := slog.NewBackend(os.Stderr)
	log := bknd.Logger("EXMP")
	log.SetLevel(slog.LevelInfo)

	c, err := jsonrpc.NewWSClient(
		jsonrpc.WithWebsocketURL(*flagURL),
		jsonrpc.WithServerTLSCertPath(*flagServerCertPath),
		jsonrpc.WithClientTLSCert(*flagClientCertPath, *flagClientKeyPath),
		jsonrpc.WithClientLog(log),
	)
	if err != nil {
		return err
	}

	g.Go(func() error { return c.Run(gctx) })

	// chat := types.NewChatServiceClient(c)
	payment := types.NewPaymentsServiceClient(c)
	poker := types.NewPokerServiceClient(c)

	// g.Go(func() error { return sendLoop(gctx, chat, log) })
	// g.Go(func() error { return receiveLoop(gctx, chat, log) })

	players := make([]*Player, 2) // Two players for simplicity
	players[0] = &Player{ID: "1", Nick: "Player1", Chips: 1000}
	players[1] = &Player{ID: "2", Nick: "Player2", Chips: 1000}

	game := &Game{
		Bot:     "e3a48ca8e8f300080fe317ac6451ed44400ac717fd196d840320d3930660c71e",
		Deck:    NewDeck().Cards,
		Players: players,
		BB:      0.01,
		SB:      0.01,
		Button:  players[0], // Starting position, can change
		Pot:     0,
	}

	g.Go(func() error { return gameLoop(gctx, poker, game, log) })

	// Deal two cards to each player
	game.ShuffleDeck()
	for _, player := range game.Players {
		player.Hand = []Card{game.Draw(), game.Draw()}
	}

	err = game.ProgressBlinds(ctx, payment)
	if err != nil {
		return err
	}

	// Begin the poker game logic. For this example, we will just print
	// the current state of the game to the console.
	// You can expand upon this by integrating chat interactions.
	for _, player := range game.Players {
		fmt.Printf("%s has %d chips and hand %v and %v\n", player.Nick, player.Chips, player.Hand[0], player.Hand[1])
	}

	game.Flop()
	fmt.Println("The Flop:")
	for i := 0; i < 3; i++ {
		fmt.Printf("%s of %s\n", game.CommunityCards[i].Value, game.CommunityCards[i].Suit)
	}

	game.Turn()
	fmt.Println("\nThe Turn:")
	fmt.Printf("%s of %s\n", game.CommunityCards[3].Value, game.CommunityCards[3].Suit)

	game.River()
	fmt.Println("\nThe River:")
	fmt.Printf("%s of %s\n", game.CommunityCards[4].Value, game.CommunityCards[4].Suit)
	// Further game interactions would follow here, such as betting rounds, board cards, etc.
	// This is just the start, and the chat client's interaction would be the next logical step.

	return g.Wait()
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
