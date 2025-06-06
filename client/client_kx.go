package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/companyzero/bisonrelay/client/clientdb"
	"github.com/companyzero/bisonrelay/client/clientintf"
	"github.com/companyzero/bisonrelay/internal/strescape"
	"github.com/companyzero/bisonrelay/ratchet"
	"github.com/companyzero/bisonrelay/rpc"
	"github.com/companyzero/bisonrelay/zkidentity"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

// Client KX flow is:
//
//          Alice                                    Bob
//         -------                                  -----
//
//     WriteNewInvite
//        \-----> OOBPublicIdentityInvite -->
//                  (out-of-band send)
//
//     kxlist.listenInvite()
//
//                                           ReadInvite()
//                                           AcceptInvite()
//                                           kxlist.acceptInvite()
//                               <-- RMOHalfKX ---/
//
//    kxlist.handleStep2IDKX()
//            \---- RMOFullKX -->
//    initRemoteUser()
//
//                                            kxlist.handleStep3IDKX()
//                                            initRemoteUser()
//

func (c *Client) takePostKXAction(ru *RemoteUser, act clientdb.PostKXAction) error {
	switch act.Type {
	case clientdb.PKXActionKXSearch:
		// Startup a KX search.
		var targetID UserID
		if err := targetID.FromString(act.Data); err != nil {
			return err
		}

		// See if we haven't found the target yet.
		if _, err := c.rul.byID(targetID); err == nil {
			// We have!
			return nil
		}

		// Not yet. Send the KX search query to them.
		var kxs clientdb.KXSearch
		if err := c.dbView(func(tx clientdb.ReadTx) error {
			var err error
			kxs, err = c.db.GetKXSearch(tx, targetID)
			return err
		}); err != nil {
			return err
		}

		return c.sendKXSearchQuery(kxs.Target, kxs.Search, ru.ID())

	case clientdb.PKXActionFetchPost:
		// Subscribe to posts, then fetch the specified post.
		var pid clientdb.PostID
		if err := pid.FromString(act.Data); err != nil {
			return err
		}

		return c.subscribeToPosts(ru.ID(), &pid, true)

	case clientdb.PKXActionInviteGC:
		// Invite user to GC
		gcname := act.Data
		gcID, err := c.GCIDByName(gcname)
		if err != nil {
			return err
		}
		if _, err := c.GetGC(gcID); err != nil {
			return err
		}

		return c.InviteToGroupChat(gcID, ru.ID())
	default:
		return fmt.Errorf("unknown post-kx action type")
	}
}

// takePostKXActions takes any post-kx actions needed after the user has been
// initialized.
func (c *Client) takePostKXActions(ru *RemoteUser, actions []clientdb.PostKXAction) {
	for _, act := range actions {
		act := act
		go func() {
			err := c.takePostKXAction(ru, act)
			if err != nil {
				ru.log.Errorf("Unable to take post-KX action %q: %v",
					act.Type, err)
			}
		}()
	}

	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		return c.db.RemovePostKXActions(tx, ru.ID())
	})
	if err != nil {
		ru.log.Warnf("Unable to move post-KX actions: %v", err)
	}
}

