package client

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/companyzero/bisonrelay/client/clientdb"
	"github.com/companyzero/bisonrelay/client/clientintf"
	"github.com/companyzero/bisonrelay/rpc"
	"github.com/companyzero/bisonrelay/zkidentity"
	"golang.org/x/exp/slices"
)

const (
	// newPTVersion is the version of newly created PokerTables.
	newPTVersion          = 0
	minSupportedPTVersion = 0
	maxSupportedPTVersion = 0
)

// NewGroupChatVersion creates a new gc with the local user as admin and the
// specified version.
func (c *Client) NewPokerTableVersion(name string, version uint8) (zkidentity.ShortID, error) {
	var id zkidentity.ShortID

	// Ensure we're not trying to duplicate the name.
	if _, err := c.PTIDByName(name); err == nil {
		return id, fmt.Errorf("pt named %q already exists", name)
	}

	if _, err := rand.Read(id[:]); err != nil {
		return id, err
	}
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		// Ensure it doesn't exist.
		_, err := c.db.GetPT(tx, id)
		if !errors.Is(err, clientdb.ErrNotFound) {
			if err == nil {
				err = fmt.Errorf("can't create pt %q (%s): %w",
					name, id.String(), errAlreadyExists)
			}
			return err
		}
		pt := rpc.RMPokerTableList{
			ID:         id,
			Name:       name,
			Generation: 1,
			Version:    version,
			Timestamp:  time.Now().Unix(),
			Members: []zkidentity.ShortID{
				c.PublicID(),
			},
		}
		if err = c.db.SavePT(tx, pt); err != nil {
			return fmt.Errorf("can't save pt %q (%s): %v", name, id.String(), err)
		}
		if aliasMap, err := c.db.SetPTAlias(tx, id, name); err != nil {
			c.log.Errorf("can't set name %s for pt %s: %v", name, id.String(), err)
		} else {
			c.setGCAlias(aliasMap)
		}
		c.log.Infof("Created new pt %q (%s)", name, id.String())

		return nil
	})
	return id, err
}

// NewGroupChat creates a group chat with the local client as admin.
func (c *Client) NewPokerTable(name string) (zkidentity.ShortID, error) {
	return c.NewPokerTableVersion(name, newPTVersion)
}

// PTIDByName returns the PT ID of the local PT with the given name. The name can
// be either a local PT alias or a full hex PT ID.
func (c *Client) PTIDByName(name string) (zkidentity.ShortID, error) {
	var id zkidentity.ShortID

	// Check if it's a full hex ID.
	if err := id.FromString(name); err == nil {
		return id, nil
	}

	// Check alias cache.
	c.ptAliasMtx.Lock()
	id, ok := c.ptAliasMap[name]
	c.ptAliasMtx.Unlock()

	if !ok {
		return id, fmt.Errorf("pt %q not found", name)
	}

	return id, nil
}

// setPTAlias sets the new group chat alias cache.
func (c *Client) setPTAlias(aliasMap map[string]zkidentity.ShortID) {
	c.gcAliasMtx.Lock()
	c.ptAliasMap = aliasMap
	c.gcAliasMtx.Unlock()
}

// uidHasGCPerm returns true whether the given UID has permission to modify the
// given GC. This takes into account the GC version.
func (c *Client) uidHasPTPerm(gc rpc.RMPokerTableList, uid clientintf.UserID) error {
	if gc.Version == 0 {
		// Version 0 GCs only have admin as Members[0].
		if len(gc.Members) > 0 && gc.Members[0].ConstantTimeEq(&uid) {
			return nil
		}

		return fmt.Errorf("user %s not version 0 GC admin", uid)
	}

	if gc.Version == 1 {
		if len(gc.Members) > 0 && gc.Members[0].ConstantTimeEq(&uid) {
			// Update from admin. Accept.
			return nil
		}

		if slices.Contains(gc.ExtraAdmins, uid) {
			// Additional admin.
			return nil
		}

		return fmt.Errorf("user %s not version 1 GC admin", uid)
	}

	return fmt.Errorf("unsupported GC version %d", gc.Version)
}

