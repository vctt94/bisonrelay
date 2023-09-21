package clientdb

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/companyzero/bisonrelay/client/clientintf"
	"github.com/companyzero/bisonrelay/inidb"
	"github.com/companyzero/bisonrelay/rpc"
	"github.com/companyzero/bisonrelay/zkidentity"
)

type PTInvite struct {
	Invite   rpc.RMPokerTableInvite
	User     UserID
	ID       uint64
	Accepted bool
}

func (i *PTInvite) marshal() (string, error) {
	blob, err := json.Marshal(i)
	if err != nil {
		return "", fmt.Errorf("could not marshal invite record: %v", err)
	}
	return hex.EncodeToString(blob), nil
}

func (i *PTInvite) unmarshal(s string) error {
	blob, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	return json.Unmarshal(blob, i)
}

func (db *DB) GetPT(tx ReadTx, id zkidentity.ShortID) (rpc.RMPokerTableList, error) {
	var pt rpc.RMPokerTableList
	filename := filepath.Join(db.root, pokertableDir, id.String())
	err := db.readPT(filename, &pt)
	if errors.Is(err, ErrNotFound) {
		return pt, fmt.Errorf("gc %s: %w", id, ErrNotFound)
	}
	return pt, err
}

// readGC reads the gc from the given filename into gl.
func (db *DB) readPT(filename string, gc *rpc.RMPokerTableList) error {
	gcJSON, err := os.ReadFile(filename)
	if err != nil && os.IsNotExist(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(gcJSON, &gc)
}

func (db *DB) SavePT(tx ReadWriteTx, pt rpc.RMPokerTableList) error {
	ptDir := filepath.Join(db.root, pokertableDir)
	filename := filepath.Join(ptDir, pt.ID.String())
	return db.saveJsonFile(filename, pt)
}

// SetPTAlias sets the alias for the specified PT ID as the name. If gcID is
// empty, the name is removed from the alias map. If gcID is filled but name is
// empty, the entry that points to the specified gcID is removed.
func (db *DB) SetPTAlias(tx ReadWriteTx, gcID zkidentity.ShortID, name string) (
	map[string]zkidentity.ShortID, error) {

	aliasMap, err := db.GetPTAliases(tx)
	if err != nil {
		return nil, err
	}

	if gcID.IsEmpty() {
		delete(aliasMap, name)
	} else {
		// Remove old aliases.
		for name, id := range aliasMap {
			if id == gcID {
				delete(aliasMap, name)
			}
		}

		if name != "" {
			aliasMap[name] = gcID
		}
	}

	filename := filepath.Join(db.root, ptAliasesFile)
	if err := db.saveJsonFile(filename, &aliasMap); err != nil {
		return nil, err
	}

	return aliasMap, nil
}

func (db *DB) GetPTAliases(tx ReadTx) (map[string]zkidentity.ShortID, error) {
	var aliasMap map[string]zkidentity.ShortID

	filename := filepath.Join(db.root, ptAliasesFile)
	err := db.readJsonFile(filename, &aliasMap)
	if errors.Is(err, ErrNotFound) {
		// New map.
		return map[string]zkidentity.ShortID{}, nil
	}
	return aliasMap, err
}

// FindGCInvite looks for an invite to a GC with a given token.
func (db *DB) FindPTInvite(tx ReadTx, ptID zkidentity.ShortID, token uint64) (rpc.RMGroupInvite, UserID, uint64, error) {
	fail := func(err error) (rpc.RMGroupInvite, UserID, uint64, error) {
		return rpc.RMGroupInvite{}, UserID{}, 0, err
	}

	var dbi GCInvite
	records := db.invites.Records(invitesTable)
	for k, v := range records {
		id, err := atoi(k)
		if err != nil {
			return fail(fmt.Errorf("invalid invite key: %v", err))
		}

		err = dbi.unmarshal(v)
		if err != nil {
			return fail(fmt.Errorf("unable to unmarshal db gc invite: %v", err))
		}

		if dbi.Invite.ID == ptID && dbi.Invite.Token == token {
			return dbi.Invite, dbi.User, id, nil
		}
	}

	return fail(fmt.Errorf("gc invite: %w", ErrNotFound))
}

func (db *DB) AddPTInvite(tx ReadWriteTx, user UserID, invite rpc.RMPokerTableInvite) (uint64, error) {
	db.invites.NewTable(invitesTable)

	newID := func() uint64 {
		return 100000 + (db.mustRandomUint64() % (1000000 - 100000))
	}

	dbi := PTInvite{
		Invite: invite,
		User:   user,
		ID:     newID(),
	}

	// Get a random invite id.
	_, err := db.invites.Get(invitesTable, itoa(dbi.ID))
	for err == nil {
		dbi.ID = newID()
		_, err = db.invites.Get(invitesTable, itoa(dbi.ID))
	}
	if !errors.Is(err, inidb.ErrNotFound) {
		return 0, err
	}

	blob, err := dbi.marshal()
	if err != nil {
		return 0, err
	}

	if err := db.invites.Set(invitesTable, itoa(dbi.ID), blob); err != nil {
		return 0, err
	}

	if err := db.invites.Save(); err != nil {
		return 0, err
	}

	return dbi.ID, nil
}

func (db *DB) GetPTInvite(tx ReadTx, inviteID uint64) (rpc.RMPokerTableInvite, UserID, error) {
	var invite rpc.RMPokerTableInvite

	blob, err := db.invites.Get(invitesTable, itoa(inviteID))
	if err != nil {
		if errors.Is(err, inidb.ErrNotFound) {
			return invite, UserID{}, fmt.Errorf("invite %d: %w", inviteID, ErrNotFound)
		}
		return invite, UserID{}, err
	}

	var dbi PTInvite
	err = dbi.unmarshal(blob)
	if err != nil {
		return invite, UserID{}, fmt.Errorf("unable to unmarshal db pt invite")
	}

	return dbi.Invite, dbi.User, nil
}

func (db *DB) MarkPTInviteAccepted(tx ReadWriteTx, inviteID uint64) error {
	blob, err := db.invites.Get(invitesTable, itoa(inviteID))
	if err != nil {
		if errors.Is(err, inidb.ErrNotFound) {
			return fmt.Errorf("invite %d: %w", inviteID, ErrNotFound)
		}
		return err
	}

	var dbi GCInvite
	if err := dbi.unmarshal(blob); err != nil {
		return fmt.Errorf("unable to unmarshal db pt invite")
	}

	dbi.Accepted = true

	blob, err = dbi.marshal()
	if err != nil {
		return err
	}

	if err := db.invites.Set(invitesTable, itoa(dbi.ID), blob); err != nil {
		return err
	}
	return db.invites.Save()
}

// ListPTInvites lists the PT invites. If pt is specified, lists only invites
// for the specified PTID.
func (db *DB) ListPTInvites(tx ReadTx, gc *zkidentity.ShortID) ([]*PTInvite, error) {
	records := db.invites.Records(invitesTable)
	res := make([]*PTInvite, 0, len(records))
	for _, v := range records {
		dbi := new(PTInvite)
		err := dbi.unmarshal(v)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal db gc invite: %v", err)
		}

		if gc != nil && !dbi.Invite.ID.ConstantTimeEq(gc) {
			continue
		}

		res = append(res, dbi)
	}

	return res, nil
}