// initRemoteUser inserts the given ratchet as a new remote user. The bool
// returns whether this is a new user.
func (c *Client) initRemoteUser(id *zkidentity.PublicIdentity, r *ratchet.Ratchet,
	updateAB bool, initialRV, myResetRV, theirResetRV clientdb.RawRVID,
	ignored bool, nickAlias string) (*RemoteUser, bool, error) {

	var postKXActions []clientdb.PostKXAction

	// Track the new user.
	ru := newRemoteUser(c.q, c.rmgr, c.db, id, c.localID.signMessage, r)
	ru.ignored = ignored
	ru.compressLevel = c.cfg.CompressLevel
	ru.log = c.cfg.logger(fmt.Sprintf("RUSR %x", id.Identity[:8]))
	ru.logPayloads = c.cfg.logger(fmt.Sprintf("RMPL %x", id.Identity[:8]))
	ru.rmHandler = c.handleUserRM
	ru.myResetRV = myResetRV
	ru.theirResetRV = theirResetRV
	ru.ntfns = c.ntfns
	if nickAlias != "" {
		ru.setNick(nickAlias)
	}

	oldRU, err := c.rul.add(ru, c.LocalNick())
	oldUser := false
	if errors.Is(err, alreadyHaveUserError{}) && oldRU != nil {
		oldRU.log.Tracef("Reusing old remote user and replacing ratchet "+
			"(initial RV %s)", initialRV)

		// Preserve the earliest known nick as a NickAlias
		// (this prevents the same remote user from having
		// their nick modified) and confusing the local client
		// operator.
		if oldRU.Nick() != ru.Nick() {
			nickAlias = oldRU.Nick()
		}

		// Already have this user running. Replace the ratchet with the
		// new one.
		ru = oldRU
		go ru.replaceRatchet(r)
		oldUser = true
	} else if err != nil {
		return nil, false, err
	} else {
		ru.log.Debugf("Initializing remote user (initial RV %s)", initialRV)
	}

	// Save the newly formed address book entry to the DB.
	var oldEntry *clientdb.AddressBookEntry
	hadKXSearch := false
	err = c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var err error
		oldEntry, err = c.db.GetAddressBookEntry(tx, id.Identity)
		if err != nil && !errors.Is(err, clientdb.ErrNotFound) {
			return err
		}

		if err := c.db.UpdateRatchet(tx, r, id.Identity); err != nil {
			return err
		}
		var ignored bool
		firstCreated := time.Now()
		if oldEntry != nil {
			ignored = oldEntry.Ignored
			firstCreated = oldEntry.FirstCreated
			nickAlias = oldEntry.NickAlias
		}
		if updateAB {
			// Store the deduped nick as an alias if we had to
			// generate a deduped nick alias.
			if nickAlias == "" && ru.Nick() != id.Nick {
				nickAlias = ru.Nick()
				ru.log.Debugf("Preserving deduped nick %q", nickAlias)
			}

			newEntry := &clientdb.AddressBookEntry{
				ID:              id,
				MyResetRV:       myResetRV,
				TheirResetRV:    theirResetRV,
				Ignored:         ignored,
				FirstCreated:    firstCreated,
				NickAlias:       nickAlias,
				LastCompletedKX: time.Now(),

				// LastHandshakeAttempt is reset due to the
				// new KX.
				LastHandshakeAttempt: time.Time{},
			}
			if err := c.db.UpdateAddressBookEntry(tx, newEntry); err != nil {
				return err
			}

			// Log in the user chat that kx completed.
			if oldEntry == nil {
				c.db.LogPM(tx, id.Identity, true, "", "Completed KX", time.Now())
			} else {
				c.db.LogPM(tx, id.Identity, true, "", "Re-done KX", time.Now())
			}
		}

		// Convert initial actions to post actions.
		if !initialRV.IsEmpty() {
			err = c.db.InitialToPostKXActions(tx, initialRV, id.Identity)
			if err != nil {
				return err
			}
		}

		// See if there are any actions to be taken after completing KX.
		postKXActions, err = c.db.ListPostKXActions(tx, id.Identity)
		if err != nil {
			return err
		}
		// Remove KX search if it exists.
		if _, err := c.db.GetKXSearch(tx, id.Identity); err == nil {
			hadKXSearch = true
		}
		if err := c.db.RemoveKXSearch(tx, id.Identity); err != nil {
			return err
		}

		// Remove unkxd data if it exists.
		if err := c.db.RemoveUnkxUserInfo(tx, id.Identity); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, false, err
	}

	// Change the reset listening state on a goroutine so we don't block on
	// it.
	go func() {
		// When oldUser == true, the ratchet is replaced and that
		// triggers an automatic re-subscription, therefore there's no
		// need to do it again.
		if !oldUser {
			ru.updateRVs()
		}

		// Unsubscribe from the old reset RV point.
		if oldEntry != nil {
			c.kxl.unlistenReset(oldEntry.MyResetRV)
		}

		// Subscribe to the reset RV point.
		if err := c.kxl.listenReset(myResetRV, id); err != nil {
			ru.log.Warnf("unable to listen to reset: %v", err)
		}
	}()

	// Run the new user.
	if !oldUser {
		// Check if we should subscribe to posts. Ignore if there's a
		// post-kx action to subscribe, which will be prioritized.
		subToPosts := c.cfg.AutoSubscribeToPosts && updateAB
		for i := range postKXActions {
			if postKXActions[i].Type == clientdb.PKXActionFetchPost {
				subToPosts = false
				break
			}
		}
		if subToPosts {
			go c.SubscribeToPosts(ru.ID())
		}
	}

	// If there are any post-kx actions to be taken, start them up.
	if len(postKXActions) > 0 {
		go c.takePostKXActions(ru, postKXActions)
	}

	// If this target was the subject of a KX search, trigger event.
	if hadKXSearch {
		c.ntfns.notifyOnKXSearchCompleted(ru)
	}

	return ru, !oldUser, nil
}

