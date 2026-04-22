package rpcserver

import (
	"context"
	"sync"

	"github.com/decred/slog"
)

// liveServerStreams tracks all current streams for a given server stream type.
//
// Contrary to serverStreams, this does not replay old events and does not
// persist them to disk.
type liveServerStreams[T clientrpcMsg] struct {
	mtx    sync.Mutex
	nextID int32
	m      map[int32]serverStream[T]
	prefix string
	log    slog.Logger
}

// runStream runs the live stream until the stream context is done.
func (cs *liveServerStreams[T]) runStream(ctx context.Context, stream serverStreamFor[T]) error {
	cs.mtx.Lock()
	ctx, cancel := context.WithCancel(ctx)
	sid := cs.nextID + 1
	cs.nextID += 1
	cs.m[sid] = serverStream[T]{cancel: cancel, stream: stream}
	cs.mtx.Unlock()

	cs.log.Tracef("Started running live stream %d for %s", sid, cs.prefix)

	<-ctx.Done()

	cs.mtx.Lock()
	delete(cs.m, sid)
	cs.mtx.Unlock()

	return ctx.Err()
}

// send sends the notification to all active live streams.
func (cs *liveServerStreams[T]) send(ntfn T) {
	cs.mtx.Lock()
	streams := make(map[int32]serverStream[T], len(cs.m))
	for id, s := range cs.m {
		streams[id] = s
	}
	cs.mtx.Unlock()

	for id, s := range streams {
		err := s.stream.Send(ntfn)
		if err != nil {
			cs.log.Errorf("Unable to send %s notification to live stream %d: %v",
				cs.prefix, id, err)
			s.cancel()
		}
	}
}

// newLiveServerStreams creates a new set of live streams for a given message type.
func newLiveServerStreams[T clientrpcMsg](prefix string, log slog.Logger) *liveServerStreams[T] {
	return &liveServerStreams[T]{
		log:    log,
		prefix: prefix,
		m:      make(map[int32]serverStream[T]),
	}
}
