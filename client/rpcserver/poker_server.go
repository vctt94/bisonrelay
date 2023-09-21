package rpcserver

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/companyzero/bisonrelay/client"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/rpc"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/slog"
)

// PokerServerCfg is the configuration for a new [types.PokerServiceServer]
// deployment.
type PokerServerCfg struct {
	// Client should be set to the [client.Client] instance.
	Client *client.Client

	// Log should be set to the app's logger.
	Log slog.Logger

	// RootReplayMsgLogs is the root dir where replaymsglogs are stored for
	// supported message types.
	RootReplayMsgLogs string

	// PayClient is the payment client needed to create funded invites.
	PayClient *client.DcrlnPaymentClient

	// InviteFundsAccount is the account to use to generate invite funds.
	// Must be a non-default account in order to generate funds for
	// invites.
	InviteFundsAccount string

	// The following handlers are called when a corresponding request is
	// received via the clientrpc interface. They may be used for displaying
	// the request in a user-friendly way in the client UI or to block the
	// request from propagating (by returning a non-nil error).
	OnPTA func(ctx context.Context, ptid client.PTID, req *types.TARequest) error
}

type pokerServer struct {
	c   *client.Client
	log slog.Logger
	cfg PokerServerCfg

	taStreams *serverStreams[*types.ReceivedTA]
}

func (c *pokerServer) SendFile(_ context.Context, req *types.SendFileRequest, _ *types.SendFileResponse) error {
	user, err := c.c.UserByNick(req.User)
	if err != nil {
		return err
	}

	return c.c.SendFile(user.ID(), req.Filename)
}

func (c *pokerServer) TAStream(ctx context.Context, req *types.TAStreamRequest, stream types.PokerService_TAStreamServer) error {
	return c.taStreams.runStream(ctx, req.UnackedFrom, stream)
}

// pmNtfnHandler is called by the client when a PM arrived from a remote user.
func (c *pokerServer) pmNtfnHandler(ru *client.RemoteUser, p rpc.RMPokerTableAction, ts time.Time) {
	ntfn := &types.ReceivedTA{
		Uid:         ru.ID().Bytes(),
		Nick:        ru.Nick(),
		TimestampMs: ts.UnixMilli(),
		Msg: &types.RMPokerTableAction{
			Action: p.Action,
			Mode:   types.MessageMode(p.Mode),
		},
	}
	fmt.Printf("aqui no (c *pokerServer) pmNtfnHandler")

	c.taStreams.send(ntfn)
}

// AckReceivedPM acks to the server that PMs up to a sequence ID have been
// processed.
func (c *pokerServer) AckReceivedTA(ctx context.Context, req *types.AckRequest,
	res *types.AckResponse) error {
	return c.taStreams.ack(req.SequenceId)
}

// GCM sends a message in a GC.
func (c *pokerServer) PTAct(ctx context.Context, req *types.TARequest, res *types.TAResponse) error {
	fmt.Printf("aqui no TACTION")
	gcid, err := c.c.PTIDByName(req.TableId)
	if err != nil {
		return err
	}
	if c.cfg.OnPTA != nil {
		err = c.cfg.OnPTA(ctx, gcid, req)
		if err != nil {
			return err
		}
	}
	return c.c.PTAction(gcid, req.Action, rpc.MessageModeNormal, nil)
}