func (c *Client) kxCompleted(public *zkidentity.PublicIdentity, r *ratchet.Ratchet,
	initialRV, myResetRV, theirResetRV clientdb.RawRVID) {

	ru, isNew, err := c.initRemoteUser(public, r, true, initialRV, myResetRV,
		theirResetRV, false, "")
	if err != nil && !errors.Is(err, clientintf.ErrSubsysExiting) {
		c.log.Errorf("unable to init user for completed kx: %v", err)
	}

	if err == nil {
		c.ntfns.notifyOnKXCompleted(&initialRV, ru, isNew)
	}
}

// AddInviteOnKX adds a post kx action, based on the initial rv,
// that invites the user to the given groupchat.
func (c *Client) AddInviteOnKX(initialRV, gcID zkidentity.ShortID) error {
	action := clientdb.PostKXAction{
		Type:      clientdb.PKXActionInviteGC,
		DateAdded: time.Now(),
		Data:      gcID.String(),
	}
	return c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		return c.db.AddInitialKXAction(tx, initialRV, action)
	})
}

// WriteNewInvite creates a new invite and writes it to the given writer.
//
// If the optional funds is specified, those funds will be redeemed by the
// remote user prior to accepting the invite.
func (c *Client) WriteNewInvite(w io.Writer, funds *rpc.InviteFunds) (rpc.OOBPublicIdentityInvite, error) {
	return c.kxl.createInvite(w, nil, nil, false, funds)
}

// CreatePrepaidInvite creates a new invite and pushes it to the server,
// pre-paying for the remote user to download it.
func (c *Client) CreatePrepaidInvite(w io.Writer, funds *rpc.InviteFunds) (rpc.OOBPublicIdentityInvite, clientintf.PaidInviteKey, error) {
	return c.kxl.createPrepaidInvite(w, funds)
}

// ReadInvite decodes an invite from the given reader. Note the invite is not
// acted upon until AcceptInvite is called.
func (c *Client) ReadInvite(r io.Reader) (rpc.OOBPublicIdentityInvite, error) {
	return c.kxl.decodeInvite(r)
}

// AcceptInvite blocks until the remote party reponds with us accepting the
// remote party's invitation. The invite should've been created by ReadInvite.
func (c *Client) AcceptInvite(invite rpc.OOBPublicIdentityInvite) error {
	return c.kxl.acceptInvite(invite, false, false)
}

// FetchPrepaidInvite fetches a pre-paid invite from the server, using the
// specified key as decryption key.
func (c *Client) FetchPrepaidInvite(ctx context.Context, key clientintf.PaidInviteKey, w io.Writer) (rpc.OOBPublicIdentityInvite, error) {
	return c.kxl.fetchPrepaidInvite(ctx, key, w)
}

