package lowlevel

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/companyzero/bisonrelay/client/clientintf"
	"github.com/companyzero/bisonrelay/internal/assert"
	"github.com/companyzero/bisonrelay/rpc"
	"github.com/companyzero/bisonrelay/zkidentity"
)

// TestAttemptsConn ensures connection attempts fail when needed.
func TestAttemptsConn(t *testing.T) {
	t.Parallel()

	var cas atomic.Uint32

	errDialer := errors.New("dialer error")
	errConfirmer := errors.New("confirmer error")

	errorDialer := func(ctx context.Context) (clientintf.Conn, *tls.ConnectionState, error) {
		return nil, nil, errDialer
	}
	noTLSDialer := func(ctx context.Context) (clientintf.Conn, *tls.ConnectionState, error) {
		return offlineConn{}, &tls.ConnectionState{}, nil
	}
	okDialer := func(ctx context.Context) (clientintf.Conn, *tls.ConnectionState, error) {
		// Ensure a unique cert per connection to force tls re-confirm.
		u := cas.Add(1)
		return newSpidConn(), mockTLSConnState(uint8(u)), nil
	}
	errorConfirmer := func(context.Context, *tls.ConnectionState, *zkidentity.PublicIdentity) error {
		return errConfirmer
	}
	okConfirmer := func(context.Context, *tls.ConnectionState, *zkidentity.PublicIdentity) error {
		return nil
	}

	tests := []struct {
		name      string
		dialer    func(ctx context.Context) (clientintf.Conn, *tls.ConnectionState, error)
		confirmer func(context.Context, *tls.ConnectionState, *zkidentity.PublicIdentity) error
		wantErr   error
	}{
		{
			name:      "dialer errors",
			dialer:    errorDialer,
			confirmer: okConfirmer,
			wantErr:   errDialer,
		},
		{
			name:      "dialer no TLS cert",
			dialer:    noTLSDialer,
			confirmer: okConfirmer,
			wantErr:   errNoPeerTLSCert,
		},
		{
			name:      "confirmer rejects cert",
			dialer:    okDialer,
			confirmer: errorConfirmer,
			wantErr:   errConfirmer,
		},
		{
			name:      "conn successful",
			dialer:    okDialer,
			confirmer: okConfirmer,
			wantErr:   nil,
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := ConnKeeperCfg{
				PC:       clientintf.FreePaymentClient{},
				Dialer:   tc.dialer,
				CertConf: tc.confirmer,
			}
			ck := NewConnKeeper(cfg)
			ck.spid = mockServerID.Public
			ck.skipPerformKX = true

			_, err := ck.attemptConn(ctx)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("%s: unexpected error. got %v, want %v", tc.name,
					err, tc.wantErr)
			}
		})
	}
}

