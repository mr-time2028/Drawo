import { Link, useNavigate, useParams } from '@tanstack/react-router';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { apiRequest, ApiError } from '@/api/http';
import { getRoomByCode, joinRoomByCode, type Room } from '@/api/room';
import { Button } from '@/components/ui/Button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input, Label } from '@/components/ui/Input';
import { useAuthStore } from '@/stores/authStore';

type RoomStatus = 'loading' | 'ready' | 'error' | 'joining';

const MIN_NICK = 2;
const MAX_NICK = 20;

export function InviteLandingPage() {
  const { t } = useTranslation();
  const { code } = useParams({ strict: false });
  const navigate = useNavigate();
  const accessToken = useAuthStore((state) => state.accessToken);

  const [room, setRoom] = useState<Record<string, unknown> | Room | null>(null);
  const [status, setStatus] = useState<RoomStatus>(code ? 'loading' : 'error');
  const [error, setError] = useState<string>(code ? '' : t('rooms.invalidCode', 'Invalid invite link.'));
  const [password, setPassword] = useState('');
  const [nickname, setNickname] = useState('');
  const [nicknameError, setNicknameError] = useState('');
  const [needPassword, setNeedPassword] = useState(false);
  const mounted = useRef(false);
  const autoJoinTried = useRef(false);

  // Reset per-mount refs under React 18 StrictMode (dev-only double-mount).
  // Without this, the simulated unmount/remount leaves refs as "already tried"
  // and the real mount skips fetches / auto-join.
  useEffect(() => {
    autoJoinTried.current = false;
  }, [code]);

  useEffect(() => {
    if (!code) return;
    let cancelled = false;
    const upperCode = code.toUpperCase();
    const fetchRoom = (skipAuth = false) =>
      skipAuth
        ? // Force-skip auth so a stale/expired Bearer token cannot 401 and prevent
          // a public invite page from rendering (e.g. when a user opens an
          // invite link in a fresh browser profile with garbage in localStorage,
          // or after server-restarted sessions).
          apiRequest<Room>(`/api/v1/rooms/by-code/${encodeURIComponent(upperCode)}`, { accessToken: null })
        : getRoomByCode(upperCode);

    // Try fetching WITHOUT sending any stored credentials first. The endpoint
    // is fully public (the backend doesn't care who asks for room metadata),
    // so skipping auth avoids 401→refresh→navigate-away side effects from a
    // stale localStorage token. A 401 response is impossible against a public
    // endpoint without an Authorization header, so we don't need a retry path.
    fetchRoom(true)
      .then((r) => {
        if (cancelled) return;
        const fetched = r as Room & { has_password?: boolean };
        if (fetched.has_password) {
          setNeedPassword(true);
        }
        setRoom(fetched);
        setStatus('ready');
      })
      .catch((err) => {
        if (cancelled) return;
        setStatus('error');
        setError(err instanceof Error ? err.message : t('errors.fallbackError'));
      });
    return () => {
      cancelled = true;
    };
    // NOTE: we intentionally do NOT use a module-level mounted ref to guard
    // this effect. React 18 StrictMode in dev mounts→unmounts→remounts every
    // effect once; a useRef guard that persists across the simulated remount
    // causes the second (real) mount to skip the fetch entirely, leaving the
    // page stuck on "Loading…" forever. The `cancelled` local is per-effect
    // and correctly handles each mount/unmount cycle.
  }, [code, t]);

  // Auto-join when the user is already logged in, the room is loaded, in
  // lobby state, and no password is required.
  useEffect(() => {
    if (!room || !accessToken || !code) return;
    if (status !== 'ready') return;
    if (autoJoinTried.current) return;
    const r = room as Room & { has_password?: boolean };
    const state = (r.state as string) || 'lobby';
    if (state !== 'lobby') return;
    if (r.has_password) {
      autoJoinTried.current = true;
      return;
    }
    autoJoinTried.current = true;
    void handleJoin();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [room, accessToken, status, code]);

  async function handleJoin() {
    if (!code) return;

    // Anonymous users must pick a nickname before joining.
    if (!accessToken) {
      const nick = nickname.trim();
      if (nick.length < MIN_NICK) {
        setNicknameError(t('rooms.nicknameTooShort', 'Nickname must be at least 2 characters.'));
        return;
      }
      if (nick.length > MAX_NICK) {
        setNicknameError(t('rooms.nicknameTooLong', 'Nickname must be at most 20 characters.'));
        return;
      }
      setNicknameError('');
    }

    setStatus('joining');
    try {
      const joined = await joinRoomByCode(code.toUpperCase(), {
        password: needPassword ? password : undefined,
        nickname: accessToken ? undefined : nickname.trim(),
      });
      toast.success(t('rooms.joined', 'Joined the room!'));
      // Guest sessions land on the same lobby route; the lobby will detect the
      // guest token from localStorage and use it for WebSocket auth.
      void navigate({ to: `/rooms/${joined.id}`, replace: true });
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('errors.fallbackError');
      if (/password/i.test(msg)) {
        setNeedPassword(true);
        setStatus('ready');
        toast.error(msg);
        return;
      }
      setStatus('ready');
      toast.error(msg);
    }
  }

  if (status === 'loading') {
    return (
      <div className="room-invite-page">
        <Card padding="lg" className="room-invite-card">
          <p className="room-invite-muted">{t('common.loading', 'Loading…')}</p>
        </Card>
      </div>
    );
  }

  if (status === 'error' || !room) {
    return (
      <div className="room-invite-page">
        <Card padding="lg" className="room-invite-card">
          <CardHeader>
            <CardTitle>{t('rooms.invalidTitle', 'Invalid invite')}</CardTitle>
            <CardDescription>{error}</CardDescription>
          </CardHeader>
          <Link to="/app" className="room-invite-home">
            <Button>{t('rooms.backToDashboard', 'Back to dashboard')}</Button>
          </Link>
        </Card>
      </div>
    );
  }

  const r = room as Room & Record<string, unknown>;
  const wordSourceLabel =
    r.word_source === 'custom'
      ? t('rooms.customWords', 'Custom categories & words')
      : t('rooms.drawoWords', 'Drawo curated words');
  const stateLabel = (r.state as string) || 'lobby';
  const isJoinable = stateLabel === 'lobby';

  return (
    <div className="room-invite-page">
      <Card padding="lg" className="room-invite-card">
        <CardHeader>
          <div>
            <CardTitle>{r.name as string}</CardTitle>
            <CardDescription>
              {t('rooms.inviteDescription', 'You were invited to a private Drawo room.')}
            </CardDescription>
          </div>
          <span className={`room-state-badge state-${stateLabel}`}>{stateLabel}</span>
        </CardHeader>

        <dl className="room-meta-grid">
          <div className="room-meta-item">
            <dt>{t('dashboard.privateRoom.maxPlayers', 'Max players')}</dt>
            <dd>{r.max_players as number}</dd>
          </div>
          <div className="room-meta-item">
            <dt>{t('dashboard.privateRoom.rounds', 'Rounds')}</dt>
            <dd>{r.max_rounds as number}</dd>
          </div>
          <div className="room-meta-item">
            <dt>{t('dashboard.privateRoom.drawTime', 'Draw time')}</dt>
            <dd>
              {r.round_time as number}
              {t('rooms.secondsUnit', 's')}
            </dd>
          </div>
          <div className="room-meta-item">
            <dt>{t('rooms.language', 'Language')}</dt>
            <dd>{(r.language as string)?.toUpperCase()}</dd>
          </div>
          <div className="room-meta-item room-meta-wide">
            <dt>{t('rooms.words', 'Words')}</dt>
            <dd>{wordSourceLabel}</dd>
          </div>
        </dl>

        {needPassword && isJoinable && (
          <div className="room-password-field">
            <Label htmlFor="room-password">{t('dashboard.privateRoom.passwordLabel', 'Password')}</Label>
            <Input
              id="room-password"
              type="text"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('rooms.passwordPlaceholder', 'Enter room password')}
            />
          </div>
        )}

        {!accessToken && isJoinable && (
          <div className="room-password-field">
            <Label htmlFor="guest-nickname">{t('rooms.nicknameLabel', 'Your nickname')}</Label>
            <Input
              id="guest-nickname"
              type="text"
              value={nickname}
              maxLength={MAX_NICK}
              onChange={(e) => {
                setNickname(e.target.value);
                if (nicknameError) setNicknameError('');
              }}
              placeholder={t('rooms.nicknamePlaceholder', 'Pick a name to join as a guest')}
            />
            {nicknameError ? <p className="room-field-error">{nicknameError}</p> : null}
            <p className="room-field-hint">
              {t(
                'rooms.guestHint',
                'You are joining as a guest. Guests are bound to this room and cannot chat or play in other rooms.',
              )}
            </p>
          </div>
        )}

        <div className="room-invite-actions">
          {isJoinable ? (
            <Button
              onClick={handleJoin}
              disabled={status === 'joining'}
              loading={status === 'joining'}
              type="button"
            >
              {accessToken ? t('rooms.joinRoom', 'Join room') : t('rooms.joinAsGuest', 'Join as guest')}
            </Button>
          ) : (
            <Button disabled>{t('rooms.gameInProgress', 'Game in progress')}</Button>
          )}
          {!accessToken && isJoinable && (
            <Link to="/login" search={{ next: `/r/${code}` }}>
              <Button variant="secondary" type="button">
                {t('rooms.loginToJoin', 'Log in to join')}
              </Button>
            </Link>
          )}
          <Link to={accessToken ? '/app' : '/'}>
            <Button variant="ghost" type="button">
              {accessToken
                ? t('rooms.backToDashboard', 'Back to dashboard')
                : t('rooms.backHome', 'Back to home')}
            </Button>
          </Link>
        </div>
      </Card>
    </div>
  );
}