// ResetRatchet requests a ratchet reset with the given user.
func (c *Client) ResetRatchet(uid UserID) error {
	ru, err := c.rul.byID(uid)
	if err != nil {
		return err
	}

	var resetRV clientdb.RawRVID
	var userPub *zkidentity.PublicIdentity
	err = c.dbView(func(tx clientdb.ReadTx) error {
		ab, err := c.db.GetAddressBookEntry(tx, uid)
		if err != nil {
			return err
		}
		resetRV = ab.TheirResetRV
		userPub = ab.ID
		return nil
	})
	if err != nil {
		return err
	}

	ru.log.Infof("Initiating reset via RV %s", resetRV)

	return c.kxl.requestReset(resetRV, userPub)
}

// ResetAllOldRatchets starts the reset ratchet procedure with all users from
// which no message has been received for the passed limit duration.
//
// If the interval is zero, then a default interval of 30 days is used.
//
// If progrChan is specified, each individual reset that is started is reported
// in progrChan.
func (c *Client) ResetAllOldRatchets(limitInterval time.Duration, progrChan chan clientintf.UserID) ([]clientintf.UserID, error) {
	<-c.abLoaded

	if limitInterval == 0 {
		limitInterval = time.Hour * 24 * 30
	}
	limitDate := time.Now().Add(-limitInterval)

	// Load list of in-progress reset attempts.
	kxMap := make(map[clientintf.UserID]struct{})
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		kxs, err := c.db.ListKXs(tx)
		if err != nil {
			return err
		}

		for _, kx := range kxs {
			if !kx.IsForReset {
				continue
			}
			if kx.MediatorID == nil {
				// Should not happen, but sanity check.
				c.log.Errorf("Reset KX %s with nil MediatorID",
					kx.InitialRV)
				continue
			}

			if !kx.Timestamp.Before(limitDate) {
				// KX is still valid.
				kxMap[*kx.MediatorID] = struct{}{}
				continue
			}

			// KX attempt is too old, retry with a new one.
			c.log.Debugf("Removing old reset KX attempt %s with %s from %s",
				kx.InitialRV, kx.MediatorID, kx.Timestamp)
			err := c.db.DeleteKX(tx, kx.InitialRV)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var res []clientintf.UserID
	g := errgroup.Group{}

	ids := c.rul.userList(false)
	for _, uid := range ids {
		ru, err := c.UserByID(uid)
		if err != nil {
			// Should not happen unless user was removed during
			// iteration.
			c.log.Warnf("Unknown user with ID %s", uid)
			continue
		}

		// Skip if we received messages from this user recently.
		_, decTime := ru.LastRatchetTimes()
		if decTime.After(limitDate) {
			continue
		}

		// Skip if we're already KX'ing with them.
		if _, ok := kxMap[uid]; ok {
			continue
		}

		// Start reset attempt. Start all in parallel, so subscriptions
		// to the reset RVs are likely done in a single step.
		res = append(res, uid)
		uid := uid
		g.Go(func() error {
			err := c.ResetRatchet(uid)
			if progrChan != nil {
				select {
				case progrChan <- uid:
				case <-c.ctx.Done():
				}
			}
			return err
		})
	}
	return res, g.Wait()
}

func (c *Client) ListKXs() ([]clientdb.KXData, error) {
	var kxs []clientdb.KXData
	err := c.dbView(func(tx clientdb.ReadTx) error {
		var err error
		kxs, err = c.db.ListKXs(tx)
		return err
	})

	return kxs, err
}

// IsIgnored indicates whether the given client has the ignored flag set.
func (c *Client) IsIgnored(uid clientintf.UserID) (bool, error) {
	ru, err := c.rul.byID(uid)
	if err != nil {
		return false, err
	}
	return ru.IsIgnored(), nil
}

// Ignore changes the setting of the local ignore flag of the specified user.
func (c *Client) Ignore(uid UserID, ignore bool) error {
	ru, err := c.rul.byID(uid)
	if err != nil {
		return err
	}

	isIgnored := ru.IsIgnored()
	switch {
	case isIgnored && ignore:
		return fmt.Errorf("user is already ignored")
	case !isIgnored && !ignore:
		return fmt.Errorf("user was not ignored")
	case ignore:
		c.log.Infof("Ignoring user %s", ru)
	case !ignore:
		c.log.Infof("Un-ignoring user %s", ru)
	}

	ru.SetIgnored(ignore)

	return c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		ab, err := c.db.GetAddressBookEntry(tx, ru.ID())
		if err != nil {
			return err
		}

		ab.Ignored = ignore
		return c.db.UpdateAddressBookEntry(tx, ab)
	})
}