// TestKeepsOnline ensures the client is kept online even when connecting
// multiple times to the server fails.
func TestKeepsOnline(t *testing.T) {
	t.Parallel()

	var attempts atomic.Uint32
	maxFails := uint32(5)
	errDialer := errors.New("test dialer error")
	ctxFailedAttemptsDone, cancelCtxFailedAttemps := context.WithCancel(context.Background())
	testDialer := func(ctx context.Context) (clientintf.Conn, *tls.ConnectionState, error) {
		// This dialer will fail the first maxFails attempts,
		// then succeed.
		a := attempts.Add(1)
		if a <= maxFails {
			return nil, nil, errDialer
		}
		cancelCtxFailedAttemps()
		return newSpidConn(), mockTLSConnState(0), nil
	}
	testConfirmer := func(context.Context, *tls.ConnectionState, *zkidentity.PublicIdentity) error {
		return nil
	}
	cfg := ConnKeeperCfg{
		PC:             clientintf.FreePaymentClient{},
		Dialer:         testDialer,
		CertConf:       testConfirmer,
		ReconnectDelay: time.Millisecond,
		//Log:            testutils.TestLoggerSys(t, "XXXX"),
	}
	ck := NewConnKeeper(cfg)
	ck.spid = mockServerID.Public
	ck.skipPerformKX = true

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error)
	go func() { errChan <- ck.Run(ctx) }()

	// First test: multiple attempts until one works.
	select {
	case err := <-errChan:
		t.Fatal(err)
	case <-ctxFailedAttemptsDone.Done():
		// Failed attempts done. Next attempts will succeed.
		time.Sleep(cfg.ReconnectDelay * 10)
	case <-time.After(30 * time.Second):
		t.Fatal("timeout")
	}
	wantAttempts := maxFails + 1
	gotAttempts := attempts.Load()
	if gotAttempts != wantAttempts {
		t.Fatalf("unexpected nb of attempts: got %d, want %d",
			gotAttempts, wantAttempts)
	}

	// Grab the next session. It should be filled.
	sess := ck.NextSession(testTimeoutCtx(t, time.Second))
	if sess == nil {
		t.Fatal("unexpected nil session")
	}

	// Second test: force disconnection due to some error.
	sess.RequestClose(errDialer)

	// NextSession() will return nil.
	sess = ck.NextSession(testTimeoutCtx(t, time.Second))
	if sess != nil {
		t.Fatalf("NextSession() returned non-nil session")
	}

	// NextSession() will reconnect.
	sess = ck.NextSession(testTimeoutCtx(t, time.Second))
	if sess == nil {
		t.Fatalf("unexpected nil session")
	}
	wantAttempts += 1
	gotAttempts = attempts.Load()
	if gotAttempts != wantAttempts {
		t.Fatalf("unexpected nb of attempts: got %d, want %d",
			gotAttempts, wantAttempts)
	}

	// Third test, ask it to go offline, ensure disconnected and ensure no
	// more attempts are made.
	ck.RemainOffline()
	sess = ck.NextSession(testTimeoutCtx(t, time.Second))
	if sess != nil {
		t.Fatalf("unexpected session (expected nil)")
	}
	time.Sleep(cfg.ReconnectDelay * 10)
	gotAttempts = attempts.Load()
	if gotAttempts != wantAttempts {
		t.Fatalf("unexpected nb of attempts: got %d, want %d",
			gotAttempts, wantAttempts)
	}

	// Fourth test: ask it to go online again and ensure the session is
	// produced.
	ck.GoOnline()
	sess = ck.NextSession(testTimeoutCtx(t, time.Second))
	if sess == nil {
		t.Fatalf("unexpected session (expected filled)")
	}
	gotAttempts = attempts.Load()
	wantAttempts += 1
	if gotAttempts != wantAttempts {
		t.Fatalf("unexpected nb of attempts: got %d, want %d",
			gotAttempts, wantAttempts)
	}

	// Final test: cancel the context and expect the func to return.
	cancel()
	select {
	case err := <-errChan:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(time.Millisecond * 10):
		t.Fatal("timeout waiting for function to end")
	}
}

// TestAttemptsSuccessfulKX ensures attemptsKX correctly works when the server
// responds with a valid kx session.
func TestAttemptsSuccessfulKX(t *testing.T) {
	t.Parallel()

	cfg := ConnKeeperCfg{}
	ck := NewConnKeeper(cfg)

	cc, sc := clientServerPipedConn()
	serverKX := mockServerKX(sc)
	cliErrChan := make(chan error)
	svrErrChan := make(chan error)
	go func() {
		_, err := ck.attemptServerKX(cc, &mockServerID.Public)
		cliErrChan <- err
	}()
	go func() {
		// attemptKX() sends the InitialCmdSession, so consume that.
		l := io.LimitedReader{
			R: sc,
			N: 1e8,
		}
		var mode string
		if err := json.NewDecoder(&l).Decode(&mode); err != nil {
			svrErrChan <- err
			return
		}
		svrErrChan <- serverKX.Respond()
	}()

	for svrErrChan != nil && cliErrChan != nil {
		select {
		case err := <-cliErrChan:
			if err != nil {
				t.Fatalf("unexpected error on client KX: %v", err)
			}
			cliErrChan = nil

		case err := <-svrErrChan:
			if err != nil {
				t.Fatalf("unexpected error on server KX: %v", err)
			}
			svrErrChan = nil

		case <-time.After(time.Second):
			t.Fatal("timeout waiting for responses")
		}
	}
}