// InviteToPokerTable invites the given user to the given poker table. The local user
// must be the admin of the table and the remote user must have been KX'd with.
func (c *Client) InviteToPokerTable(ptID zkidentity.ShortID, user UserID) error {
	ru, err := c.rul.byID(user)
	if err != nil {
		return err
	}

	invite := rpc.RMPokerTableInvite{
		ID:      ptID,
		Expires: time.Now().Add(time.Hour * 24).Unix(),
	}

	err = c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		// Ensure poker table exists and we're the admin.
		pt, err := c.db.GetPT(tx, ptID)
		if err != nil {
			return err
		}

		if err := c.uidHasPTPerm(pt, c.PublicID()); err != nil {
			return fmt.Errorf("not permitted to send send invite: %v", err)
		}

		invite.Name = pt.Name
		invite.Version = pt.Version

		// Generate an unused token.
		for {
			// The % 1000000 is to generate a shorter token and
			// maintain compat to old client.
			invite.Token = c.mustRandomUint64() % 1000000
			_, _, _, err := c.db.FindPTInvite(tx, ptID, invite.Token)
			if errors.Is(err, clientdb.ErrNotFound) {
				break
			} else if err != nil {
				return err
			}
		}

		// Add to db.
		_, err = c.db.AddPTInvite(tx, user, invite)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Send the invite.
	c.log.Infof("Inviting %s to poker table %q (%s)", ru, invite.Name, ptID)
	payEvent := fmt.Sprintf("pt.%s.sendinvite", ptID.ShortLogID())
	return ru.sendRM(invite, payEvent)
}

// handlePTInvite handles a message where a remote user is inviting us to join
// a poker table.
func (c *Client) handlePTInvite(ru *RemoteUser, invite rpc.RMPokerTableInvite) error {
	if invite.ID.IsEmpty() {
		return fmt.Errorf("cannot accept poker table invite: pt id is empty")
	}

	invite.Name = strings.TrimSpace(invite.Name)
	if invite.Name == "" {
		invite.Name = hex.EncodeToString(invite.ID[:8])
	}

	if invite.Version < minSupportedPTVersion || invite.Version > maxSupportedPTVersion {
		return fmt.Errorf("invited to poker table %s (%q) with unsupported version %d",
			invite.ID, invite.Name, invite.Version)
	}

	if invite.ID.IsEmpty() {
		return fmt.Errorf("poker table id is empty")
	}

	// Add this invite to the DB.
	var iid uint64
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var err error
		_, err = c.db.GetPT(tx, invite.ID)
		if !errors.Is(err, clientdb.ErrNotFound) {
			if err == nil {
				err = fmt.Errorf("can't accept poker table invite: pt %q already exists",
					invite.ID.String())
			}
			return err
		}

		iid, err = c.db.AddPTInvite(tx, ru.ID(), invite)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Let user know about it.
	c.log.Infof("Received invitation to poker table %q from user %s", invite.ID.String(), ru)
	c.ntfns.notifyInvitedToPT(ru, iid, invite)
	return nil
}

// handleGCJoin handles a msg when a remote user is asking to join a GC we
// administer (that is, responding to an invite previously sent by us).
func (c *Client) handlePTJoin(ru *RemoteUser, invite rpc.RMPokerTableJoin) error {
	var gc rpc.RMPokerTableList
	updated := false
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var err error
		_, uid, iid, err := c.db.FindPTInvite(tx,
			invite.ID, invite.Token)
		if err != nil {
			return err
		}

		// Ensure we received this join from the same user we sent it
		// to.
		if uid != ru.ID() {
			return fmt.Errorf("received PTJoin from user %s when "+
				"it was sent to user %s", ru.ID(), uid)
		}

		// Ensure we have permission to add people to GC.
		gc, err = c.db.GetPT(tx, invite.ID)
		if err != nil {
			return err
		}
		if err := c.uidHasPTPerm(gc, c.PublicID()); err != nil {
			return fmt.Errorf("local user does not have permission "+
				"to add gc member: %v", err)
		}

		// This invitation is fulfilled.
		if err = c.db.DelPTInvite(tx, iid); err != nil {
			return err
		}

		// Ensure user is not on gc yet.
		if slices.Contains(gc.Members, uid) {
			return fmt.Errorf("user %s already part of gc %q",
				uid, gc.ID.String())
		}

		if invite.Error == "" {
			// Add the new member, increment generation, save the
			// new gc group.
			gc.Members = append(gc.Members, uid)
			gc.Generation += 1
			gc.Timestamp = time.Now().Unix()
			if err = c.db.SavePT(tx, gc); err != nil {
				return err
			}
			updated = true
		} else {
			c.log.Infof("User %s rejected invitation to %q: %q",
				ru, gc.ID.String(), invite.Error)
		}
		return nil
	})

	if err != nil || !updated {
		return err
	}

	c.log.Infof("User %s joined gc %s (%q)", ru, gc.ID, gc.Name)

	// Join fulfilled. Send new group list to every member except admin
	// (us).
	err = c.sendToPTMembers(gc.ID, gc.Members, "sendlist", gc, nil)
	if err != nil {
		return err
	}

	c.ntfns.notifyPTInviteAccepted(ru, gc)
	return nil
}