func (db *DB) DelPTInvite(tx ReadWriteTx, inviteID uint64) error {
	if err := db.invites.Del(invitesTable, itoa(inviteID)); err != nil {
		return err
	}
	return db.invites.Save()
}

func (db *DB) ListPTs(tx ReadTx) ([]rpc.RMPokerTableList, error) {
	gcDir := filepath.Join(db.root, pokertableDir)
	entries, err := os.ReadDir(gcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	groups := make([]rpc.RMPokerTableList, 0, len(entries))
	for _, v := range entries {
		if v.IsDir() {
			continue
		}

		fname := filepath.Join(gcDir, v.Name())
		if strings.HasSuffix(fname, gcBlockListExt) {
			continue
		}

		var gc rpc.RMPokerTableList
		err := db.readPT(fname, &gc)
		if err != nil {
			db.log.Warnf("Unable to read gc file for listing %s: %v",
				fname, err)
			continue
		}

		groups = append(groups, gc)
	}

	return groups, nil
}

// CacheReceivedGCM stores a cached received GC message.
func (db *DB) CacheReceivedPTM(tx ReadWriteTx, rgcm clientintf.ReceivedGCMsg) error {
	filename := filepath.Join(db.root, cachedPTMsDir, rgcm.MsgID.String())
	return db.saveJsonFile(filename, rgcm)
}