// TestAttemptsFailedKX ensures attemptsKX returns an error when the server
// responds with an invalid kx.
func TestAttemptsFailedKX(t *testing.T) {
	cfg := ConnKeeperCfg{}
	ck := NewConnKeeper(cfg)

	// Right now, the client-side of the KX'd session can only fail due to
	// writes in the conn failing after writing 'InitialCmdSession'.  So
	// test only that situation.
	cc, sc := clientServerPipedConn()
	cliErrChan := make(chan error)
	go func() {
		_, err := ck.attemptServerKX(cc, &mockServerID.Public)
		cliErrChan <- err
	}()

	go func() {
		// Read the initial cmd in XDR encoding, then close the conn so
		// that further writes from the client fail.
		var b [7]byte
		_, _ = sc.Read(b[:4]) // str len
		_, _ = sc.Read(b[:7]) // str
		_, _ = sc.Read(b[:])  // padding
		_ = sc.Close()
	}()

	select {
	case err := <-cliErrChan:
		if !errors.Is(err, kxError{}) {
			t.Fatalf("unexpected error on client KX: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("timeout waiting for response")
	}
}

// TestAttemptsWelcomeUnknownProps asserts the behavior of the welcome stage
// of the conn keeper when unknown properties are sent by the server.
func TestAttemptsWelcomeUnknownProps(t *testing.T) {
	// Prepare the test harness.
	cfg := ConnKeeperCfg{}
	unwelcomeErrChan := make(chan error, 5)
	cfg.OnUnwelcomeError = func(err error) {
		unwelcomeErrChan <- err
	}
	ck := NewConnKeeper(cfg)
	cc := offlineConn{}
	serverKX := newMockKX()
	cliErrChan := make(chan error)

	// Prepare the welcome msg.
	wmsg := rpc.Welcome{
		Version:    rpc.ProtocolVersion,
		ServerTime: time.Now().Unix(),
		Properties: rpc.SupportedServerProperties(),
	}
	for i := range wmsg.Properties {
		prop := &wmsg.Properties[i]
		switch prop.Key {
		case rpc.PropServerTime:
			prop.Value = strconv.FormatInt(time.Now().Unix(), 10)
		}
	}
	msg := &rpc.Message{Command: rpc.SessionCmdWelcome}

	// Attempting welcome with the default properties should work.
	go func() {
		_, err := ck.attemptWelcome(cc, serverKX)
		cliErrChan <- err
	}()
	serverKX.pushReadMsg(t, msg, wmsg)
	assert.NilErrFromChan(t, cliErrChan)
	assert.ChanNotWritten(t, unwelcomeErrChan, 200*time.Millisecond)

	// Add a new, unknown, optional server property.
	wmsg.Properties = append(wmsg.Properties, rpc.ServerProperty{
		Key:      "****unknown",
		Value:    "*",
		Required: false,
	})
	propIdx := len(wmsg.Properties) - 1

	// Attempting welcome with the new, optional property should work.
	go func() {
		_, err := ck.attemptWelcome(cc, serverKX)
		cliErrChan <- err
	}()
	serverKX.pushReadMsg(t, msg, wmsg)
	assert.NilErrFromChan(t, cliErrChan)
	assert.ChanNotWritten(t, unwelcomeErrChan, 200*time.Millisecond)

	// Switch the property to a required server property should make it
	// fail welcome.
	wmsg.Properties[propIdx].Required = true
	go func() {
		_, err := ck.attemptWelcome(cc, serverKX)
		cliErrChan <- err
	}()
	serverKX.pushReadMsg(t, msg, wmsg)
	err := assert.ChanWritten(t, cliErrChan)
	assert.ErrorIs(t, err, UnwelcomeError{})
	reason := err.(UnwelcomeError).Reason
	assert.DeepEqual(t, reason, "unhandled server property: ****unknown")
	err = assert.ChanWritten(t, unwelcomeErrChan)
	assert.ErrorIs(t, err, UnwelcomeError{})
}

// TestAttemptsWelcomeUnknownMaxMsgSizeVersion tests that if the server sends
// a max msg size version that is unknown to the client, the connection is
// dropped.
func TestAttemptsWelcomeUnknownMaxMsgSizeVersion(t *testing.T) {
	// Prepare the test harness.
	cfg := ConnKeeperCfg{}
	unwelcomeErrChan := make(chan error, 5)
	cfg.OnUnwelcomeError = func(err error) {
		unwelcomeErrChan <- err
	}
	ck := NewConnKeeper(cfg)
	cc := offlineConn{}
	serverKX := newMockKX()
	cliErrChan := make(chan error)

	// Prepare the welcome msg.
	wmsg := rpc.Welcome{
		Version:    rpc.ProtocolVersion,
		ServerTime: time.Now().Unix(),
		Properties: rpc.SupportedServerProperties(),
	}
	for i := range wmsg.Properties {
		prop := &wmsg.Properties[i]
		switch prop.Key {
		case rpc.PropServerTime:
			prop.Value = strconv.FormatInt(time.Now().Unix(), 10)
		case rpc.PropMaxMsgSizeVersion:
			prop.Value = strconv.FormatInt(1<<24-1, 10)
		}
	}
	msg := &rpc.Message{Command: rpc.SessionCmdWelcome}

	// Attempting welcome should fail.
	go func() {
		_, err := ck.attemptWelcome(cc, serverKX)
		cliErrChan <- err
	}()
	serverKX.pushReadMsg(t, msg, wmsg)
	err := assert.ChanWritten(t, cliErrChan)
	assert.ErrorIs(t, err, UnwelcomeError{})
	reason := err.(UnwelcomeError).Reason
	assert.DeepEqual(t, reason, "server did not send a supported max msg size version")
	err = assert.ChanWritten(t, unwelcomeErrChan)
	assert.ErrorIs(t, err, UnwelcomeError{})
}

// TestAttemptsWelcomeUnknownProtocolVersion asserts that attempting to connect
// to a server with a protocol version higher than supported triggers an error.
func TestAttemptsWelcomeUnknownProtocolVersion(t *testing.T) {
	// Prepare the test harness.
	cfg := ConnKeeperCfg{}
	unwelcomeErrChan := make(chan error, 5)
	cfg.OnUnwelcomeError = func(err error) {
		unwelcomeErrChan <- err
	}
	ck := NewConnKeeper(cfg)
	cc := offlineConn{}
	serverKX := newMockKX()
	cliErrChan := make(chan error)

	// Prepare the welcome msg.
	wmsg := rpc.Welcome{
		Version:    rpc.ProtocolVersion + 1,
		ServerTime: time.Now().Unix(),
		Properties: rpc.SupportedServerProperties(),
	}
	msg := &rpc.Message{Command: rpc.SessionCmdWelcome}

	// Attempting welcome should fail.
	go func() {
		_, err := ck.attemptWelcome(cc, serverKX)
		cliErrChan <- err
	}()
	serverKX.pushReadMsg(t, msg, wmsg)
	err := assert.ChanWritten(t, cliErrChan)
	assert.ErrorIs(t, err, UnwelcomeError{})
	reason := err.(UnwelcomeError).Reason
	assert.DeepEqual(t, reason, fmt.Sprintf("protocol version mismatch: got %v wanted %v",
		rpc.ProtocolVersion+1, rpc.ProtocolVersion))
	err = assert.ChanWritten(t, unwelcomeErrChan)
	assert.ErrorIs(t, err, UnwelcomeError{})
}

func TestAttemptsWelcomePingLimit(t *testing.T) {
	tests := []struct {
		name         string
		pingInterval time.Duration
		pingLimit    string // seconds, encoded as string
		wantErr      error
		want         time.Duration
	}{{
		name:         "default ping interval",
		pingInterval: rpc.DefaultPingInterval,
		pingLimit:    strconv.Itoa(int(rpc.PropPingLimitDefault / time.Second)),
		want:         rpc.DefaultPingInterval,
	}, {
		name:         "long ping interval",
		pingInterval: 2 * time.Minute,
		pingLimit:    "300", // 5 minutes
		want:         2 * time.Minute,
	}, {
		name:         "ping interval higher than ping limit",
		pingInterval: 10 * time.Minute,
		pingLimit:    "300", // 5 minutes
		want:         5 * time.Minute * 3 / 4,
	}, {
		name:         "short ping interval with short ping limit",
		pingInterval: 5 * time.Second,
		pingLimit:    "10",
		want:         5 * time.Second,
	}, {
		name:         "zero ping interval",
		pingInterval: 0,
		pingLimit:    "300",
		want:         0,
	}, {
		name:         "zero ping interval with short ping limit",
		pingInterval: 0,
		pingLimit:    "10",
		want:         0,
	}, {
		name:         "long ping interval with short ping limit",
		pingInterval: time.Minute,
		pingLimit:    "10",
		wantErr:      errShortPingLimit,
	}}

	type res struct {
		err  error
		sess *serverSession
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := ConnKeeperCfg{
				PingInterval: tc.pingInterval,
			}
			unwelcomeErrChan := make(chan error, 5)
			cfg.OnUnwelcomeError = func(err error) {
				unwelcomeErrChan <- err
			}

			ck := NewConnKeeper(cfg)
			cc := offlineConn{}
			serverKX := newMockKX()

			// Prepare the welcome msg.
			wmsg := rpc.Welcome{
				Version:    rpc.ProtocolVersion,
				ServerTime: time.Now().Unix(),
				Properties: rpc.SupportedServerProperties(),
			}
			for i := range wmsg.Properties {
				prop := &wmsg.Properties[i]
				switch prop.Key {
				case rpc.PropServerTime:
					prop.Value = strconv.FormatInt(time.Now().Unix(), 10)
				case rpc.PropPingLimit:
					prop.Value = tc.pingLimit
				}
			}
			msg := &rpc.Message{Command: rpc.SessionCmdWelcome}

			// Attempting welcome should fail.
			resChan := make(chan res, 1)
			go func() {
				sess, err := ck.attemptWelcome(cc, serverKX)
				resChan <- res{sess: sess, err: err}
			}()
			serverKX.pushReadMsg(t, msg, wmsg)
			gotRes := assert.ChanWritten(t, resChan)
			assert.ErrorIs(t, gotRes.err, tc.wantErr)
			if gotRes.sess != nil {
				assert.DeepEqual(t, gotRes.sess.pingInterval, tc.want)
			}
		})
	}
}
