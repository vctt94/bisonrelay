package rpcserver

import (
	"context"

	"github.com/companyzero/bisonrelay/client"
	grpctypes "github.com/companyzero/bisonrelay/clientplugin/grpctypes"
	"github.com/decred/slog"
)

// PluginServerCfg is the configuration for a new PluginServiceServer deployment.
type PluginServerCfg struct {
	Client *client.Client
	Log    slog.Logger
}

type pluginServer struct {
	cfg PluginServerCfg
	c   *client.Client
	log slog.Logger
}

func (p *pluginServer) Init(ctx context.Context, req *grpctypes.PluginStartStreamRequest, stream grpctypes.PluginService_InitServer) error {
	// clientID := req.ClientId

	// p.c.RegisterPluginStream(req.ClientId, stream)

	// player.notifier.Send(&pong.GameStartedStreamResponse{Message: "Notifier stream Initialized"})

	// // Listen for context cancellation to handle disconnection
	// for range ctx.Done() {
	// 	s.handleDisconnect(clientID)
	// 	return ctx.Err()
	// }

	// return nil
	return nil
}

// InitPluginService initializes and binds a PluginService server to the RPC server.
func (s *Server) InitPluginService(cfg PluginServerCfg) error {

	// ps := &pluginServer{
	// 	c:   cfg.Client,
	// 	log: cfg.Log,
	// }
	// s.services.
	// s.services.Bind("PluginService", types.PluginServiceDefn(), ps)
	return nil
}
