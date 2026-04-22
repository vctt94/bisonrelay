package rpcserver

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/companyzero/bisonrelay/client"
	"github.com/companyzero/bisonrelay/client/clientdb"
	"github.com/companyzero/bisonrelay/clientrpc/types"
	"github.com/companyzero/bisonrelay/rpc"
	"github.com/companyzero/bisonrelay/zkidentity"
	"github.com/decred/slog"
)

// RTDTServerCfg is the configuration for the RTDT clientrpc service.
type RTDTServerCfg struct {
	// Client should be set to the [client.Client] instance.
	Client *client.Client

	// Log should be set to the app's logger.
	Log slog.Logger

	// RootReplayMsgLogs is the root dir where replaymsglogs are stored for
	// supported message types.
	RootReplayMsgLogs string
}

type rtdtServer struct {
	cfg RTDTServerCfg
	log slog.Logger
	c   *client.Client

	inviteStreams  *serverStreams[*types.ReceivedRTDTInvite]
	updateStreams  *serverStreams[*types.RTDTSessionUpdatedEvent]
	messageStreams *serverStreams[*types.RTDTChatMessageReceivedEvent]
	audioStreams   *liveServerStreams[*types.RTDTAudioPacketReceivedEvent]
}

type filteredRTDTAudioPacketStream struct {
	sessionRV string
	stream    types.RTDTService_AudioPacketsStreamServer
}

func (f filteredRTDTAudioPacketStream) Send(m *types.RTDTAudioPacketReceivedEvent) error {
	if f.sessionRV != "" && m.SessionRv != f.sessionRV {
		return nil
	}
	return f.stream.Send(m)
}

func parseShortIDHex(s string) (zkidentity.ShortID, error) {
	var id zkidentity.ShortID
	if err := id.FromString(strings.TrimSpace(s)); err != nil {
		return id, err
	}
	return id, nil
}

func idsToStrings(ids []zkidentity.ShortID) []string {
	res := make([]string, len(ids))
	for i := range ids {
		res[i] = ids[i].String()
	}
	return res
}

func marshalRTDTSessionPublisher(pub rpc.RMRTDTSessionPublisher) *types.RTDTSessionPublisher {
	return &types.RTDTSessionPublisher{
		PublisherId: pub.PublisherID.String(),
		PeerId:      uint32(pub.PeerID),
		Alias:       pub.Alias,
	}
}

func marshalRTDTLiveSession(live *client.LiveRTDTSession) *types.RTDTLiveSession {
	if live == nil {
		return nil
	}

	peerIDs := make([]uint32, 0, len(live.Peers))
	for peerID := range live.Peers {
		peerIDs = append(peerIDs, uint32(peerID))
	}
	slices.Sort(peerIDs)

	res := &types.RTDTLiveSession{
		HotAudio: live.HotAudio,
		Peers:    make([]*types.RTDTLivePeer, 0, len(peerIDs)),
	}
	for _, peerID := range peerIDs {
		peer := live.Peers[rpc.RTDTPeerID(peerID)]
		res.Peers = append(res.Peers, &types.RTDTLivePeer{
			PeerId:         peerID,
			HasSoundStream: peer.HasSoundStream,
			HasSound:       peer.HasSound,
			VolumeGain:     peer.VolumeGain,
			BufferedCount:  peer.BufferedCount,
		})
	}
	return res
}

