// rtdtaudio is an example showing how to use the clientrpc RTDT service to
// capture microphone audio in an external app and forward it to a live RTDT
// session handled by brclient.
//
// By default this example only sends microphone audio to brclient. Use
// -playaudio to also play remote RTDT audio received through clientrpc.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/companyzero/bisonrelay/clientrpc/jsonrpc"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/internal/audio"
	"github.com/companyzero/bisonrelay/internal/strescape"
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

func captureAudioLoop(ctx context.Context, rec *audio.NoteRecorder, rtdt types.RTDTServiceClient, session string) error {
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

			ps.Input(pkt.OpusPacket, pkt.Timestamp)
		}

		time.Sleep(time.Second)
	}
}

func printDevices(devices *audio.Devices) {
	printSet := func(label string, devs []audio.Device) {
		if len(devs) == 0 {
			fmt.Printf("No %s devices found\n", label)
			return
		}

		fmt.Printf("%s devices:\n", label)
		for i := range devs {
			dev := devs[i]
			defaultStr := ""
			if dev.IsDefault {
				defaultStr = " (default)"
			}
			fmt.Printf("  %d. %s%s\n", i, dev.Name, defaultStr)
			fmt.Printf("     id: %s\n", strescape.Nick(dev.ID.String()))
		}
	}

	printSet("Capture", devices.Capture)
	printSet("Playback", devices.Playback)
}

var (
	flagURL            = flag.String("url", "wss://127.0.0.1:7676/ws", "URL of the websocket endpoint")
	flagServerCertPath = flag.String("servercert", "~/.brclient/rpc.cert", "Path to rpc.cert file")
	flagClientCertPath = flag.String("clientcert", "~/.brclient/rpc-client.cert", "Path to rpc-client.cert file")
	flagClientKeyPath  = flag.String("clientkey", "~/.brclient/rpc-client.key", "Path to rpc-client.key file")
	flagRPCUser        = flag.String("rpcuser", "rpcuser", "RPC user for basic auth")
	flagRPCPass        = flag.String("rpcpass", "rpcpass", "RPC password for basic auth")
	flagSession        = flag.String("session", "", "RTDT session RV to join and use for audio")
	flagJoin           = flag.Bool("join", true, "Join the live RTDT session on startup if not already joined")
	flagPlayAudio      = flag.Bool("playaudio", false, "Play incoming RTDT audio packets to the playback device")
	flagListDevices    = flag.Bool("lsdev", false, "List audio devices and quit")
	flagCaptureDevice  = flag.String("capturedevice", "", "Capture device ID to use instead of the default")
	flagPlaybackDevice = flag.String("playbackdevice", "", "Playback device ID to use instead of the default")
)

func realMain() error {
	flag.Parse()

	bknd := slog.NewBackend(os.Stderr)
	log := bknd.Logger("EXMP")
	log.SetLevel(slog.LevelInfo)

	if *flagListDevices {
		devices, err := audio.ListAudioDevices(log)
		if err != nil {
			return err
		}
		printDevices(&devices)
		return nil
	}

	if *flagSession == "" {
		return fmt.Errorf("-session is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	g, gctx := errgroup.WithContext(ctx)

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

	rec, err := audio.NewRecorder(log)
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

	rtdt := types.NewRTDTServiceClient(c)
	g.Go(func() error { return c.Run(gctx) })

	if *flagJoin {
		if err := ensureJoinedSession(gctx, rtdt, *flagSession, log); err != nil {
			return err
		}
	}

	log.Infof("Capturing microphone audio for RTDT session %s", *flagSession)
	g.Go(func() error { return captureAudioLoop(gctx, rec, rtdt, *flagSession) })
	if *flagPlayAudio {
		log.Infof("Playing incoming RTDT audio for session %s", *flagSession)
		g.Go(func() error { return receiveAudioLoop(gctx, rec, rtdt, *flagSession, log) })
	}

	err = g.Wait()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func main() {
	err := realMain()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
