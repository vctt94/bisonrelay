package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/jsonrpc"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/slog"
	"golang.org/x/sync/errgroup"
)

var (
	flagURL            = flag.String("url", "wss://127.0.0.1:7676/ws", "URL of the websocket endpoint")
	flagServerCertPath = flag.String("servercert", "/home/pokerbot/brclient/rpc.cert", "Path to rpc.cert file")
	flagClientCertPath = flag.String("clientcert", "/home/pokerbot/brclient/rpc-client.cert", "Path to rpc-client.cert file")
	flagClientKeyPath  = flag.String("clientkey", "/home/pokerbot/brclient/rpc-client.key", "Path to rpc-client.key file")
	game               *PokerGame
	botId              string
	gameMutex          sync.Mutex
)

func sendPaymentLoop(ctx context.Context, payment types.PaymentsServiceClient, chat types.ChatServiceClient, log slog.Logger) error {
	var ackRes types.AckResponse
	var ackReq types.AckRequest
	for {
		// Keep requesting a new stream if the connection breaks. Also
		// request any messages received since the last one we acked.
		streamReq := types.TipProgressRequest{UnackedFrom: ackReq.SequenceId}
		stream, err := payment.TipProgress(ctx, &streamReq)
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
			var tip types.TipProgressEvent
			err := stream.Recv(&tip)
			if errors.Is(err, context.Canceled) {
				// Program is done.
				return err
			}
			if err != nil {
				log.Warnf("Error while receiving stream: %v", err)
				break
			}

			// Escape content before sending it to the terminal.
			nick := escapeNick(tip.Nick)
			var amount int64
			if tip.AmountMatoms > 0 {
				amount = tip.AmountMatoms
			}

			ruid := tip.Uid
			uid := hex.EncodeToString(ruid[:])

			var botReply string
			if tip.IsSending {
				continue
			}
			if game != nil {
				gameMutex.Lock()
				if game.CurrentStage == Draw {
					if uid == game.Players[game.BigBlind].ID {
						game.Players[game.BigBlind].HasActed = true
						game.Pot += game.BB
						botReply = "Big Blind Paid"
					}
					if uid == game.Players[game.SmallBlind].ID {
						game.Players[game.SmallBlind].HasActed = true
						game.Pot += game.SB
						botReply = "Small Blind Paid"
					}
					// only progress if all players paid
					if game.AllPlayersActed() {
						game.ProgressPokerGame()
						botReply += fmt.Sprintf("\n|---------------\n"+
							"Current Stage: %s\n"+
							"Community Cards: %v\n"+
							"Pot: %f\n"+
							"---------------|\n"+
							"Current Player: %s\n",

							game.CurrentStage, game.CommunityCards, game.Pot, game.Players[game.CurrentPlayer].Nick)
					}
				} else {
					currentPlayer := game.Players[game.CurrentPlayer]
					value := float64(tip.AmountMatoms)
					if uid == currentPlayer.ID {
						// Identify the action based on the tip amount and current bet
						if value == game.CurrentBet {
							// Call
							game.Call()
							botReply = fmt.Sprintf("Player %s called.", currentPlayer.Nick)
						} else if value > game.CurrentBet {
							if game.CurrentBet == 0 {
								// Bet
								game.Bet(value)
								botReply = fmt.Sprintf("Player %s has bet %f. Current pot is %f", game.Players[game.CurrentPlayer].Nick, value, game.Pot)
							} else {
								// Raise
								game.Raise(value)
								botReply = fmt.Sprintf("Player %s raised to: %f", currentPlayer.Nick, game.CurrentBet)
							}
						} else {
							botReply = "Invalid Bet Amount"
						}
					}
				}
				gameMutex.Unlock()
			}

			req := &types.GCMRequest{
				Gc:  game.ID,
				Msg: botReply,
			}
			var res types.GCMResponse
			err = chat.GCM(ctx, req, &res)
			if err != nil {
				log.Warnf("Err: %v", err)
				continue
			}

			log.Debugf("Received tip from '%s' with amount %d and sequence %s",
				nick, amount, types.DebugSequenceID(tip.SequenceId))

			// Ack to client that tip is processed.
			if tip.Completed {
				ackReq.SequenceId = tip.SequenceId
				err = payment.AckTipProgress(ctx, &ackReq, &ackRes)
				if err != nil {
					log.Warnf("Error while ack'ing received pm: %v", err)
					break
				}
			}
		}

		time.Sleep(time.Second)
	}
}