func marshalRTDTSession(sess *clientdb.RTDTSession, live *client.LiveRTDTSession) *types.RTDTSession {
	res := &types.RTDTSession{
		Metadata: &types.RTDTSessionMetadata{
			Generation:  sess.Metadata.Generation,
			SessionRv:   sess.Metadata.RV.String(),
			Size:        sess.Metadata.Size,
			Description: sess.Metadata.Description,
			Owner:       sess.Metadata.Owner.String(),
			Admins:      idsToStrings(sess.Metadata.Admins),
			IsInstant:   sess.Metadata.IsInstant,
		},
		LocalPeerId:      uint32(sess.LocalPeerID),
		NextPeerId:       uint32(sess.NextPeerID),
		Gc:               "",
		LocalIsAdmin:     sess.LocalIsAdmin(),
		LocalIsPublisher: sess.PublisherKey != nil,
		Live:             marshalRTDTLiveSession(live),
	}

	if sess.GC != nil {
		res.Gc = sess.GC.String()
	}

	res.Metadata.Publishers = make([]*types.RTDTSessionPublisher, 0, len(sess.Metadata.Publishers))
	for _, pub := range sess.Metadata.Publishers {
		res.Metadata.Publishers = append(res.Metadata.Publishers, marshalRTDTSessionPublisher(pub))
	}

	res.Members = make([]*types.RTDTSessionMember, 0, len(sess.Members))
	for _, member := range sess.Members {
		accepted := member.AcceptedTimestamp != nil
		var acceptedTs int64
		if accepted {
			acceptedTs = *member.AcceptedTimestamp
		}
		res.Members = append(res.Members, &types.RTDTSessionMember{
			Uid:               member.UID.String(),
			PeerId:            uint32(member.PeerID),
			Tag:               member.Tag,
			SentTimestamp:     member.SentTimestamp,
			Publisher:         member.Publisher,
			Accepted:          accepted,
			AcceptedTimestamp: acceptedTs,
			IsAdmin:           sess.Metadata.IsOwnerOrAdmin(member.UID),
			IsOwner:           sess.Metadata.Owner == member.UID,
		})
	}

	return res
}

func marshalRTDTInvite(invite *rpc.RMRTDTSessionInvite) *types.RTDTSessionInvite {
	res := &types.RTDTSessionInvite{
		SessionRv:          invite.RV.String(),
		Size:               invite.Size,
		Description:        invite.Description,
		AllowedAsPublisher: invite.AllowedAsPublisher,
		PeerId:             uint32(invite.PeerID),
		IsInstant:          invite.IsInstant,
	}
	if invite.GC != nil {
		res.Gc = invite.GC.String()
	}
	return res
}

func (r *rtdtServer) getSession(rv *zkidentity.ShortID) (*types.RTDTSession, error) {
	sess, err := r.c.GetRTDTSession(rv)
	if err != nil {
		return nil, err
	}
	return marshalRTDTSession(sess, r.c.GetLiveRTSession(rv)), nil
}

func (r *rtdtServer) resolveUsers(users []string) ([]client.UserID, error) {
	res := make([]client.UserID, len(users))
	for i, user := range users {
		uid, err := r.c.UIDByNick(user)
		if err != nil {
			return nil, err
		}
		res[i] = uid
	}
	return res, nil
}

func (r *rtdtServer) ListSessions(_ context.Context, _ *types.ListRTDTSessionsRequest, res *types.ListRTDTSessionsResponse) error {
	rvs := r.c.ListRTDTSessions()
	slices.SortFunc(rvs, func(a, b zkidentity.ShortID) int {
		return strings.Compare(a.String(), b.String())
	})

	res.Sessions = make([]*types.RTDTSession, 0, len(rvs))
	for _, rv := range rvs {
		sess, err := r.getSession(&rv)
		if err != nil {
			return err
		}
		res.Sessions = append(res.Sessions, sess)
	}
	return nil
}

func (r *rtdtServer) GetSession(_ context.Context, req *types.GetRTDTSessionRequest, res *types.GetRTDTSessionResponse) error {
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}

	res.Session, err = r.getSession(&rv)
	return err
}

func (r *rtdtServer) CreateSession(_ context.Context, req *types.CreateRTDTSessionRequest, res *types.CreateRTDTSessionResponse) error {
	if req.Size > math.MaxUint16 {
		return fmt.Errorf("size %d exceeds max allowed %d", req.Size, math.MaxUint16)
	}

	sess, err := r.c.CreateRTDTSession(uint16(req.Size), req.Description)
	if err != nil {
		return err
	}

	res.Session = marshalRTDTSession(sess, r.c.GetLiveRTSession(&sess.Metadata.RV))
	return nil
}

func (r *rtdtServer) CreateInstantSession(_ context.Context, req *types.CreateInstantRTDTSessionRequest, res *types.CreateInstantRTDTSessionResponse) error {
	users, err := r.resolveUsers(req.Users)
	if err != nil {
		return err
	}

	sess, err := r.c.CreateInstantRTDTSession(users)
	if err != nil {
		return err
	}

	res.Session = marshalRTDTSession(sess, r.c.GetLiveRTSession(&sess.Metadata.RV))
	return nil
}