// AcceptPokerTableInvite accepts the given invitation, previously received from
// some user.
func (c *Client) AcceptPokerTableInvite(iid uint64) error {
	var invite rpc.RMPokerTableInvite
	var uid UserID
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var err error
		invite, uid, err = c.db.GetPTInvite(tx, iid)
		if err != nil {
			return err
		}

		if err := c.db.MarkPTInviteAccepted(tx, iid); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return err
	}

	ru, err := c.rul.byID(uid)
	if err != nil {
		return err
	}

	join := rpc.RMPokerTableJoin{
		ID:    invite.ID,
		Token: invite.Token,
	}
	c.log.Infof("Accepting invitation to poker table %q (%s) from %s", invite.Name, invite.ID.String(), ru)
	payEvent := fmt.Sprintf("pt.%s.acceptinvite", invite.ID.ShortLogID())
	return ru.sendRM(join, payEvent)
}

// ListPTInvitesFor returns all poker table invites. If ptid is specified, only invites
// for the specified poker table are returned.
func (c *Client) ListPTInvitesFor(ptid *zkidentity.ShortID) ([]*clientdb.PTInvite, error) {
	var invites []*clientdb.PTInvite
	err := c.dbView(func(tx clientdb.ReadTx) error {
		var err error
		invites, err = c.db.ListPTInvites(tx, ptid)
		return err
	})
	return invites, err
}

// GetGC returns information about the given gc the local user participates in.
func (c *Client) GetPT(ptID zkidentity.ShortID) (rpc.RMPokerTableList, error) {
	var pt rpc.RMPokerTableList
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var err error
		pt, err = c.db.GetPT(tx, ptID)
		return err
	})
	return pt, err
}

// GetGCAlias returns the local alias for the specified GC.
func (c *Client) GetPTAlias(ptID zkidentity.ShortID) (string, error) {
	var alias string
	c.ptAliasMtx.Lock()
	for v, id := range c.ptAliasMap {
		if id == ptID {
			alias = v
			break
		}
	}
	c.ptAliasMtx.Unlock()
	if alias == "" {
		return "", fmt.Errorf("gc %s not found", ptID)
	}
	return alias, nil
}

