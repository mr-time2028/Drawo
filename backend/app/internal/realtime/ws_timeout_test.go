package realtime

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustListen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	return ln
}

// TestWebSocket_LongLivedIdleConnectionDoesNotTimeout reproduces the original
// user-reported error:
//
//	"read tcp [::1]:8080->[::1]:xxxxx: i/o timeout"
//
// That happened because net/http.Server defaults (ReadTimeout=0 but various
// earlier incarnations had a short ReadTimeout) plus the WS handler's tight
// authDeadline caused Go's http server to enforce an idle-read deadline on
// hijacked WebSocket connections, killing them shortly after handshake.
//
// After the fix:
//   - cmd/serve.go sets ReadTimeout=0, WriteTimeout=0, ReadHeaderTimeout=20s,
//     IdleTimeout=120s (hijacked WS conns inherit no read/write deadlines).
//   - handler.go bumps authDeadline to 15s and joinDeadline to 20s.
//   - Once past handshake, the readPump sets its own pongWait deadline (60s)
//     that is refreshed on every pong; writePump sends pings at 54s.
//
// This test opens a real net/http server with the exact same timeout config
// as cmd/serve.go, completes a full WS handshake, then leaves the socket idle
// for longer than the original buggy deadlines (25s). If the bug regresses,
// the server closes the connection and we time out waiting for the ping or
// get a read error within the idle window.
func TestWebSocket_LongLivedIdleConnectionDoesNotTimeout(t *testing.T) {
	env := newGuestWSTestEnv(t)

	// Build a real *http.Server with the SAME timeout configuration used in
	// cmd/serve.go (this is what serves production traffic).
	srv := &http.Server{
		Handler:           env.handler,
		ReadHeaderTimeout: 20 * time.Second,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}
	ln := mustListen(t)
	t.Cleanup(func() { _ = srv.Close() })
	go func() { _ = srv.Serve(ln) }()

	// Verify the HTTP server is serving first (non-WS request should 400/404).
	{
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://" + ln.Addr().String() + "/")
		t.Logf("probe GET / -> status=%v err=%v", func() int { if resp!=nil{return resp.StatusCode}; return 0 }(), err)
	}

	// Dial a registered user (owner) through the real server.
	wsURL := "ws://" + ln.Addr().String()
	t.Logf("dialing %s", wsURL)
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	var conn *websocket.Conn
	{
		var dialResp *http.Response
		var err error
		conn, dialResp, err = dialer.Dial(wsURL, nil)
		if err != nil {
			if dialResp != nil {
				t.Fatalf("dial err %v, HTTP status=%d", err, dialResp.StatusCode)
			}
			t.Fatalf("dial err %v", err)
		}
	}
	t.Cleanup(func() { _ = conn.Close() })

	// Complete handshake: auth -> auth_ok -> join -> joined (+ initial game_state).
	envEnvelope := MessageEnvelope{
		Type:    EventAuth,
		Payload: json.RawMessage(`{"access_token":"` + env.ownerAccess + `"}`),
	}
	authJSON, err := json.Marshal(envEnvelope)
	require.NoError(t, err)
	t.Logf("sending auth: %s", string(authJSON))
	require.NoError(t, conn.SetWriteDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, authJSON))
	t.Logf("sent auth, waiting for auth_ok")
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	conn.SetCloseHandler(func(code int, text string) error {
		t.Logf("server sent close: code=%d text=%q", code, text)
		return nil
	})
	// Also set a ping handler so any server pings don't cause unexpected control-frame errors
	conn.SetPingHandler(func(appData string) error {
		t.Logf("received server ping %q", appData)
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeWait))
	})
	mtype, data, err := conn.ReadMessage()
	t.Logf("first response: type=%d data=%q err=%v", mtype, string(data), err)
	if err != nil {
		t.Fatalf("read failed after auth send: %v", err)
	}
	var first MessageEnvelope
	require.NoError(t, json.Unmarshal(data, &first))
	require.Equal(t, EventAuthOK, first.Type, "expected auth_ok, got %s: %s", first.Type, string(first.Payload))
	writeEnvelope(t, conn, EventJoin, JoinPayload{Mode: "private", InviteCode: env.room.InviteCode})
	t.Logf("sent join for code %q, draining messages…", env.room.InviteCode)
	// Read until we see 'joined', logging everything.
	joinDeadline := time.Now().Add(5 * time.Second)
	var joinedEnv MessageEnvelope
	for time.Now().Before(joinDeadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		mtype, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read while waiting for joined: %v", err)
		}
		t.Logf("  post-join msg type=%d data=%s", mtype, string(data))
		var env MessageEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if env.Type == EventJoined {
			joinedEnv = env
			break
		}
		if env.Type == EventError {
			t.Logf("ERROR FROM SERVER: %s", string(env.Payload))
		}
	}
	assert.NotEmpty(t, joinedEnv.Payload)

	// From this point on, run a dedicated reader goroutine that drains
	// incoming frames and dispatches control handlers + text messages to a
	// buffer. This avoids the "repeated read on failed connection" panic
	// from gorilla when our code hits a short read deadline and keeps the
	// Pong/Ping handlers installed on the client alive for the whole
	// lifecycle of the socket.
	type frame struct {
		typ int
		b   []byte
		err error
	}
	frameCh := make(chan frame, 32)
	pongCh := make(chan struct{}, 8)
	closeCh := make(chan struct{})
	var closedOnce sync.Once
	closeReader := func() { closedOnce.Do(func() { close(closeCh) }) }
	conn.SetPingHandler(func(appData string) error {
		// Answer server pings with a pong (default behavior).
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeWait))
	})
	conn.SetPongHandler(func(string) error {
		select {
		case pongCh <- struct{}{}:
		default:
		}
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		t.Logf("server sent close: code=%d text=%q", code, text)
		closeReader()
		return nil
	})
	// Set a long overall deadline covering the idle window plus pong wait.
	_ = conn.SetReadDeadline(time.Time{})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			typ, b, err := conn.ReadMessage()
			select {
			case frameCh <- frame{typ, b, err}:
			case <-closeCh:
				return
			}
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		closeReader()
		_ = conn.Close()
		<-readerDone
	})
	// Drain any post-join messages already queued (canvas_sync already seen,
	// game_state may follow, etc.).
	drainTo := time.After(800 * time.Millisecond)