func (r *rtdtServer) InviteToSession(_ context.Context, req *types.InviteToRTDTSessionRequest, _ *types.InviteToRTDTSessionResponse) error {
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	users, err := r.resolveUsers(req.Users)
	if err != nil {
		return err
	}

	return r.c.InviteToRTDTSession(rv, req.AllowedAsPublisher, users...)
}

func (r *rtdtServer) AcceptInvite(_ context.Context, req *types.AcceptRTDTSessionInviteRequest, _ *types.AcceptRTDTSessionInviteResponse) error {
	inviter, err := r.c.UIDByNick(req.Inviter)
	if err != nil {
		return err
	}
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	return r.c.AcceptRTDTSessionInviteByRV(inviter, rv, req.AcceptAsPublisher)
}

func (r *rtdtServer) CancelInvite(_ context.Context, req *types.CancelRTDTSessionInviteRequest, _ *types.CancelRTDTSessionInviteResponse) error {
	inviter, err := r.c.UIDByNick(req.Inviter)
	if err != nil {
		return err
	}
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	return r.c.CancelRTDTSessionInviteByRV(inviter, rv)
}

func (r *rtdtServer) JoinSession(_ context.Context, req *types.JoinLiveRTDTSessionRequest, _ *types.JoinLiveRTDTSessionResponse) error {
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	return r.c.JoinLiveRTDTSession(rv)
}

func (r *rtdtServer) LeaveSession(_ context.Context, req *types.LeaveLiveRTDTSessionRequest, _ *types.LeaveLiveRTDTSessionResponse) error {
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	return r.c.LeaveLiveRTSession(rv)
}

func (r *rtdtServer) ExitSession(_ context.Context, req *types.ExitRTDTSessionRequest, _ *types.ExitRTDTSessionResponse) error {
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	return r.c.ExitRTDTSession(&rv)
}

func (r *rtdtServer) DissolveSession(_ context.Context, req *types.DissolveRTDTSessionRequest, _ *types.DissolveRTDTSessionResponse) error {
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	return r.c.DissolveRTDTSession(&rv)
}

func (r *rtdtServer) SendChatMessage(_ context.Context, req *types.SendRTDTChatMessageRequest, _ *types.SendRTDTChatMessageResponse) error {
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	return r.c.SendRTDTChatMsg(rv, req.Message)
}

func (r *rtdtServer) SendAudioPacket(ctx context.Context, req *types.SendRTDTAudioPacketRequest, _ *types.SendRTDTAudioPacketResponse) error {
	rv, err := parseShortIDHex(req.SessionRv)
	if err != nil {
		return err
	}
	return r.c.SendRTDTAudioPacket(ctx, &rv, req.OpusPacket, req.Timestamp)
}

func (r *rtdtServer) InvitesStream(ctx context.Context, req *types.RTDTInvitesStreamRequest, stream types.RTDTService_InvitesStreamServer) error {
	return r.inviteStreams.runStream(ctx, req.UnackedFrom, stream)
}

func (r *rtdtServer) AckReceivedInvites(_ context.Context, req *types.AckRequest, _ *types.AckResponse) error {
	return r.inviteStreams.ack(req.SequenceId)
}

func (r *rtdtServer) SessionUpdatesStream(ctx context.Context, req *types.RTDTSessionUpdatesStreamRequest, stream types.RTDTService_SessionUpdatesStreamServer) error {
	return r.updateStreams.runStream(ctx, req.UnackedFrom, stream)
}

func (r *rtdtServer) AckSessionUpdates(_ context.Context, req *types.AckRequest, _ *types.AckResponse) error {
	return r.updateStreams.ack(req.SequenceId)
}

func (r *rtdtServer) ChatMessagesStream(ctx context.Context, req *types.RTDTChatMessagesStreamRequest, stream types.RTDTService_ChatMessagesStreamServer) error {
	return r.messageStreams.runStream(ctx, req.UnackedFrom, stream)
}

func (r *rtdtServer) AckChatMessages(_ context.Context, req *types.AckRequest, _ *types.AckResponse) error {
	return r.messageStreams.ack(req.SequenceId)
}