// Block blocks a uid.
func (c *Client) Block(uid UserID) error {
	<-c.abLoaded

	ru, err := c.rul.byID(uid)
	if err != nil {
		return err
	}
	c.log.Infof("Blocking user %s", ru)

	payEvent := "blockUser"
	err = c.sendWithSendQPriority(payEvent, rpc.RMBlock{}, priorityPM, nil, uid)
	if err != nil {
		return err
	}

	// Delete user
	c.rul.del(ru)
	ru.stop()
	return c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		return c.db.RemoveUser(tx, ru.ID(), true)
	})
}

// handleRMBlock handles an incoming block message.
func (c *Client) handleRMBlock(ru *RemoteUser, bl rpc.RMBlock) error {
	c.log.Infof("Blocking user due to received request: %s", ru)

	// Delete user
	c.rul.del(ru)
	ru.stop()
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		return c.db.RemoveUser(tx, ru.ID(), true)
	})
	if err != nil {
		return err
	}

	c.ntfns.notifyOnBlock(ru)
	return nil
}

// RenameUser modifies the nick for the specified user.
func (c *Client) RenameUser(uid UserID, newNick string) error {
	<-c.abLoaded

	ru, err := c.rul.byID(uid)
	if err != nil {
		return err
	}

	_, err = c.UserByNick(newNick)
	if err == nil {
		return fmt.Errorf("user with nick %q already exists", newNick)
	}

	ru.log.Infof("Renaming user to %q", newNick)
	c.rul.modifyUserNick(ru, newNick)

	return c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		ab, err := c.db.GetAddressBookEntry(tx, ru.ID())
		if err != nil {
			return err
		}

		ab.Ignored = ru.IsIgnored()
		ab.NickAlias = newNick
		return c.db.UpdateAddressBookEntry(tx, ab)
	})
}

// SuggestKX sends a message to invitee suggesting they KX with target (through
// the local client).
func (c *Client) SuggestKX(invitee, target UserID) error {
	_, err := c.rul.byID(invitee)
	if err != nil {
		return err
	}

	ruAB, err := c.getAddressBookEntry(target)
	if err != nil {
		return err
	}

	rm := rpc.RMKXSuggestion{Target: *ruAB.ID}
	payEvent := "kxsuggest." + target.String()
	return c.sendWithSendQ(payEvent, rm, invitee)
}

func (c *Client) handleKXSuggestion(ru *RemoteUser, kxsg rpc.RMKXSuggestion) error {
	known := "known"
	targetNick := kxsg.Target.Nick
	targetRu, err := c.rul.byID(kxsg.Target.Identity)
	if err != nil {
		known = "unknown"
	}
	if targetRu != nil {
		targetNick = targetRu.Nick()
	}

	ru.log.Infof("Received suggestion to KX with %s %s (%q)", known,
		kxsg.Target.Identity, targetNick)

	if c.cfg.KXSuggestion != nil {
		c.cfg.KXSuggestion(ru, kxsg.Target)
	}

	c.ntfns.notifyOnKXSuggested(ru, kxsg.Target)
	return nil
}

// LastUserReceivedTime is a record for user and their last decrypted message
// time.
type LastUserReceivedTime struct {
	UID           clientintf.UserID
	LastDecrypted time.Time
}

// ListUsersLastReceivedTime returns the UID and time of last received message
// for all users.
func (c *Client) ListUsersLastReceivedTime() ([]LastUserReceivedTime, error) {
	c.rul.Lock()
	res := make([]LastUserReceivedTime, len(c.rul.m))
	var i int
	for uid, ru := range c.rul.m {
		_, decTS := ru.LastRatchetTimes()
		res[i] = LastUserReceivedTime{UID: uid, LastDecrypted: decTS}
		i += 1
	}
	c.rul.Unlock()

	sort.Slice(res, func(i, j int) bool { return res[i].LastDecrypted.After(res[j].LastDecrypted) })
	return res, nil
}