drainLoop:
	for {
		select {
		case f := <-frameCh:
			if f.err != nil {
				t.Fatalf("unexpected error draining post-join frames: %v", f.err)
			}
			t.Logf("  post-join msg type=%d data=%s", f.typ, string(f.b))
		case <-drainTo:
			break drainLoop
		}
	}

	// Idle the connection for longer than the pre-fix 5s deadline.
	const idleWindow = 8 * time.Second
	t.Logf("idling conn for %s…", idleWindow)
	idleTimer := time.NewTimer(idleWindow)
	defer idleTimer.Stop()
idleWait:
	for {
		select {
		case f := <-frameCh:
			if f.err != nil {
				closeReader()
				t.Fatalf("connection failed during idle window (bug regression?): %v", f.err)
			}
			t.Logf("  idle-period msg type=%d data=%s", f.typ, string(f.b))
		case <-idleTimer.C:
			break idleWait
		}
	}

	// Liveness check: send a ping and expect a pong from the server.
	require.NoError(t, conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(2*time.Second)))
	select {
	case <-pongCh:
		// ok — connection is alive
	case f := <-frameCh:
		t.Fatalf("expected pong but got frame/err: %v data=%q", f.err, string(f.b))
	case <-time.After(5 * time.Second):
		t.Fatal("server did not respond to ping after idle window — connection was likely killed")
	}
	closeReader()
}
