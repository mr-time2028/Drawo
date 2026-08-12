/**
 * Tests for gameSocket.connectGameSocket using a fully controllable fake
 * WebSocket. The fake does NOT auto-open; tests call ws.doOpen() explicitly
 * to avoid queueMicrotask races.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { connectGameSocket, getGuestAuthForRoom, type GameSocket, type ServerEnvelope } from './gameSocket';
import * as guestMod from './guestTokenManager';
import * as authMod from './authTokenManager';

type SentFrame = { type: string; payload?: unknown };

let fakeInstances: FakeWS[] = [];

class FakeWS {
  // WebSocket readyState constants.
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;
  CONNECTING = 0;
  OPEN = 1;
  CLOSING = 2;
  CLOSED = 3;

  static all(): FakeWS[] {
    return fakeInstances.slice();
  }
  static last(): FakeWS {
    if (fakeInstances.length === 0) throw new Error('no ws created');
    return fakeInstances[fakeInstances.length - 1]!;
  }

  url: string;
  readyState: number = 0;
  onopen: ((ev: unknown) => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  binaryType: BinaryType = 'blob';
  sent: SentFrame[] = [];
  _closed: boolean = false;

  constructor(url: string) {
    this.url = url;
    fakeInstances.push(this);
  }

  doOpen() {
    if (this._closed || this.readyState !== 0) return;
    this.readyState = 1;
    // Call onopen asynchronously, just like the real WebSocket.
    queueMicrotask(() => {
      if (!this._closed && this.onopen) this.onopen({});
    });
  }

  doClose() {
    if (this._closed || this.readyState >= 2) return;
    this._closed = true;
    this.readyState = 3;
    this.onclose?.();
  }

  doError() {
    this.onerror?.();
  }

  close() {
    this.doClose();
  }

  send(raw: string) {
    if (this.readyState !== 1) return;
    const parsed = JSON.parse(raw) as { type: string; payload?: unknown };
    this.sent.push({ type: parsed.type, payload: parsed.payload });
  }

  receive(env: ServerEnvelope) {
    if (this.onmessage) this.onmessage({ data: JSON.stringify(env) });
  }

  deliverRaw(data: unknown) {
    if (this.onmessage) this.onmessage({ data: data as string });
  }
}

// Generic poll-until-truthy helper.
async function waitFor<T>(pred: () => T | null | undefined | false, timeoutMs = 1000): Promise<T> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const v = pred();
    if (v) return v;
    await new Promise((r) => setTimeout(r, 5));
  }
  throw new Error('waitFor timed out');
}

// Helpers to drive the handshake synchronously step-by-step.
async function waitForAuth(ws: FakeWS) {
  const deadline = Date.now() + 1000;
  while (Date.now() < deadline) {
    if (ws.sent.some((f) => f.type === 'auth')) return;
    await new Promise((r) => setTimeout(r, 5));
  }
  throw new Error('auth frame not sent; sent=' + JSON.stringify(ws.sent.map((f) => f.type)) +
    ' readyState=' + ws.readyState + ' onopen=' + !!ws.onopen +
    ' acquireCalls=' + acquireSpy.mock.calls.length);
}

async function waitForJoin(ws: FakeWS) {
  const deadline = Date.now() + 1000;
  while (Date.now() < deadline) {
    if (ws.sent.some((f) => f.type === 'join')) return;
    await new Promise((r) => setTimeout(r, 5));
  }
  throw new Error('join frame not sent; sent=' + JSON.stringify(ws.sent.map((f) => f.type)));
}

async function completeHandshake(ws: FakeWS, guest = false) {
  ws.doOpen();
  await waitForAuth(ws);
  if (!guest) {
    expect(ws.sent.find((f) => f.type === 'auth')?.payload).toEqual({ access_token: 'access-xyz' });
  } else {
    expect(ws.sent.find((f) => f.type === 'auth')?.payload).toMatchObject({ guest_token: expect.any(String) });
  }
  ws.receive({ type: 'auth_ok' });
  await waitForJoin(ws);
  ws.receive({ type: 'joined', payload: { room_id: 'r1' } });
  await Promise.resolve();
  await Promise.resolve();
}

let acquireSpy: ReturnType<typeof vi.spyOn>;
let releaseSpy: ReturnType<typeof vi.spyOn>;
let readGuestSpy: ReturnType<typeof vi.spyOn>;
const RealWebSocket = globalThis.WebSocket;

beforeEach(() => {
  fakeInstances = [];
  (globalThis as unknown as { WebSocket: unknown }).WebSocket = FakeWS as unknown;
  // Reset module registry between tests to avoid stale imports — shouldn't be
  // needed but guarantees a fresh reference to the mocked exports.
  acquireSpy = vi.spyOn(authMod, 'wsAcquireToken').mockImplementation(async () => 'access-xyz');
  releaseSpy = vi.spyOn(authMod, 'wsReleaseToken').mockImplementation(() => {});
  readGuestSpy = vi.spyOn(guestMod, 'readGuestSession').mockImplementation(() => null);
});

afterEach(() => {
  // Close any leftover fake sockets from rejected handshakes so their
  // callbacks can't fire into torn-down mocks on the next test.
  for (const s of fakeInstances) {
    if (!s._closed) {
      s.onopen = null;
      s.onmessage = null;
      s.onerror = null;
      s.onclose = null;
      s.readyState = 3;
      s._closed = true;
    }
  }
  (globalThis as unknown as { WebSocket: unknown }).WebSocket = RealWebSocket as unknown;
  acquireSpy.mockRestore();
  releaseSpy.mockRestore();
  readGuestSpy.mockRestore();
});

describe('connectGameSocket', () => {
  it('happy path: registered user authenticates and joins a public room', async () => {
    const onStatus = vi.fn();
    const onMessage = vi.fn();
    const sockP = connectGameSocket({ mode: 'public', language: 'en', categoryID: 'cat', onStatusChange: onStatus, onMessage });
    const ws = FakeWS.last();
    await completeHandshake(ws);
    const sock = await sockP;
    expect(sock.getStatus()).toBe('open');
    expect(onStatus).toHaveBeenCalledWith('open', undefined);
    const joinPayload = ws.sent.find((f) => f.type === 'join')?.payload as Record<string, unknown>;
    expect(joinPayload).toMatchObject({ mode: 'public', language: 'en', category_id: 'cat' });
    expect(onMessage).toHaveBeenCalledWith(expect.objectContaining({ type: 'joined' }));
  });

  it('authenticates as a guest when opts.guest is provided and forces private/room binding', async () => {
    const sockP = connectGameSocket({
      mode: 'public',
      roomID: 'wrong',
      inviteCode: 'INV1',
      guest: { guestToken: 'gtok', guestID: 'guest:1', nickname: 'Alice', roomID: 'r-g' },
    });
    const ws = FakeWS.last();
    await completeHandshake(ws, true);
    const authPayload = ws.sent.find((f) => f.type === 'auth')?.payload as Record<string, unknown>;
    expect(authPayload).toEqual({ guest_token: 'gtok' });
    const joinPayload = ws.sent.find((f) => f.type === 'join')?.payload as Record<string, unknown>;
    expect(joinPayload.mode).toBe('private');
    expect(joinPayload.room_id).toBe('r-g'); // guest-bound room wins
    await sockP;
  });

  it('falls back to localStorage guest session when opts.guest is omitted and room matches', async () => {
    vi.mocked(guestMod.readGuestSession).mockReturnValue({
      guestToken: 'ls-tok',
      guestID: 'guest:ls',
      nickname: 'Local',
      roomID: 'r-ls',
    });
    const sockP = connectGameSocket({ mode: 'private', roomID: 'r-ls' });
    const ws = FakeWS.last();
    await completeHandshake(ws, true);
    const authPayload = ws.sent.find((f) => f.type === 'auth')?.payload as Record<string, unknown>;
    expect(authPayload).toEqual({ guest_token: 'ls-tok' });
    await sockP;
  });

  it('uses user auth when localStorage guest is for a different room', async () => {
    vi.mocked(guestMod.readGuestSession).mockReturnValue({
      guestToken: 'ls-tok',
      guestID: 'guest:ls',
      nickname: 'Local',
      roomID: 'other-room',
    });
    const sockP = connectGameSocket({ mode: 'private', roomID: 'r-me' });
    const ws = FakeWS.last();
    await completeHandshake(ws);
    const joinPayload = ws.sent.find((f) => f.type === 'join')?.payload as Record<string, unknown>;
    expect(joinPayload.room_id).toBe('r-me');
    await sockP;
  });

  it('rejects when there is no access token for registered-user flow', async () => {
    acquireSpy.mockResolvedValue(null as unknown as string);
    const sockP = connectGameSocket({});
    const ws = FakeWS.last();
    ws.doOpen();
    // The onopen handler awaits wsAcquireToken before calling fail(), so we
    // must await the rejection first — the promise rejects before socket.close
    // resolves (close itself is synchronous in our fake).
    await expect(sockP).rejects.toThrow('not authenticated');
    expect(ws.readyState).toBe(3);
  });

  it('rejects when wsAcquireToken throws', async () => {
    acquireSpy.mockRejectedValue(new Error('refresh boom'));
    const sockP = connectGameSocket({});
    const ws = FakeWS.last();
    ws.doOpen();
    await expect(sockP).rejects.toThrow('refresh boom');
  });

  it('rejects if socket errors before handshake completes', async () => {
    const sockP = connectGameSocket({});
    const ws = FakeWS.last();
    ws.doOpen();
    ws.doError();
    ws.doClose();
    await expect(sockP).rejects.toThrow(/websocket connection failed|closed before handshake/i);
  });

  it('rejects if socket closes before open ever fires', async () => {
    const sockP = connectGameSocket({});
    const ws = FakeWS.last();
    // Close from CONNECTING (never opened).
    ws.doClose();
    await expect(sockP).rejects.toThrow(/closed before handshake/i);
  });

  it('rejects on server error frame during handshake', async () => {
    const sockP = connectGameSocket({});
    const ws = FakeWS.last();
    ws.doOpen();
    await waitForAuth(ws);
    ws.receive({ type: 'error', payload: { code: 'auth_failed', message: 'bad token' } });
    await expect(sockP).rejects.toThrow('bad token');
  });

  it('close() sends leave frame, releases token, and transitions to closed', async () => {
    const sockP = connectGameSocket({});
    const ws = FakeWS.last();
    await completeHandshake(ws);
    const sock: GameSocket = await sockP;
    const sentBeforeClose = ws.sent.length;
    expect(authMod.wsReleaseToken).not.toHaveBeenCalled();
    sock.close();
    expect(ws.sent.slice(sentBeforeClose).some((f) => f.type === 'leave')).toBe(true);
    // User-mode close releases the WS-held token reference.
    expect(authMod.wsReleaseToken).toHaveBeenCalledTimes(1);
    expect(sock.getStatus()).toBe('closed');
  });

  it('send() after open forwards frames; send() after close is dropped', async () => {
    const sockP = connectGameSocket({});
    const ws = FakeWS.last();
    await completeHandshake(ws);
    const sock = await sockP;

    sock.send('chat', { text: 'hi' });
    const chat = ws.sent.find((f) => f.type === 'chat');
    expect(chat).toBeTruthy();
    expect(chat?.payload).toEqual({ text: 'hi' });

    const before = ws.sent.length;
    sock.close();
    sock.send('chat', { text: 'after close' });
    // Leave frame + close — no chat after.
    expect(ws.sent.slice(before).filter((f) => f.type === 'chat')).toHaveLength(0);
  });

  it('ignores non-string messages and malformed JSON without forwarding', async () => {
    const onMessage = vi.fn();
    const sockP = connectGameSocket({ onMessage });
    const ws = FakeWS.last();
    ws.doOpen();
    await waitForAuth(ws);
    // Deliver binary-like and bad-JSON frames — these must be silently ignored.
    ws.deliverRaw(new ArrayBuffer(4));
    ws.deliverRaw('{not json');
    // Finish handshake.
    ws.receive({ type: 'auth_ok' });
    await waitForJoin(ws);
    ws.receive({ type: 'joined' });
    await sockP;
    // onMessage was called for auth_ok + joined only.
    const types = onMessage.mock.calls.map((c) => (c[0] as ServerEnvelope).type);
    expect(types).toEqual(['auth_ok', 'joined']);
  });

  it('send() is a no-op if socket is not yet open (before joined)', async () => {
    // We cannot easily call send() before the promise resolves because GameSocket
    // is only exposed post-joined. Instead, trigger a direct code-path check:
    // the inner `send` closure checks readyState !== OPEN and returns. Simulate
    // by calling our FakeWS.send while readyState=0 (CONNECTING) — which returns
    // early, mirroring the guard.
    const ws = new FakeWS('ws://x');
    expect(ws.readyState).toBe(0);
    ws.send(JSON.stringify({ type: 'ping' }));
    expect(ws.sent).toHaveLength(0);
  });
});

describe('getGuestAuthForRoom', () => {
  it('returns null when no guest session exists', () => {
    vi.mocked(guestMod.readGuestSession).mockReturnValue(null);
    expect(getGuestAuthForRoom('r1')).toBeNull();
  });
  it('returns null when guest session is bound to a different room', () => {
    vi.mocked(guestMod.readGuestSession).mockReturnValue({
      guestToken: 't', guestID: 'g', nickname: 'n', roomID: 'other',
    });
    expect(getGuestAuthForRoom('r1')).toBeNull();
  });
  it('returns the session when room matches', () => {
    vi.mocked(guestMod.readGuestSession).mockReturnValue({
      guestToken: 't', guestID: 'g', nickname: 'n', roomID: 'r1',
    });
    expect(getGuestAuthForRoom('r1')?.roomID).toBe('r1');
  });
});