// handshakeIdleUsers attempts to handshake any users from which no message has
// been received for the passed limitInterval and for which no handshake attempt
// has been made in the given interval as well.
func (c *Client) handshakeIdleUsers() error {
	<-c.abLoaded

	limitInterval := c.cfg.AutoHandshakeInterval
	if limitInterval == 0 {
		// Autohandshake disabled.
		c.log.Debugf("Auto handshake with idle users is disabled")
		return nil
	}
	limitDate := time.Now().Add(-limitInterval)

	users := c.rul.userList(false)
	for _, uid := range users {
		ru, err := c.rul.byID(uid)
		if err != nil {
			continue
		}

		// Skip if we received a message from this user more recently
		// than the limit date.
		_, lastDecTime := ru.LastRatchetTimes()
		if lastDecTime.After(limitDate) {
			continue
		}

		ab, err := c.getAddressBookEntry(uid)
		if err != nil {
			continue
		}

		// If the last handshake attempt time is before the last
		// decryption time, zero the handshake attempt time.
		//
		// This fixes a bug introduced in commit 15690ddfa, after which
		// clients that had performed a handshake prior to this commit
		// would still keep the attempt time recorded (even after
		// successful handshakes). This would mean that an automatic
		// unsub would take place because a new handshake would not be
		// attempted.
		//
		// This fix works by detecting a decryption time that is more
		// recent than the handshake attempt, which implies the clients
		// still have an intact ratchet and the handshake worked (or
		// the ratchet was reset).
		if ab.LastHandshakeAttempt.Before(lastDecTime) {
			ru.log.Infof("Clearing last handshake time (%s) that "+
				"is earlier than last decryption time (%s)",
				ab.LastHandshakeAttempt.Format(time.RFC3339Nano),
				lastDecTime.Format(time.RFC3339Nano))

			err := c.clearLastHandshakeAttemptTime(ru)
			if err != nil {
				ru.log.Warnf("Unable to clear last handshake "+
					"time during init: %v", err)
				continue
			}
			ab.LastHandshakeAttempt = time.Time{}
		}

		// Skip if we already attempted a handshake with this user that
		// has not completed.
		if !ab.LastHandshakeAttempt.IsZero() {
			continue
		}

		// Skip if this user was created more recently than the limit
		// date.
		if !ab.FirstCreated.IsZero() && ab.FirstCreated.After(limitDate) {
			continue
		}

		// Attempt handshake.
		ru.log.Infof("Automatic handshake with user %s due to idle messages",
			strescape.Nick(ru.Nick()))
		err = c.Handshake(uid)
		if err != nil {
			return fmt.Errorf("unable to handshake with %s: %v", uid, err)
		}
	}

	return nil
}