// sendToGCMembers sends the given message to all GC members of the given slice
// (unless that is the local client).
func (c *Client) sendToPTMembers(gcID zkidentity.ShortID,
	members []zkidentity.ShortID, payType string, msg interface{},
	progressChan chan SendProgress) error {

	localID := c.PublicID()
	payEvent := fmt.Sprintf("pt.%s.%s", gcID.ShortLogID(), payType)

	// missingKXInfo will be used to track information about members that
	// the local client hasn't KXd with yet.
	type missingKXInfo struct {
		uid      clientintf.UserID
		hasKX    bool
		hasMI    bool
		medID    *clientintf.UserID
		miCount  uint32
		skipWarn bool
		skipMI   bool
	}
	var missingKX []missingKXInfo

	// Add the set of outbound messages to the sendq.
	ids := make([]clientintf.UserID, 0, len(members)-1)
	for _, uid := range members {
		if uid == localID {
			continue
		}

		// If the user isn't KX'd with, and they haven't been warned
		// recently, add to the missingKX list to perform some actions
		// later on.
		_, err := c.rul.byID(uid)
		if err != nil {
			c.unkxdWarningsMtx.Lock()
			if t := c.unkxdWarnings[uid]; time.Since(t) > c.cfg.UnkxdWarningTimeout {
				missingKX = append(missingKX, missingKXInfo{uid: uid})
			}
			c.unkxdWarningsMtx.Unlock()
			continue
		}

		ids = append(ids, uid)
	}
	sqid, err := c.addToSendQ(payEvent, msg, priorityGC, ids...)
	if err != nil {
		return fmt.Errorf("Unable to add pt msg to send queue: %v", err)
	}

	// These will be used to track the sending progress.
	var progressMtx sync.Mutex
	var sent, total int

	// Start the sending process for each member.
	for _, id := range members {
		if id == localID {
			continue
		}

		ru, err := c.rul.byID(id)
		if err != nil {
			continue
		}

		progressMtx.Lock() // Unlikely, but could race with the result.
		total += 1
		progressMtx.Unlock()

		// Send as a goroutine to fulfill for all users concurrently.
		go func() {
			err := ru.sendRMPriority(msg, payEvent, priorityGC)
			if errors.Is(err, clientintf.ErrSubsysExiting) {
				return
			}

			// Remove from sendq independently of error.
			c.removeFromSendQ(sqid, ru.ID())
			if err != nil {
				c.log.Errorf("Unable to send %T on gc %q to user %s: %v",
					msg, gcID.String(), ru, err)
				return
			}

			// Alert about progress.
			if progressChan != nil {
				progressMtx.Lock()
				sent += 1
				progressChan <- SendProgress{
					Sent:  sent,
					Total: total,
					Err:   err,
				}
				progressMtx.Unlock()
			}
		}()
	}

	// Early return if there are no members that are missing kx.
	if len(missingKX) == 0 {
		return nil
	}

	// Handle GC members for which we don't have KX. Determine if there
	// is a KX/MediateID attempt for them, start a new one if needed and
	// warn the UI about it.
	//
	// First: go through the DB to see if they are being KX'd with.
	err = c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var localID, gcOwner clientintf.UserID
		var gotGCInfo bool
		for i := range missingKX {
			target := missingKX[i].uid

			// Check if already KXing.
			kxs, err := c.db.HasKXWithUser(tx, target)
			if err != nil {
				return err
			}
			missingKX[i].hasKX = len(kxs) > 0

			// Check if already has MediateID requests.
			hasRecent, err := c.db.HasAnyRecentMediateID(tx, target,
				c.cfg.RecentMediateIDThreshold)
			if err != nil {
				return err
			}
			missingKX[i].hasMI = hasRecent

			// Check if the attempts to KX with the target crossed
			// a max attempt count limit or if we're expecting a
			// KX request from them (because _they_ joined the
			// GC).
			if unkx, err := c.db.ReadUnxkdUserInfo(tx, target); err == nil {
				missingKX[i].miCount = unkx.MIRequests
				if unkx.MIRequests >= uint32(c.cfg.MaxAutoKXMediateIDRequests) {
					c.log.Debugf("Skipping autoKX with GC's %s member %s "+
						"due to MI requests %d >= max %d",
						gcID, target, unkx.MIRequests,
						c.cfg.MaxAutoKXMediateIDRequests)
					missingKX[i].skipMI = true
					missingKX[i].skipWarn = unkx.MIRequests > uint32(c.cfg.MaxAutoKXMediateIDRequests)
					if !missingKX[i].skipWarn {
						// Add one to MIRequests to avoid warning again.
						unkx.MIRequests += 1
						err := c.db.StoreUnkxdUserInfo(tx, unkx)
						if err != nil {
							return err
						}
					}
				} else if unkx.AddedToGCTime != nil && time.Since(*unkx.AddedToGCTime) < c.cfg.RecentMediateIDThreshold {
					c.log.Debugf("Skipping autoKX with GC's %s member %s "+
						"due to interval from GC add %s < recent "+
						"MI threshold %s", gcID, target,
						time.Since(*unkx.AddedToGCTime),
						c.cfg.RecentMediateIDThreshold)
					missingKX[i].skipMI = true
				}
				if unkx.AddedToGCTime != nil && time.Since(*unkx.AddedToGCTime) < c.cfg.UnkxdWarningTimeout {
					missingKX[i].skipWarn = true
				}
			}

			if missingKX[i].skipMI || missingKX[i].hasMI {
				continue
			}

			// Fetch the group list if needed.
			if !gotGCInfo {
				gcl, err := c.db.GetPT(tx, gcID)
				if err != nil {
					return err
				}
				localID = c.PublicID()
				gcOwner = gcl.Members[0]
				gotGCInfo = true
			}

			// Determine if we can ask the GC's owner for a mediate
			// ID request.
			if gcOwner != localID {
				missingKX[i].medID = &gcOwner
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Next, log a warning and send a ntfn to the UI about each user's
	// situation.
	c.unkxdWarningsMtx.Lock()
	now := time.Now()
	for _, mkx := range missingKX {
		if mkx.skipWarn {
			continue
		}
		if t := c.unkxdWarnings[mkx.uid]; now.Sub(t) < c.cfg.UnkxdWarningTimeout {
			// Already warned.
			continue
		}
		c.unkxdWarnings[mkx.uid] = now
		c.log.Warnf("Unable to send %T to unKXd member %s in GC %s",
			msg, mkx.uid, gcID)
		c.ntfns.notifyGCWithUnxkdMember(gcID, mkx.uid, mkx.hasKX, mkx.hasMI,
			mkx.miCount, mkx.medID)

	}
	c.unkxdWarningsMtx.Unlock()

	// Next, start the mediate id requests that are needed.
	for _, mkx := range missingKX {
		if mkx.hasKX || mkx.hasMI || mkx.medID == nil || mkx.skipMI {
			continue
		}

		err := c.maybeRequestMediateID(*mkx.medID, mkx.uid)
		if err != nil && !errors.Is(err, clientintf.ErrSubsysExiting) {
			c.log.Errorf("Unable to request mediate ID of target %s "+
				"to mediator %s: %v", mkx.uid, mkx.medID, err)
		}
	}

	return nil
}

// PTAct sends an action to the given PT. If progressChan is not nil,
// events are sent to it as the sending process progresses. Writes to
// progressChan are serial, so it's important that it not block indefinitely.
func (c *Client) PTAct(gcID zkidentity.ShortID, msg string, mode rpc.MessageMode,
	progressChan chan SendProgress) error {

	var gc rpc.RMPokerTableList
	var gcBlockList clientdb.GCBlockList
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var err error
		if gc, err = c.db.GetPT(tx, gcID); err != nil {
			return err
		}
		if gcBlockList, err = c.db.GetGCBlockList(tx, gcID); err != nil {
			return err
		}

		gcAlias, err := c.GetPTAlias(gcID)
		if err != nil {
			gcAlias = gc.Name
		}

		return c.db.LogPTAct(tx, gcAlias, gcID, false, c.id.Public.Nick, msg, time.Now())
	})
	if err != nil {
		return err
	}

	p := rpc.RMPokerTableAction{
		ID:         gcID,
		Generation: gc.Generation,
		Action:     msg,
		Mode:       mode,
	}
	members := gcBlockList.FilterMembers(gc.Members)
	if len(members) == 0 {
		return nil
	}

	return c.sendToPTMembers(gcID, members, "act", p, progressChan)
}

