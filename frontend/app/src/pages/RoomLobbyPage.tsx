import { useNavigate, useParams } from '@tanstack/react-router';
import { Copy, Crown, LogOut, Play, Settings, Trash2, Wifi, WifiOff } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { buildInviteURL, closeRoom, getRoom, leaveRoom, type Room } from '@/api/room';
import { readGuestSession, clearGuestSession } from '@/api/guestTokenManager';
import { getProfile } from '@/api/user';
import { Avatar } from '@/components/ui/Avatar';
import { Button } from '@/components/ui/Button';
import { Card, CardDescription, CardTitle } from '@/components/ui/Card';
import { GameView } from '@/game/GameView';
import { useGameChannel } from '@/game/useGameChannel';

export function RoomLobbyPage() {
  const { t } = useTranslation();
  const { roomId } = useParams({ strict: false });
  const navigate = useNavigate();
  const [room, setRoom] = useState<Room | null>(null);
  const [currentUserID, setCurrentUserID] = useState<string>('');
  const [displayName, setDisplayName] = useState<string>('');
  const [isGuest, setIsGuest] = useState(false);
  const [loading, setLoading] = useState(true);
  const [starting, setStarting] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const [closing, setClosing] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      if (!roomId) {
        setLoading(false);
        return;
      }
      setLoading(true);
      try {
        const g = readGuestSession();
        const profilePromise = getProfile().catch(() => null);
        const [roomRes, profileRes] = await Promise.all([getRoom(roomId), profilePromise]);
        if (cancelled) return;
        setRoom(roomRes);
        if (profileRes) {
          setCurrentUserID(profileRes.user.id);
          setDisplayName(profileRes.user.username);
          setIsGuest(false);
        } else if (g && g.roomID === roomId) {
          setCurrentUserID(g.guestID);
          setDisplayName(g.nickname);
          setIsGuest(true);
        } else {
          void navigate({ to: '/', replace: true });
          return;
        }
      } catch (err) {
        if (cancelled) return;
        toast.error(err instanceof Error ? err.message : t('errors.fallbackError'));
        void navigate({ to: '/', replace: true });
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [roomId, navigate, t]);

  // Open the WebSocket once we have the room metadata. For private rooms the
  // backend expects either room_id or invite_code in the join payload; we send
  // room_id (faster, one less lookup on the server). The channel also carries
  // the full in-game state (canvas ops, chat, word events) so the same socket
  // serves both the lobby AND the game screen with no reconnect between them.
  const socket = useGameChannel(room?.id, room?.invite_code);

  // When the server reports an error, surface it as a toast.
  useEffect(() => {
    if (socket.error && socket.status === 'error') {
      toast.error(socket.error);
    }
  }, [socket.error, socket.status]);

  const inviteURL = room ? buildInviteURL(room.invite_code) : '';
  const isOwner = Boolean(room && currentUserID && room.owner_id === currentUserID);

  // Merge the REST-fetched room state with the live socket state. In the lobby
  // we trust REST until the socket has opened and sent a game_state; after that
  // we switch to live players/counts so join/leave events are reflected.
  const players =
    socket.players.length > 0
      ? socket.players
      : [
          {
            user_id: currentUserID,
            username: displayName,
            score: 0,
            is_drawer: false,
            is_online: true,
            is_owner: isOwner,
          },
        ];
  const onlineCount = players.filter((p) => p.is_online).length;
  // Backend game_state values: waiting_for_players, countdown, word_selection,
  // drawing, round_end, leaderboard, drawer_disconnected, game_end.
  // REST room.state uses: lobby, playing, finished.
  const restState = room?.state ?? 'lobby';
  const socketLive = socket.status === 'open' && socket.gameState;
  const gameState = socketLive
    ? socket.gameState
    : restState === 'playing'
      ? 'countdown'
      : 'waiting_for_players';
  const isInLobby = gameState === 'waiting_for_players';
  const isPlaying = [
    'countdown',
    'word_selection',
    'drawing',
    'round_end',
    'leaderboard',
    'drawer_disconnected',
  ].includes(gameState);
  // Guests cannot reconnect (a new invite join is a new identity). Hide any
  // leftover offline guest row. Logged-in players stay visible as offline
  // while we wait for them to come back.
  const visiblePlayers = players.filter((p) => {
    const guest = Boolean(p.is_guest || p.user_id.startsWith('guest:'));
    return p.is_online || !guest;
  });

  async function copyLink() {
    if (!inviteURL) return;
    try {
      await navigator.clipboard.writeText(inviteURL);
      setCopied(true);
      toast.success(t('rooms.linkCopied', 'Invite link copied!'));
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error(t('rooms.copyFailed', 'Could not copy link automatically.'));
    }
  }

  function handleStart() {
    if (!room) return;
    if (socket.status !== 'open') {
      toast.error(t('rooms.connecting', 'Connecting…'));
      return;
    }
    setStarting(true);
    // The owner starts the countdown over the WebSocket. The hub transitions
    // the room through countdown → word_selection → drawing on its own and
    // broadcasts a fresh game_state as soon as it does.
    socket.sendStart();
    // Give the server one tick to flip state, then clear the loading flag.
    // Any error frames will surface via the socket's error toast effect.
    setTimeout(() => setStarting(false), 500);
  }

  async function handleLeave() {
    if (!room) return;
    setLeaving(true);
    try {
      // Tell the room to drop this seat immediately (not "offline / reconnect").
      socket.sendLeave();
      if (isGuest) {
        clearGuestSession();
        toast.success(t('rooms.left', 'You left the room.'));
        void navigate({ to: '/', replace: true });
        return;
      }
      await leaveRoom(room.id);
      toast.success(t('rooms.left', 'You left the room.'));
      void navigate({ to: '/app', replace: true });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('errors.fallbackError'));
    } finally {
      setLeaving(false);
    }
  }

  async function handleClose() {
    if (!room) return;
    setClosing(true);
    try {
      await closeRoom(room.id);
      toast.success(t('rooms.closed', 'Room closed.'));
      void navigate({ to: '/app', replace: true });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t('errors.fallbackError'));
    } finally {
      setClosing(false);
    }
  }

  if (loading || !room) {
    return (
      <div className="room-lobby-page">
        <div className="room-lobby-loading">{t('common.loading', 'Loading…')}</div>
      </div>
    );
  }

  // Once the match is live (any in-game state), swap the lobby for the game
  // screen. The same socket/channel keeps running underneath, so the canvas
  // has already received canvas_sync and no ops are lost in the transition.
  if (socket.status === 'open' && (isPlaying || gameState === 'game_end')) {
    return (
      <GameView
        channel={socket}
        currentUserID={currentUserID}
        displayName={displayName}
        onExit={() => {
          socket.sendLeave();
          if (isGuest) {
            clearGuestSession();
            void navigate({ to: '/', replace: true });
          } else {
            void navigate({ to: '/app', replace: true });
          }
        }}
      />
    );
  }

  const wordSourceLabel =
    room.word_source === 'custom'
      ? t('rooms.customWords', 'Custom categories & words')
      : t('rooms.drawoWords', 'Drawo curated words');

  const connectionStatusLabel =
    socket.status === 'open'
      ? t('rooms.connected', 'Connected')
      : socket.status === 'connecting'
        ? t('rooms.connecting', 'Connecting…')
        : socket.status === 'closed'
          ? t('rooms.disconnected', 'Disconnected')
          : t('rooms.connectionError', 'Connection error');

  return (
    <div className="room-lobby-page">
      <div className="room-lobby-inner">
        <Card padding="none" className="room-lobby-card">
          <header className="room-lobby-header">
            <div className="room-lobby-heading">
              <CardTitle className="room-lobby-title">{room.name}</CardTitle>
              <CardDescription className="room-lobby-meta">
                <span className="room-lobby-kind">
                  {room.type === 'private'
                    ? t('rooms.privateRoom', 'Private room')
                    : t('rooms.publicRoom', 'Public room')}
                </span>
                <span className={`room-state-badge state-${isInLobby ? 'lobby' : gameState}`}>
                  {isInLobby ? t('rooms.lobby', 'lobby') : gameState}
                </span>
                <span className="room-connection-status" data-status={socket.status}>
                  {socket.status === 'open' ? (
                    <Wifi size={12} aria-hidden />
                  ) : (
                    <WifiOff size={12} aria-hidden />
                  )}
                  {connectionStatusLabel}
                </span>
              </CardDescription>
            </div>
            {isOwner && (
              <button
                type="button"
                className="room-lobby-close"
                onClick={handleClose}
                disabled={closing}
                aria-label={t('rooms.closeRoom', 'Close room')}
              >
                <Trash2 size={15} />
              </button>
            )}
          </header>

          <div className="room-invite-bar">
            <div className="room-invite-url" title={inviteURL}>
              {inviteURL}
            </div>
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<Copy size={14} />}
              onClick={copyLink}
              type="button"
              className={copied ? 'room-invite-copy is-copied' : 'room-invite-copy'}
            >
              {copied ? t('rooms.copied', 'Copied') : t('rooms.copyLink', 'Copy link')}
            </Button>
          </div>

          <div className="room-lobby-grid">
            <section className="room-section room-section-settings">
              <h3 className="room-section-title">
                <Settings size={15} aria-hidden />
                {t('rooms.settingsTitle', 'Game settings')}
              </h3>
              <dl className="room-meta-grid">
                <div className="room-meta-item">
                  <dt>{t('dashboard.privateRoom.maxPlayers', 'Max players')}</dt>
                  <dd>{room.max_players}</dd>
                </div>
                <div className="room-meta-item">
                  <dt>{t('dashboard.privateRoom.rounds', 'Rounds')}</dt>
                  <dd>{room.max_rounds}</dd>
                </div>
                <div className="room-meta-item">
                  <dt>{t('dashboard.privateRoom.drawTime', 'Draw time')}</dt>
                  <dd>
                    {room.round_time}
                    {t('rooms.secondsUnit', 's')}
                  </dd>
                </div>
                <div className="room-meta-item">
                  <dt>{t('rooms.language', 'Language')}</dt>
                  <dd>{room.language.toUpperCase()}</dd>
                </div>
                <div className="room-meta-item room-meta-wide">
                  <dt>{t('rooms.words', 'Words')}</dt>
                  <dd>{wordSourceLabel}</dd>
                </div>
              </dl>
            </section>

            <section className="room-section room-section-players">
              <h3 className="room-section-title">
                <Crown size={15} aria-hidden />
                {t('rooms.players', 'Players')}
                <span className="room-player-count">
                  {onlineCount}/{socket.maxPlayers || room.max_players}
                </span>
              </h3>
              <ul className="room-players-list" tabIndex={players.length > 5 ? 0 : undefined}>
                {players.map((p) => {
                  const isMe = p.user_id === currentUserID;
                  const isGuestPlayer = p.is_guest || p.user_id.startsWith('guest:') || (isMe && isGuest);
                  const name = isMe
                    ? displayName || p.username || t('rooms.you', 'You')
                    : p.username || p.user_id.slice(0, 8);
                  const role = p.is_owner
                    ? t('rooms.owner', 'Owner')
                    : isGuestPlayer
                      ? t('rooms.guest', 'Guest')
                      : t('rooms.player', 'Player');
                  return (
                    <li
                      className={[
                        'room-player',
                        isMe ? 'is-you' : '',
                        p.is_owner ? 'is-owner' : '',
                        !p.is_online ? 'is-offline' : '',
                      ]
                        .filter(Boolean)
                        .join(' ')}
                      key={p.user_id}
                    >
                      <Avatar size="sm" alt={name} className="room-player-avatar" />
                      <div className="room-player-info">
                        <p className="room-player-name">
                          {name}
                          {isMe ? <span className="room-player-you">{t('rooms.you', 'You')}</span> : null}
                        </p>
                        <p className="room-player-sub">
                          {role}
                          {!p.is_online ? ` · ${t('rooms.offline', 'offline')}` : ''}
                        </p>
                      </div>
                      {p.is_owner && (
                        <span className="room-owner-badge" title={t('rooms.owner', 'Owner')}>
                          <Crown size={12} />
                        </span>
                      )}
                    </li>
                  );
                })}
              </ul>
              {visiblePlayers.length < (socket.minPlayers || 2) && (
                <p className="room-empty-hint">
                  {t('rooms.waitingForPlayers', 'Share the invite link to add more players.')}
                </p>
              )}
            </section>
          </div>

          <footer className="room-lobby-actions">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleLeave}
              leftIcon={<LogOut size={15} />}
              type="button"
              loading={leaving}
              disabled={leaving || closing || starting}
              className="room-lobby-leave"
            >
              {t('rooms.leave', 'Leave room')}
            </Button>
            {isOwner ? (
              !isInLobby ? (
                <Button disabled variant="ghost" size="sm" className="room-lobby-status">
                  {isPlaying
                    ? t('rooms.gameInProgressState', 'Game in progress')
                    : t('rooms.roomUnavailable', 'Room unavailable')}
                </Button>
              ) : (
                <Button
                  size="sm"
                  onClick={handleStart}
                  loading={starting}
                  leftIcon={<Play size={15} />}
                  type="button"
                  disabled={leaving || closing || onlineCount < (socket.minPlayers || 2)}
                  className="room-lobby-start"
                >
                  {t('dashboard.startMatch', 'Start Match')}
                </Button>
              )
            ) : (
              <Button disabled variant="ghost" size="sm" className="room-lobby-status">
                {isPlaying
                  ? t('rooms.gameInProgressState', 'Game in progress')
                  : t('rooms.waitingForOwner', 'Waiting for owner to start…')}
              </Button>
            )}
          </footer>
        </Card>
      </div>
    </div>
  );
}
