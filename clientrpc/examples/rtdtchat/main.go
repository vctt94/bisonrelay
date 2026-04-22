// rtdtchat is an example showing how to use the clientrpc RTDT service to send
// and receive RTDT chat messages and live audio in a single session.
//
// Chat messages to send are read from stdin, one message per line.
//
// Audio is captured from the local microphone and sent to the RTDT session.
// Incoming RTDT speech packets are decoded and played to the local playback
// device.
//
// Received chat messages are written to stdout in a "<- <nick> <msg>" format.

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/jsonrpc"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/internal/audio"
	"github.com/decred/slog"
	"golang.org/x/sync/errgroup"
)

func ensureJoinedSession(ctx context.Context, rtdt types.RTDTServiceClient, session string, log slog.Logger) error {
	req := &types.GetRTDTSessionRequest{SessionRv: session}
	var res types.GetRTDTSessionResponse
	if err := rtdt.GetSession(ctx, req, &res); err != nil {
		return err
	}
	if res.Session != nil && res.Session.Live != nil {
		return nil
	}

	log.Infof("Joining live RTDT session %s", session)
	joinReq := &types.JoinLiveRTDTSessionRequest{SessionRv: session}
	var joinRes types.JoinLiveRTDTSessionResponse
	return rtdt.JoinSession(ctx, joinReq, &joinRes)
}

func sendLoop(ctx context.Context, rtdt types.RTDTServiceClient, session string, log slog.Logger) error {
	r := bufio.NewScanner(os.Stdin)
	for r.Scan() {
		msg := strings.TrimSpace(r.Text())
		if msg == "" {
			continue
		}

		req := &types.SendRTDTChatMessageRequest{
			SessionRv: session,
			Message:   msg,
		}
		var res types.SendRTDTChatMessageResponse
		err := rtdt.SendChatMessage(ctx, req, &res)
		if errors.Is(err, context.Canceled) {
			return err
		}
		if err != nil {
			log.Warnf("Unable to send last RTDT message: %v", err)
			continue
		}

		fmt.Printf("-> %v\n", escapeContent(msg))
	}
	return r.Err()
}