// unsubIdleUsers forcibly unsubscribes and removes from GCs the local client
// admins any users from which no messages have been received since
// limitInterval and from which the last handshake attempt was made at least
// lastHandshakeInterval in the past.
func (c *Client) unsubIdleUsers() error {
	<-c.abLoaded

	limitInterval := c.cfg.AutoRemoveIdleUsersInterval
	if limitInterval == 0 {
		// Auto unsubscribe disabled.
		c.log.Debugf("Auto removal of idle users is disabled")
		return nil
	}
	limitDate := time.Now().Add(-limitInterval)

	// Determine limit date for autohandshake when it is enabled.
	var limitHandshakeDate time.Time
	lastHandshakeInterval := c.cfg.AutoHandshakeInterval / 2
	if lastHandshakeInterval == 0 {
		// Auto handshake disabled.
		c.log.Debugf("Auto removal of idle users is disabled due to " +
			"autohandshake disabled")
		return nil
	}
	limitHandshakeDate = time.Now().Add(-lastHandshakeInterval)

	// Make a map of all GCs we admin.
	gcs, err := c.ListGCs()
	if err != nil {
		return err
	}
	adminGCs := make([]*rpc.RMGroupList, 0, len(gcs))
	for i := range gcs {
		gc := gcs[i]
		if err := c.uidHasGCPerm(&gc.Metadata, c.PublicID()); err != nil {
			// Cannot admin this GC.
			continue
		}
		adminGCs = append(adminGCs, &gcs[i].Metadata)
	}

	// Make a map of all subscribers to our posts.
	ids, err := c.ListPostSubscribers()
	if err != nil {
		return err
	}
	postSubs := make(map[clientintf.UserID]struct{}, len(ids))
	for _, id := range ids {
		postSubs[id] = struct{}{}
	}

	c.log.Debugf("Starting auto unsubscribe of idle users with limitDate %s "+
		"and limitHandshakeDate %s", limitDate.Format(time.RFC3339),
		limitHandshakeDate.Format(time.RFC3339))

	// Build ignore list.
	ignoreList := make(map[clientintf.UserID]struct{}, len(c.cfg.AutoRemoveIdleUsersIgnoreList))
	for _, nick := range c.cfg.AutoRemoveIdleUsersIgnoreList {
		uid, err := c.UIDByNick(nick)
		if err == nil {
			ignoreList[uid] = struct{}{}
		} else {
			c.log.Warnf("User %q in list to ignore from auto remove "+
				"not found", nick)
		}
	}

	users := c.rul.userList(false)
	for _, uid := range users {
		// Do not perform autounsub if this user is in the list to
		// ignore unsubbing.
		if _, ok := ignoreList[uid]; ok {
			c.log.Tracef("Ignoring %s in auto unsubscribe action", uid)
			continue
		}

		ru, err := c.rul.byID(uid)
		if err != nil {
			continue
		}

		// Skip if we received a message from this user more recently
		// than the limit date.
		_, lastDecTime := ru.LastRatchetTimes()
		if lastDecTime.After(limitDate) {
			ru.log.Tracef("Ignoring auto unsubscribe due to "+
				"lastDecTime %s > limitDate %s", lastDecTime,
				limitDate)
			continue
		}

		// Skip if this user is not older than the limit date.
		ab, err := c.getAddressBookEntry(uid)
		if err != nil {
			continue
		}
		if ab.FirstCreated.After(limitDate) {
			ru.log.Tracef("Ignoring auto unsubscribe due to "+
				"firstCreated %s > limitDate %s",
				ab.FirstCreated, limitDate)
			continue
		}

		// Skip if a last handshake was not attempted (or was attempted
		// too recently). This helps prevent the case where this this
		// logic is first run from automatically unsubbing too many
		// people before a handshake is attempted.
		//
		// This check only takes effect if automatic handshaking is
		// enabled.
		handshakeTooRecent := ab.LastHandshakeAttempt.IsZero() ||
			ab.LastHandshakeAttempt.After(limitHandshakeDate)
		if handshakeTooRecent {
			ru.log.Tracef("Ignoring auto unsubscribe due to "+
				"recent handshake %s with limitHandshakeDate %s",
				ab.LastHandshakeAttempt, limitHandshakeDate)
			continue
		}

		// If this user is not a member of any GC we admin and is not
		// subbed to our posts, skip it.
		var gcsToUnsub []clientintf.ID
		for _, gc := range adminGCs {
			if slices.Contains(gc.Members, uid) {
				gcsToUnsub = append(gcsToUnsub, gc.ID)
			}
		}
		_, isSubbedToPosts := postSubs[uid]

		msg := fmt.Sprintf("User %s is idle (last received msg time is %s, "+
			"last handshake attempt time is %s).",
			strescape.Nick(ru.Nick()), lastDecTime.Format(time.RFC3339),
			ab.LastHandshakeAttempt.Format(time.RFC3339))
		if len(gcsToUnsub) == 0 && !isSubbedToPosts {
			msg += fmt.Sprintf(" No actions to take (gcsToUnsub=%d isSubbedToPosts=%v).",
				len(gcsToUnsub), isSubbedToPosts)
			ru.log.Debugf(msg)
			continue
		}

		if isSubbedToPosts {
			msg += " Unsubscribing from local posts."
		}
		if len(gcsToUnsub) > 0 {
			msg += fmt.Sprintf(" Removing from %d GCs.", len(gcsToUnsub))
		}
		ru.log.Info(msg)
		c.ntfns.notifyUnsubscribingIdleRemote(ru, lastDecTime)

		// Forcibly make user unsub from posts.
		if isSubbedToPosts {
			go func() {
				err := c.unsubRemoteFromLocalPosts(ru, false)
				if err != nil {
					ru.log.Warnf("Unable to unsubscribe from local posts: %v", err)
				}
			}()
		}

		// Remove user from any GCs we admin.
		for _, gcid := range gcsToUnsub {
			gcid := gcid
			uid := uid
			go func() {
				err := c.GCKick(gcid, uid, "User is idle for too long")
				if err != nil {
					c.log.Warnf("Unable to remove user %s from GC %s: %v",
						uid, gcid, err)
				}
			}()
		}
	}

	return nil
}