func (c *pokerServer) WriteNewInvite(ctx context.Context, req *types.WriteNewInviteRequest, res *types.WriteNewInviteResponse) error {
	var funds *rpc.InviteFunds
	if req.FundAmount > 0 {
		if c.cfg.PayClient == nil {
			return fmt.Errorf("PayClient is nil in pokerServer")
		}
		if c.cfg.InviteFundsAccount == "" || c.cfg.InviteFundsAccount == "default" {
			return fmt.Errorf("cannot generate invite funds in default account")
		}

		var err error
		funds, err = c.cfg.PayClient.CreateInviteFunds(ctx,
			dcrutil.Amount(req.FundAmount), c.cfg.InviteFundsAccount)
		if err != nil {
			return fmt.Errorf("unable to create invite funds: %v", err)
		}
	}

	b := bytes.NewBuffer(nil)
	invite, key, err := c.c.CreatePrepaidInvite(b, funds)
	if err != nil {
		return err
	}
	encKey, err := key.Encode()
	if err != nil {
		return err
	}
	*res = types.WriteNewInviteResponse{
		InviteBytes: b.Bytes(),
		Invite:      marshalOOBPublicIDInvite(&invite, res.Invite),
		InviteKey:   encKey,
	}
	if req.Gc != "" {
		gcid, err := c.c.GCIDByName(req.Gc)
		if err != nil {
			return err
		}
		if err = c.c.AddInviteOnKX(invite.InitialRendezvous, gcid); err != nil {
			return err
		}
	}

	return nil
}

func (c *pokerServer) AcceptTableInvite(_ context.Context, req *types.AcceptInviteRequest, res *types.AcceptInviteResponse) error {
	b := bytes.NewBuffer(req.InviteBytes)
	invite, err := c.c.ReadInvite(b)
	if err != nil {
		return err
	}
	err = c.c.AcceptInvite(invite)
	if err != nil {
		return err
	}
	res.Invite = marshalOOBPublicIDInvite(&invite, res.Invite)
	return nil
}

func (p *pokerServer) AckJoinedTables(context.Context, *types.AckRequest, *types.AckResponse) error {
	return nil
}

func (p *pokerServer) AckMembersAdded(context.Context, *types.AckRequest, *types.AckResponse) error {
	return nil
}

func (p *pokerServer) MembersRemoved(context.Context, *types.TableMembersRemovedRequest, types.PokerService_MembersRemovedServer) error {
	return nil
}

func (p *pokerServer) AckMembersRemoved(context.Context, *types.AckRequest, *types.AckResponse) error {
	return nil
}

func (p *pokerServer) AckReceivedTableInvites(context.Context, *types.AckRequest, *types.AckResponse) error {
	return nil
}

func (p *pokerServer) GetTable(context.Context, *types.GetTableRequest, *types.GetTableResponse) error {
	return nil
}

func (p *pokerServer) InviteToTable(context.Context, *types.InviteToTableRequest, *types.InviteToTableResponse) error {
	return nil
}

func (p *pokerServer) JoinedTables(context.Context, *types.JoinedTablesRequest, types.PokerService_JoinedTablesServer) error {
	return nil
}

func (p *pokerServer) KickFromTable(context.Context, *types.KickFromTableRequest, *types.KickFromTableResponse) error {
	return nil
}

func (p *pokerServer) List(context.Context, *types.ListTablesRequest, *types.ListTablesResponse) error {
	return nil
}

func (p *pokerServer) MembersAdded(context.Context, *types.TableMembersAddedRequest, types.PokerService_MembersAddedServer) error {
	return nil
}

func (p *pokerServer) ReceivedTableInvites(context.Context, *types.ReceivedTableInvitesRequest, types.PokerService_ReceivedTableInvitesServer) error {
	return nil
}

// registerOfflineMessageStorageHandlers registers the handlers for streams on
// the client's notification manager.
func (c *pokerServer) registerOfflineMessageStorageHandlers() {
	nmgr := c.c.NotificationManager()
	nmgr.RegisterSync(client.OnPTANtfn(c.pmNtfnHandler))
}

var _ types.PokerServiceServer = (*pokerServer)(nil)

// InitPokerService initializes and binds a PokerService server to the RPC server.
func (s *Server) InitPokerService(cfg PokerServerCfg) error {
	taStreams, err := newServerStreams[*types.ReceivedTA](cfg.RootReplayMsgLogs, "act", cfg.Log)
	if err != nil {
		return err
	}

	cs := &pokerServer{
		cfg: cfg,
		log: cfg.Log,
		c:   cfg.Client,

		taStreams: taStreams,
	}
	cs.registerOfflineMessageStorageHandlers()
	s.services.Bind("PokerService", types.PokerServiceDefn(), cs)
	return nil
}
