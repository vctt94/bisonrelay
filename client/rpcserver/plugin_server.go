package rpcserver

import (
	"context"
	"time"

	"github.com/companyzero/bisonrelay/client"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/decred/slog"
)

// ChatServerCfg is the configuration for a new [types.ChatServiceServer]
// deployment.
type PluginServerCfg struct {
	// Client should be set to the [client.Client] instance.
	Client *client.Client

	// Log should be set to the app's logger.
	Log slog.Logger

	// RootReplayMsgLogs is the root dir where replaymsglogs are stored for
	// supported message types.
	RootReplayMsgLogs string
	// The following handlers are called when a corresponding request is
	// received via the clientrpc interface. They may be used for displaying
	// the request in a user-friendly way in the client UI or to block the
	// request from propagating (by returning a non-nil error).

	OnInit   func(ctx context.Context, uid client.UserID, req *types.PluginStartStreamResponse) error
	OnAction func(ctx context.Context, uid client.UserID, req *types.PluginCallActionStreamResponse) error
}

type pluginServer struct {
	c   *client.Client
	log slog.Logger
	cfg PluginServerCfg

	nftnStreams   *serverStreams[*types.PluginStartStreamResponse]
	actionStreams *serverStreams[*types.PluginCallActionStreamResponse]
}

func (c *pluginServer) Input(ctx context.Context, req *types.PMRequest, res *types.PMResponse) error {
	return nil
}

func (c *pluginServer) NtfnStream(ctx context.Context, req *types.PMStreamRequest, stream types.PluginService_InitServer) error {
	return c.nftnStreams.runStream(ctx, req.UnackedFrom, stream)
}

// pmNtfnHandler is called by the client when a PM arrived from a remote user.
func (c *pluginServer) ntfnHandler(ru *client.RemoteUser, message string, ts time.Time) {
	ntfn := &types.PluginStartStreamResponse{
		ClientId: ru.ID().String(),
		Message:  message,
	}

	c.nftnStreams.send(ntfn)
}

// InitChatService initializes and binds a ChatService server to the RPC server.
func (s *Server) InitPluginService(cfg PluginServerCfg) error {
	ntfnStreams, err := newServerStreams[*types.PluginStartStreamResponse](cfg.RootReplayMsgLogs, "start", cfg.Log)
	if err != nil {
		return err
	}

	actionStreams, err := newServerStreams[*types.PluginCallActionStreamResponse](cfg.RootReplayMsgLogs, "action", cfg.Log)
	if err != nil {
		return err
	}

	cs := &pluginServer{
		cfg: cfg,
		log: cfg.Log,
		c:   cfg.Client,

		nftnStreams:   ntfnStreams,
		actionStreams: actionStreams,
	}
	s.services.Bind("PluginService", types.ChatServiceDefn(), cs)
	return nil
}