// UpdateLocalAvatar changes the local avatar. If set to nil or an empty slice,
// an update will be sent to remote clients to clear the avatar.
func (c *Client) UpdateLocalAvatar(avatar []byte) error {
	// Restrict max size of avatar stored by default to ensure
	// OOBPublicIdentityInvite is less than the max msg size and can
	// flow through a single server message.
	maxAvatarSize := 200 * 1024 // 200KiB
	if len(avatar) > maxAvatarSize {
		return fmt.Errorf("avatar byte size %d > max avatar size %d",
			len(avatar), maxAvatarSize)
	}

	// Update the DB.
	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		id, err := c.db.LocalID(tx)
		if err != nil {
			return err
		}
		if len(avatar) == 0 {
			id.Public.Avatar = nil
		} else {
			id.Public.Avatar = avatar
		}
		return c.db.UpdateLocalID(tx, id)
	})
	if err != nil {
		return err
	}

	// Update runtime.
	c.profileMtx.Lock()
	c.profile.avatar = avatar
	c.profileMtx.Unlock()

	// Let everyone know the avatar has been updated.
	if avatar == nil {
		avatar = []byte{}
	}
	rmpu := rpc.RMProfileUpdate{
		Avatar: avatar,
	}
	allUsers := c.rul.userList(false)
	payType := "profile.avatar"
	return c.sendWithSendQ(payType, rmpu, allUsers...)
}

func (c *Client) handleProfileUpdate(ru *RemoteUser, rmpu rpc.RMProfileUpdate) error {
	// Update the profile in the DB.
	var ab *clientdb.AddressBookEntry
	var fields []ProfileUpdateField
	if rmpu.Avatar != nil {
		fields = append(fields, ProfileUpdateAvatar)
	}

	if len(fields) == 0 {
		return fmt.Errorf("profile update message without any updates")
	}

	err := c.dbUpdate(func(tx clientdb.ReadWriteTx) error {
		var err error
		ab, err = c.db.GetAddressBookEntry(tx, ru.ID())
		if err != nil {
			return err
		}

		// Update the fields that were sent in this update msg.
		if rmpu.Avatar != nil {
			if len(rmpu.Avatar) == 0 {
				ab.ID.Avatar = nil
			} else {
				ab.ID.Avatar = rmpu.Avatar
			}
		}

		return c.db.UpdateAddressBookEntry(tx, ab)
	})
	if err != nil {
		return err
	}

	fieldsStr := ""
	for i := range fields {
		if i > 0 {
			fieldsStr += ", "
		}
		fieldsStr += string(fields[i])
	}

	ru.log.Infof("Updated profile (fields %s, avatar size %d)",
		fieldsStr, len(ab.ID.Avatar))
	c.ntfns.notifyProfileUpdated(ru, ab, fields)
	return nil
}