func (r *rtdtServer) AudioPacketsStream(ctx context.Context, req *types.RTDTAudioPacketsStreamRequest, stream types.RTDTService_AudioPacketsStreamServer) error {
	return r.audioStreams.runStream(ctx, filteredRTDTAudioPacketStream{
		sessionRV: req.SessionRv,
		stream:    stream,
	})
}

func (r *rtdtServer) invitedToRTDTSession(ru *client.RemoteUser, invite *rpc.RMRTDTSessionInvite, ts time.Time) {
	r.inviteStreams.send(&types.ReceivedRTDTInvite{
		InviterUid:  ru.ID().String(),
		InviterNick: ru.Nick(),
		TimestampMs: ts.UnixMilli(),
		Invite:      marshalRTDTInvite(invite),
	})
}

func (r *rtdtServer) rtdtSessionUpdated(ru *client.RemoteUser, update *client.RTDTSessionUpdateNtfn) {
	sess, err := r.getSession(&update.SessionRV)
	if err != nil {
		r.log.Warnf("Unable to load updated RTDT session %s: %v", update.SessionRV, err)
		return
	}

	r.updateStreams.send(&types.RTDTSessionUpdatedEvent{
		UpdaterUid:  ru.ID().String(),
		UpdaterNick: ru.Nick(),
		SessionRv:   update.SessionRV.String(),
		Session:     sess,
		InitialJoin: update.InitialJoin,
	})
}

func (r *rtdtServer) rtdtChatMessageReceived(sessionRV zkidentity.ShortID, pub rpc.RMRTDTSessionPublisher, msg string, ts uint32) {
	r.messageStreams.send(&types.RTDTChatMessageReceivedEvent{
		SessionRv: sessionRV.String(),
		Publisher: marshalRTDTSessionPublisher(pub),
		Message:   msg,
		Timestamp: ts,
	})
}

func (r *rtdtServer) rtdtAudioPacketReceived(sessionRV zkidentity.ShortID, peerID rpc.RTDTPeerID, opusPacket []byte, ts uint32) {
	r.audioStreams.send(&types.RTDTAudioPacketReceivedEvent{
		SessionRv:  sessionRV.String(),
		PeerId:     uint32(peerID),
		Timestamp:  ts,
		OpusPacket: opusPacket,
	})
}

func (r *rtdtServer) registerOfflineMessageStorageHandlers() {
	nmgr := r.c.NotificationManager()
	nmgr.RegisterSync(client.OnInvitedToRTDTSession(r.invitedToRTDTSession))
	nmgr.RegisterSync(client.OnRTDTSesssionUpdated(r.rtdtSessionUpdated))
	nmgr.RegisterSync(client.OnRTDTChatMessageReceived(r.rtdtChatMessageReceived))
	nmgr.RegisterSync(client.OnRTDTAudioPacketReceived(r.rtdtAudioPacketReceived))
}

var _ types.RTDTServiceServer = (*rtdtServer)(nil)

// InitRTDTService initializes and binds an RTDTService server to the RPC server.
func (s *Server) InitRTDTService(cfg RTDTServerCfg) error {
	inviteStreams, err := newServerStreams[*types.ReceivedRTDTInvite](cfg.RootReplayMsgLogs, "rtdtinvites", cfg.Log)
	if err != nil {
		return err
	}

	updateStreams, err := newServerStreams[*types.RTDTSessionUpdatedEvent](cfg.RootReplayMsgLogs, "rtdtupdates", cfg.Log)
	if err != nil {
		return err
	}

	messageStreams, err := newServerStreams[*types.RTDTChatMessageReceivedEvent](cfg.RootReplayMsgLogs, "rtdtchatmsgs", cfg.Log)
	if err != nil {
		return err
	}

	rs := &rtdtServer{
		cfg: cfg,
		log: cfg.Log,
		c:   cfg.Client,

		inviteStreams:  inviteStreams,
		updateStreams:  updateStreams,
		messageStreams: messageStreams,
		audioStreams:   newLiveServerStreams[*types.RTDTAudioPacketReceivedEvent]("rtdtaudio", cfg.Log),
	}
	rs.registerOfflineMessageStorageHandlers()
	s.services.Bind("RTDTService", types.RTDTServiceDefn(), rs)
	return nil
}