// handleDelayedGCMessages is called by the gc message cacher when it's time
// to let external callers know about new messages.
func (c *Client) handleDelayedPTActions(msg clientintf.ReceivedPTAct) {
	user, err := c.UserByID(msg.UID)
	if err != nil {
		// Should only happen if we blocked the user
		// during the gcm cacher delay.
		c.log.Warnf("Delayed GC message with unknown user %s", msg.UID)
		return
	}

	// Log the message and remove the cached GCM from the db.
	err = c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		// if err := c.db.RemoveCachedRGCM(tx, msg); err != nil {
		// 	c.log.Warnf("Unable to remove cached RGCM: %v", err)
		// }

		gcAlias, _ := c.GetPTAlias(msg.PTA.ID)
		err := c.db.LogGCMsg(tx, gcAlias, msg.PTA.ID, false, user.Nick(),
			msg.PTA.Action, msg.TS)
		if err != nil {
			c.log.Warnf("Unable to log RGCM: %v", err)
		}

		return nil
	})
	if err != nil {
		// Not a fatal error, so just log a warning.
		c.log.Warnf("Unable to handle cached RGCM: %v", err)
	}

	c.ntfns.notifyOnPTA(user, msg.PTA, msg.TS)
}

// ListGCs lists all local GCs the user is participating in.
func (c *Client) ListPTs() ([]rpc.RMPokerTableList, error) {
	var gcs []rpc.RMPokerTableList
	err := c.dbView(func(tx clientdb.ReadTx) error {
		var err error
		gcs, err = c.db.ListPTs(tx)
		return err
	})
	return gcs, err
}