func gameLoop(ctx context.Context, chat types.ChatServiceClient, gcService types.GCServiceClient, payment types.PaymentsServiceClient, log slog.Logger) error {
	// var ackRes types.AckResponse
	var ackReq types.AckRequest
	for {
		// Keep requesting a new stream if the connection breaks. Also
		// request any action received since the last one we acked.
		streamReq := types.GCMStreamRequest{UnackedFrom: ackReq.SequenceId}
		stream, err := chat.GCMStream(ctx, &streamReq)
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
			var gcm types.GCReceivedMsg
			err = stream.Recv(&gcm)
			if errors.Is(err, context.Canceled) {
				// Program is done.
				return err
			}

			if gcm.Msg == nil {
				continue
			}

			var msg, botReply, gcidstr, uid string
			var resp *types.GetGCResponse
			var err error
			msg = escapeContent(gcm.Msg.Message)
			gcid, err := zkidentity.Byte2ID(gcm.Msg.Id)
			if err != nil {
				return err
			}
			gcidstr = hex.EncodeToString((*gcid)[:])
			resp = &types.GetGCResponse{}
			uid = hex.EncodeToString(gcm.Uid[:])

			err = gcService.GetGC(ctx, &types.GetGCRequest{Gc: gcidstr}, resp)
			if err != nil {
				log.Warnf("Error while listing gcs: %v", err)
				continue
			}
			if resp.Gc == nil {
				log.Warnf("not possible to find gc: %s", gcm.GcAlias)
				continue
			}

			// commands
			// start
			if strings.HasPrefix(msg, "!start") {
				info := &types.InfoResponse{}
				err := chat.Info(ctx, &types.InfoRequest{}, info)
				if err != nil {
					log.Warnf("err: %v", err)
					continue
				}
				botId = hex.EncodeToString((info.Identity)[:])

				n := len(resp.Gc.Members)
				if n < 3 {
					botReply = "minimum 2 players to start game"
					err = sendMessage(ctx, chat, gcidstr, botReply)
					if err != nil {
						log.Warnf("not possible to send message: %s", err)
						continue
					}
					continue
				}
				gameMutex.Lock()

				players := make([]Player, n)
				for i, member := range resp.Gc.Members {
					mid, err := zkidentity.Byte2ID(member)
					if err != nil {
						return err
					}
					ruid := hex.EncodeToString((*mid)[:])
					var resp types.InfoResponse
					// deactivate bot
					if ruid == botId {
						players[i] = Player{
							ID:       ruid,
							IsActive: false,
						}
						continue
					}
					err = chat.Info(ctx, &types.InfoRequest{User: ruid}, &resp)
					if err != nil {
						log.Warnf("not possible to chat.Info: %s", err)
						continue
					}
					players[i] = Player{
						ID:       ruid,
						Nick:     resp.Nick,
						Hand:     []Card{},
						IsActive: true,
						HasActed: false,
					}

				}
				game = New(gcidstr, players, 0, 0.005, 0.01)
				game.ShuffleDeck()

				// blinds need to be paid
				game.Players[game.BigBlind].HasActed = false
				game.Players[game.SmallBlind].HasActed = false

				// draw cards
				for i := range players {
					if !players[i].IsActive {
						continue
					}
					players[i].Hand = []Card{game.Draw(), game.Draw()}
					playerHandReq := &types.PMRequest{
						User: players[i].ID,
						Msg: &types.RMPrivateMessage{
							Message: fmt.Sprintf("Hand: %v\n"+
								"___________________________________", players[i].Hand),
						},
					}
					var playerHandRes types.PMResponse
					err = chat.PM(ctx, playerHandReq, &playerHandRes)
					if err != nil {
						log.Warnf("Err: %v", err)
						continue
					}
				}
				gameMutex.Unlock()

				botReply = fmt.Sprintf("\n---------------\n"+
					"Current Stage: %s\n"+
					"Community Cards: %v\n"+
					"Pot: %f\n"+
					"---------------|\n"+
					fmt.Sprintf("Waiting for:\nBB: %f from %s\nSB: %f from %s\n", game.BB, game.Players[game.BigBlind].Nick, game.SB, game.Players[game.SmallBlind].Nick)+
					"Current Player: %s\n",

					game.CurrentStage, game.CommunityCards, game.Pot, game.Players[game.CurrentPlayer].Nick)

				if err != nil {
					log.Warnf("Err: %v", err)
					continue
				}
				req := &types.GCMRequest{
					Gc:  gcidstr,
					Msg: botReply,
				}
				var res types.GCMResponse
				err = chat.GCM(ctx, req, &res)
				if err != nil {
					log.Warnf("Err: %v", err)
					continue
				}
				// game started can continue.
				continue
			}

			// commands from here need the game started.
			if game == nil {
				continue
			}

			if game.Players[game.CurrentPlayer].ID != uid {
				botReply = "Not your turn to play"
				msg = ""
			}
			// check
			if strings.HasPrefix(msg, "!check") {
				gameMutex.Lock()
				game.Players[game.CurrentPlayer].HasActed = true
				game.ProgressPokerGame()
				if game.CurrentStage == Showdown {
					// XXX distribute pot
					winner := game.Winner[0]
					botReply = fmt.Sprintf("\n---------------\n"+
						"Current Stage: %s\n"+
						"Community Cards: %v\n"+
						"Pot: %f\n"+
						"Winner Player: %s\n"+
						"---------------|\n",
						game.CurrentStage, game.CommunityCards, game.Pot, game.Players[winner].Nick)
					err = payment.TipUser(ctx, &types.TipUserRequest{
						User:        game.Players[winner].ID,
						DcrAmount:   game.Pot,
						MaxAttempts: 1,
					}, &types.TipUserResponse{})

					if err != nil {
						log.Warnf("Error while sending pot: %v", err)
					}
				} else {
					botReply = fmt.Sprintf("\n---------------\n"+
						"Current Stage: %s\n"+
						"Community Cards: %v\n"+
						"Pot: %f\n"+
						"Current Player: %s\n"+
						"---------------|\n",
						game.CurrentStage, game.CommunityCards, game.Pot, game.Players[game.CurrentPlayer].Nick)
				}
				gameMutex.Unlock()
			}

			if strings.HasPrefix(msg, "!fold") {
				gameMutex.Lock()
				// Mark the player as folded
				game.Players[game.CurrentPlayer].Folded = true
				game.Players[game.CurrentPlayer].HasActed = true
				game.ProgressPokerGame()
				botReply = fmt.Sprintf("Player %s has folded", game.Players[game.CurrentPlayer].Nick)
				gameMutex.Unlock()
			}

			req := &types.GCMRequest{
				Gc:  gcidstr,
				Msg: botReply,
			}
			var res types.GCMResponse
			err = chat.GCM(ctx, req, &res)
			if err != nil {
				log.Warnf("Err: %v", err)
				continue
			}
		}
	}
}

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

	payment := types.NewPaymentsServiceClient(c)
	chat := types.NewChatServiceClient(c)
	gc := types.NewGCServiceClient(c)

	g.Go(func() error { return gameLoop(gctx, chat, gc, payment, log) })
	g.Go(func() error { return sendPaymentLoop(gctx, payment, chat, log) })

	return g.Wait()
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
