package main

import (
	"sync"

	"github.com/companyzero/bisonrelay/clientrpc/types"
)

// Player represents a player in the game
type Player struct {
	ID    string
	Move  string
	Ready bool

	stream   types.PluginService_CallActionServer
	notifier types.PluginService_InitServer
}

func NewPlayer(id string) *Player {
	return &Player{
		ID: id,
	}
}

type PlayerSessions struct {
	mu       sync.Mutex
	sessions map[string]*Player // Map client ID to Player
}

func NewPlayerSessions() *PlayerSessions {
	return &PlayerSessions{
		sessions: make(map[string]*Player),
	}
}

func (ps *PlayerSessions) AddOrUpdatePlayer(player *Player) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.sessions[player.ID] = player
}

func (ps *PlayerSessions) RemovePlayer(clientID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.sessions, clientID)
}

func (ps *PlayerSessions) GetPlayer(clientID string) (*Player, bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	player, exists := ps.sessions[clientID]
	return player, exists
}

type WaitingRoom struct {
	mu    sync.Mutex
	queue []*Player
}

func NewWaitingRoom() *WaitingRoom {
	return &WaitingRoom{
		queue: make([]*Player, 0),
	}
}

func (wr *WaitingRoom) AddPlayer(player *Player) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	// don't add repeated players
	for _, p := range wr.queue {
		if p.ID == player.ID {
			return
		}
	}
	wr.queue = append(wr.queue, player)
}

func (wr *WaitingRoom) ReadyPlayers() ([]*Player, bool) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	if len(wr.queue) >= 2 {
		players := wr.queue[:2]
		wr.queue = wr.queue[2:]
		return players, true
	}
	return nil, false
}

func (wr *WaitingRoom) GetPlayer(clientID string) (*Player, bool) {
	wr.mu.Lock()
	defer wr.mu.Unlock()
	for _, player := range wr.queue {
		if player.ID == clientID {
			return player, true
		}
	}
	return nil, false
}

func (wr *WaitingRoom) RemovePlayer(clientID string) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	for i, player := range wr.queue {
		if player.ID == clientID {
			wr.queue = append(wr.queue[:i], wr.queue[i+1:]...)
			break
		}
	}
}