func (c *Client) handlePTMessage(ru *RemoteUser, gcm rpc.RMPokerTableAction, ts time.Time) error {
	var gc rpc.RMPokerTableList
	var found, isBlocked bool
	var gcAlias string

	// Create the local cached structure for a received GCM. The MsgID is
	// just a random id used for caching purposes.
	rgcm := clientintf.ReceivedPTAct{
		UID: ru.ID(),
		PTA: rpc.RMPokerTableAction{
			ID:            gcm.ID,
			Generation:    gcm.Generation,
			Action:        gcm.Action,
			Mode:          gcm.Mode,
			CurrentPlayer: gcm.CurrentPlayer,
		},
		TS: ts,
	}
	_, _ = rand.Read(rgcm.MsgID[:])

	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		// Ensure gc exists.
		var err error
		gc, err = c.db.GetPT(tx, gcm.ID)
		if err != nil {
			return err
		}
		for i := range gc.Members {
			if ru.ID() == gc.Members[i] {
				found = true
				break
			}
		}
		if !found {
			return nil
		}

		// gcBlockList, err := c.db.GetGCBlockList(tx, gcm.ID)
		// if err != nil {
		// 	return err
		// }
		// isBlocked = gcBlockList.IsBlocked(ru.ID())
		// if isBlocked {
		// 	return nil
		// }

		return c.db.CacheReceivedPTA(tx, rgcm)
	})
	// if errors.Is(err, clientdb.ErrNotFound) {
	// 	// Remote user sent message on group chat we're no longer a
	// 	// member of. Alert them not to resend messages in this GC to
	// 	// us.
	// 	ru.log.Warnf("Received message on unknown groupchat %q", gcm.ID)
	// 	rmgp := rpc.RMGroupPart{
	// 		ID:     gcm.ID,
	// 		Reason: "I am not in that groupchat",
	// 	}
	// 	payEvent := fmt.Sprintf("gc.%s.preventiveGroupPart", gcm.ID.ShortLogID())
	// 	return ru.sendRMPriority(rmgp, payEvent, priorityGC)
	// }
	if err != nil {
		return err
	}

	if isBlocked {
		c.log.Warnf("Received message in GC %q from blocked member %s",
			gcAlias, ru)
		return nil
	}

	if !found {
		// The sender is not in the GC list we have.
		c.log.Warnf("Received message in GC %q from non-member %s",
			gcAlias, ru)
		return nil
	}

	// if filter, _ := c.FilterGCM(ru.ID(), gc.ID, gcm.Message); filter {
	// 	return nil
	// }

	ru.log.Debugf("Received message of len %d in GC %q (%s)", len(gcm.Action),
		gcAlias, gc.ID)

	c.gcmq.PTActionReceived(rgcm)
	return nil
}

// handleGCList handles updates to a GC metadata. The sending user must have
// been the admin, otherwise this update is rejected.
func (c *Client) handlePTList(ru *RemoteUser, gl rpc.RMPokerTableList) error {
	var gcName string

	// Check if GC exists to determine if it's the first GC list.
	_, err := c.GetPT(gl.ID)
	isNewGC := err != nil && errors.Is(err, clientdb.ErrNotFound)
	if err != nil && !isNewGC {
		return err
	}

	if !isNewGC {
		// Existing GC update. Do the update, then return.
		oldGC, err := c.maybeUpdatePT(ru, gl)
		if err != nil {
			return err
		}

		gcName, _ = c.GetPTAlias(gl.ID)
		c.log.Infof("Received updated GC list %s (%q) from %s", gl.ID, gcName, ru)
		c.notifyUpdatedPT(ru, oldGC, gl)
		return nil
	}

	// First GC list from a GC we just joined.
	gcName, err = c.saveJoinedPT(ru, gl)
	if err != nil {
		return err
	}
	c.log.Infof("Received first GC list of %s (%q) from %s", gl.ID, gcName, ru)
	c.ntfns.notifyOnJoinedPT(gl)

	// Start kx with unknown members. They are relying on us performing
	// transitive KX via an admin.
	me := c.PublicID()
	for _, v := range gl.Members {
		v := v
		if v == me {
			continue
		}

		_, err := c.rul.byID(v)
		if !errors.Is(err, userNotFoundError{}) {
			continue
		}

		err = c.maybeRequestMediateID(ru.ID(), v)
		if err != nil && !errors.Is(err, clientintf.ErrSubsysExiting) {
			c.log.Errorf("Unable to autokx with %s via %s: %v",
				v, ru, err)
		}
	}

	return nil
}