func receiveLoop(ctx context.Context, rtdt types.RTDTServiceClient, session string, log slog.Logger) error {
	var ackRes types.AckResponse
	var ackReq types.AckRequest
	for {
		streamReq := types.RTDTChatMessagesStreamRequest{UnackedFrom: ackReq.SequenceId}
		stream, err := rtdt.ChatMessagesStream(ctx, &streamReq)
		if errors.Is(err, context.Canceled) {
			return err
		}
		if err != nil {
			log.Warnf("Error while obtaining RTDT chat stream: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for {
			var msg types.RTDTChatMessageReceivedEvent
			err := stream.Recv(&msg)
			if errors.Is(err, context.Canceled) {
				return err
			}
			if err != nil {
				log.Warnf("Error while receiving RTDT chat stream: %v", err)
				break
			}

			nick := "unknown"
			if msg.Publisher != nil {
				if msg.Publisher.Alias != "" {
					nick = msg.Publisher.Alias
				} else if msg.Publisher.PublisherId != "" {
					nick = msg.Publisher.PublisherId
				}
			}
			nick = escapeNick(nick)
			content := escapeContent(msg.Message)

			log.Debugf("Received RTDT chat message from '%s' in session %s sequence %s",
				nick, msg.SessionRv, types.DebugSequenceID(msg.SequenceId))

			if msg.SessionRv == session {
				fmt.Printf("<- %v %v\n", nick, content)
			}

			ackReq.SequenceId = msg.SequenceId
			err = rtdt.AckChatMessages(ctx, &ackReq, &ackRes)
			if err != nil {
				log.Warnf("Error while ack'ing RTDT chat message: %v", err)
				break
			}
		}

		time.Sleep(time.Second)
	}
}

func captureAudioLoop(ctx context.Context, rec *audio.NoteRecorder, rtdt types.RTDTServiceClient, session string, log slog.Logger) error {
	log.Infof("Starting microphone capture")
	cs, err := rec.CaptureStream(ctx, func(ctx context.Context, data []byte, timestamp uint32) error {
		req := &types.SendRTDTAudioPacketRequest{
			SessionRv:  session,
			Timestamp:  timestamp,
			OpusPacket: data,
		}
		var res types.SendRTDTAudioPacketResponse
		return rtdt.SendAudioPacket(ctx, req, &res)
	})
	if err != nil {
		return err
	}

	<-cs.CaptureDone()
	return cs.Err()
}

func receiveAudioLoop(ctx context.Context, rec *audio.NoteRecorder, rtdt types.RTDTServiceClient, session string, log slog.Logger) error {
	streamReq := &types.RTDTAudioPacketsStreamRequest{SessionRv: session}
	playbackStreams := make(map[uint32]*audio.PlaybackStream)
	defer func() {
		for _, ps := range playbackStreams {
			ps.MarkInputDone(ctx)
		}
	}()
	for {
		stream, err := rtdt.AudioPacketsStream(ctx, streamReq)
		if errors.Is(err, context.Canceled) {
			return err
		}
		if err != nil {
			log.Warnf("Error while obtaining RTDT audio stream: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for {
			var pkt types.RTDTAudioPacketReceivedEvent
			err := stream.Recv(&pkt)
			if errors.Is(err, context.Canceled) {
				return err
			}
			if err != nil {
				log.Warnf("Error while receiving RTDT audio stream: %v", err)
				break
			}

			ps := playbackStreams[pkt.PeerId]
			if ps == nil {
				log.Infof("Starting playback stream for peer %08x", pkt.PeerId)
				ps = rec.PlaybackStream(ctx, nil)
				playbackStreams[pkt.PeerId] = ps
			}

			log.Debugf("Received RTDT audio packet from peer %08x in session %s ts %d len %d",
				pkt.PeerId, pkt.SessionRv, pkt.Timestamp, len(pkt.OpusPacket))
			ps.Input(pkt.OpusPacket, pkt.Timestamp)
		}

		time.Sleep(time.Second)
	}
}

var (
	flagURL            = flag.String("url", "wss://127.0.0.1:7676/ws", "URL of the websocket endpoint")
	flagServerCertPath = flag.String("servercert", "~/.brclient/rpc.cert", "Path to rpc.cert file")
	flagClientCertPath = flag.String("clientcert", "~/.brclient/rpc-client.cert", "Path to rpc-client.cert file")
	flagClientKeyPath  = flag.String("clientkey", "~/.brclient/rpc-client.key", "Path to rpc-client.key file")
	flagRPCUser        = flag.String("rpcuser", "rpcuser", "RPC user for basic auth")
	flagRPCPass        = flag.String("rpcpass", "rpcpass", "RPC password for basic auth")
	flagSession        = flag.String("session", "", "RTDT session RV to join and use for chat")
	flagJoin           = flag.Bool("join", true, "Join the live RTDT session on startup if not already joined")
	flagSendAudio      = flag.Bool("sendaudio", true, "Capture microphone audio and send it to the RTDT session")
	flagPlayAudio      = flag.Bool("playaudio", true, "Play incoming RTDT audio packets to the playback device")
	flagCaptureDevice  = flag.String("capturedevice", "", "Capture device ID to use instead of the default")
	flagPlaybackDevice = flag.String("playbackdevice", "", "Playback device ID to use instead of the default")
)

func realMain() error {
	flag.Parse()
	if *flagSession == "" {
		return fmt.Errorf("-session is required")
	}

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
		jsonrpc.WithClientBasicAuth(*flagRPCUser, *flagRPCPass),
	)
	if err != nil {
		return err
	}

	rtdt := types.NewRTDTServiceClient(c)
	g.Go(func() error { return c.Run(gctx) })

	var rec *audio.NoteRecorder
	if *flagSendAudio || *flagPlayAudio {
		rec, err = audio.NewRecorder(log)
		if err != nil {
			return err
		}
		defer rec.FreeContext()

		if *flagCaptureDevice != "" {
			if err := rec.SetCaptureDevice(audio.DeviceID(*flagCaptureDevice)); err != nil {
				return err
			}
		}
		if *flagPlaybackDevice != "" {
			if err := rec.SetPlaybackDevice(audio.DeviceID(*flagPlaybackDevice)); err != nil {
				return err
			}
		}
	}

	if *flagJoin {
		if err := ensureJoinedSession(gctx, rtdt, *flagSession, log); err != nil {
			return err
		}
	}

	g.Go(func() error { return sendLoop(gctx, rtdt, *flagSession, log) })
	g.Go(func() error { return receiveLoop(gctx, rtdt, *flagSession, log) })
	if rec != nil && *flagSendAudio {
		g.Go(func() error { return captureAudioLoop(gctx, rec, rtdt, *flagSession, log) })
	}
	if rec != nil && *flagPlayAudio {
		g.Go(func() error { return receiveAudioLoop(gctx, rec, rtdt, *flagSession, log) })
	}

	return g.Wait()
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