// saveJoinedGC is called when the local client receives the first RMGroupList
// after requesting to join the GC with the GC admin.
//
// Returns the new GC name.
func (c *Client) saveJoinedPT(ru *RemoteUser, gl rpc.RMPokerTableList) (string, error) {
	// var checkVersionWarning bool
	var gcName string
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		// Double check GC does not exist yet.
		_, err := c.db.GetPT(tx, gl.ID)
		if err == nil {
			return fmt.Errorf("GC %s already exists when attempting "+
				"to save after joining", gl.ID)
		}

		// This must have been an invite we accepted. Ensure
		// this came from the expected user.
		invite, _, err := c.db.FindAcceptedGCInvite(tx, gl.ID, ru.ID())
		if err != nil {
			return fmt.Errorf("unable to list gc invites: %v", err)
		}

		// This is set to true before the perm check because future
		// versions might change the permissions about who can send the
		// list.
		// checkVersionWarning = true

		// Ensure we received this from someone that can add
		// members.
		if err := c.uidHasPTPerm(gl, ru.ID()); err != nil {
			return err
		}

		// Remove all invites received to this GC.
		if err := c.db.DelAllInvitesToGC(tx, gl.ID); err != nil {
			return fmt.Errorf("unable to del gc invite: %v", err)
		}

		// Figure out the GC name.
		gcName = invite.Name
		_, err = c.PTIDByName(gcName)
		for i := 1; err == nil; i += 1 {
			gcName = fmt.Sprintf("%s_%d", invite.Name, i)
			_, err = c.PTIDByName(gcName)
		}

		if aliasMap, err := c.db.SetPTAlias(tx, gl.ID, gcName); err != nil {
			c.log.Errorf("can't set name %s for gc %s: %v", gcName, gl.ID.String(), err)
		} else {
			c.setGCAlias(aliasMap)
		}

		// All is well. Update the local gc data.
		if err := c.db.SavePT(tx, gl); err != nil {
			return fmt.Errorf("unable to save gc: %v", err)
		}
		return nil
	})
	// if checkVersionWarning {
	// 	c.maybeNotifyPTVersionWarning(ru, gl.ID, gl)
	// }
	return gcName, err
}

// notifyUpdatedGC determines what changed between two GC definitions and
// notifies the user about it.
func (c *Client) notifyUpdatedPT(ru *RemoteUser, oldGC, newGC rpc.RMPokerTableList) {
	// if oldGC.Version != newGC.Version {
	// 	c.ntfns.notifyOnGCUpgraded(newGC, oldGC.Version)
	// }

	// memberChanges := sliceDiff(oldGC.Members, newGC.Members)
	// if len(memberChanges.added) > 0 {
	// 	c.ntfns.notifyOnAddedGCMembers(newGC, memberChanges.added)
	// }
	// if len(memberChanges.removed) > 0 {
	// 	c.ntfns.notifyOnRemovedGCMembers(newGC, memberChanges.removed)
	// }

	// adminChanges := sliceDiff(oldGC.ExtraAdmins, newGC.ExtraAdmins)

	// // Also check if the "owner" (Members[0] admin) changed.
	// if oldGC.Members[0] != newGC.Members[0] {
	// 	adminChanges.added = append(memberChanges.added, newGC.Members[0])
	// 	adminChanges.removed = append(memberChanges.removed, oldGC.Members[0])
	// }

	// if len(adminChanges.removed) > 0 || len(adminChanges.added) > 0 {
	// 	c.ntfns.notifyGCAdminsChanged(ru, newGC, adminChanges.added, adminChanges.removed)
	// }
}

// maybeUpdateGCFunc verifies that the given gcid exists, calls f() with the
// existing GC definition, then updates the DB with the modified value. It
// returns both the old and new GC definitions.
//
// Checks are performed to ensure the new GC definitions are sane and allowed
// by the given remote user. If ru is nil, then the update is assumed to be
// made by the local client.
//
// f is called within a DB tx.
func (c *Client) maybeUpdatePTFunc(ru *RemoteUser, gcid zkidentity.ShortID, f func(*rpc.RMPokerTableList) error) (oldGC, newGC rpc.RMPokerTableList, err error) {
	// var checkVersionWarning bool
	var updaterID clientintf.UserID
	localID := c.PublicID()
	if ru != nil {
		updaterID = ru.ID()
	} else {
		updaterID = localID
	}

	err = c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		// Fetch GC.
		var err error
		oldGC, err = c.db.GetPT(tx, gcid)
		if err != nil {
			return err
		}

		if len(oldGC.Members) == 0 {
			return fmt.Errorf("old GC %s has zero members", gcid)
		}

		// Produce the new GC. Deep copy the old GC so f() can mutate
		// everything.
		newGC = oldGC
		newGC.Members = slices.Clone(oldGC.Members)
		newGC.ExtraAdmins = slices.Clone(oldGC.ExtraAdmins)
		if err := f(&newGC); err != nil {
			return err
		}

		// Ensure no backtrack on generation.
		if newGC.Generation < oldGC.Generation {
			return fmt.Errorf("cannot backtrack GC generation on "+
				"GC %s (%d < %d)", gcid, oldGC.Generation,
				newGC.Generation)
		}

		// Ensure no downgrade in version.
		if newGC.Version < oldGC.Version {
			return fmt.Errorf("cannot downgrade GC version on "+
				"GC %s (%d < %d)", gcid, oldGC.Generation,
				newGC.Generation)
		}

		// Special case changing the admin: only the admin itself
		// can do it.
		if oldGC.Members[0] != newGC.Members[0] && oldGC.Members[0] != updaterID {
			return fmt.Errorf("only previous GC admin %s may change "+
				"GC's %s admin", oldGC.Members[0], gcid)
		}

		// This check is done before checking for permission because a
		// future version might have different rules for checking
		// permission.
		// checkVersionWarning = ru != nil

		if err := c.uidHasPTPerm(oldGC, updaterID); err != nil {
			return err
		}

		// Handle case where the local client was removed from GC.
		stillMember := slices.Contains(newGC.Members, c.PublicID())
		if !stillMember {
			if err := c.db.DeleteGC(tx, oldGC.ID); err != nil {
				return err
			}
			if aliasMap, err := c.db.SetGCAlias(tx, oldGC.ID, ""); err != nil {
				return err
			} else {
				c.setGCAlias(aliasMap)
			}
			return nil
		}

		// This is an update, so any new members added to the GC that
		// we haven't KX'd with are expected to send a MI to the GC
		// owner/admin (because _they_ are the ones joining). So add
		// info to prevent us attempting a crossed MI with them for
		// some time.
		if ru != nil && stillMember {
			for _, uid := range newGC.Members {
				if uid == localID {
					continue
				}
				if c.db.AddressBookEntryExists(tx, uid) {
					continue
				}

				unkx, err := c.db.ReadUnxkdUserInfo(tx, uid)
				if err != nil {
					if !errors.Is(err, clientdb.ErrNotFound) {
						return err
					}
					unkx.UID = uid
				}
				if unkx.AddedToGCTime != nil {
					continue
				}
				now := time.Now()
				unkx.AddedToGCTime = &now
				if err := c.db.StoreUnkxdUserInfo(tx, unkx); err != nil {
					return err
				}
			}
		}

		return c.db.SavePT(tx, newGC)
	})

	// if checkVersionWarning {
	// 	c.maybeNotifyPTVersionWarning(ru, newGC.ID, newGC)
	// }

	return
}

// maybeUpdateGC updates the given GC definitions for the specified one.
func (c *Client) maybeUpdatePT(ru *RemoteUser, newGC rpc.RMPokerTableList) (oldGC rpc.RMPokerTableList, err error) {
	cb := func(ngc *rpc.RMPokerTableList) error {
		*ngc = newGC
		return nil
	}
	oldGC, _, err = c.maybeUpdatePTFunc(ru, newGC.ID, cb)
	return
}

// GCMessage sends a message to the given GC. If progressChan is not nil,
// events are sent to it as the sending process progresses. Writes to
// progressChan are serial, so it's important that it not block indefinitely.
func (c *Client) PTAction(gcID zkidentity.ShortID, msg string, mode rpc.MessageMode,
	progressChan chan SendProgress) error {

	var gc rpc.RMPokerTableList
	// var gcBlockList clientdb.GCBlockList
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var err error
		if gc, err = c.db.GetPT(tx, gcID); err != nil {
			return err
		}
		// if gcBlockList, err = c.db.GetGCBlockList(tx, gcID); err != nil {
		// 	return err
		// }

		gcAlias, err := c.GetPTAlias(gcID)
		if err != nil {
			gcAlias = gc.Name
		}

		return c.db.LogGCMsg(tx, gcAlias, gcID, false, c.id.Public.Nick, msg, time.Now())
	})
	if err != nil {
		return err
	}

	p := rpc.RMPokerTableAction{
		ID:         gcID,
		Generation: gc.Generation,
		Action:     msg,
		Mode:       mode,
	}
	// members := gcBlockList.FilterMembers(gc.Members)
	// if len(members) == 0 {
	// 	return nil
	// }

	return c.sendToPTMembers(gcID, gc.Members, "act", p, progressChan)
}
