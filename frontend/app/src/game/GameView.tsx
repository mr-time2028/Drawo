/**
 * GameView — the in-game screen shown while a match is running.
 *
 * Composition:
 *   header   round counter · masked word (or the real word for the drawer) · timer
 *   main     DrawingBoard (drawer gets tools; guessers get a read-only board)
 *   side     players panel (scores, drawer, guessed ticks) + chat/guess feed
 *   overlays countdown · word selection (drawer) / waiting (guessers) ·
 *            round end (word reveal) · leaderboard · game end
 */
import { AlarmClock, Check, Crown, Flag, Palette, RefreshCw, Trophy } from 'lucide-react';
import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { Avatar } from '@/components/ui/Avatar';
import { Button } from '@/components/ui/Button';

import { DrawingBoard } from './DrawingBoard';
import type { useGameChannel } from './useGameChannel';

type Channel = ReturnType<typeof useGameChannel>;

export type GameViewProps = {
  channel: Channel;
  currentUserID: string;
  displayName: string;
  onExit: () => void;
};

function useCountdown(endsAt: number): number {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    if (!endsAt) return;
    const id = setInterval(() => setNow(Math.floor(Date.now() / 1000)), 500);
    return () => clearInterval(id);
  }, [endsAt]);
  if (!endsAt) return 0;
  return Math.max(0, endsAt - now);
}

export function GameView({ channel, currentUserID, displayName, onExit }: GameViewProps) {
  const { t, i18n } = useTranslation();
  const isRTL = i18n.dir() === 'rtl';
  const secondsLeft = useCountdown(channel.endsAt);
  const [guess, setGuess] = useState('');
  const [reportTarget, setReportTarget] = useState<string | null>(null);
  const chatEndRef = useRef<HTMLDivElement | null>(null);

  const isDrawer = channel.drawerID === currentUserID && Boolean(channel.drawerID);
  const me = channel.players.find((p) => p.user_id === currentUserID);
  const isOwner = Boolean(me?.is_owner);
  const state = channel.gameState;
  const isDrawingPhase = state === 'drawing';
  const canDraw = isDrawer && isDrawingPhase;
  const chatDisabled = (isDrawer && isDrawingPhase) || Boolean(me?.guessed_word) || channel.status !== 'open';

  // Auto-scroll chat.
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [channel.chat.length]);

  // Surface socket errors as toasts (rate limits, bad words, …).
  useEffect(() => {
    if (channel.error && channel.status === 'open') {
      toast.error(channel.error);
      channel.clearError();
    }
  }, [channel, channel.error]);

  const playerName = useMemo(() => {
    const map = new Map<string, string>();
    for (const p of channel.players) {
      map.set(p.user_id, p.user_id === currentUserID ? displayName || p.username || '' : (p.username ?? ''));
    }
    return (id?: string) => (id ? map.get(id) || id.slice(0, 8) : '');
  }, [channel.players, currentUserID, displayName]);

  const sortedPlayers = useMemo(
    () => [...channel.players].sort((a, b) => b.score - a.score),
    [channel.players],
  );

  function submitGuess(e: FormEvent) {
    e.preventDefault();
    if (!guess.trim()) return;
    channel.sendChat(guess);
    setGuess('');
  }

  // ------------------------------------------------------------------
  // Word display
  // ------------------------------------------------------------------
  const wordDisplay = (() => {
    if (state === 'round_end' && channel.wordRevealed) {
      return { label: t('game.wordWas', 'The word was'), value: channel.wordRevealed, revealed: true };
    }
    if (isDrawer && channel.myWord) {
      return { label: t('game.yourWord', 'Draw this'), value: channel.myWord, revealed: true };
    }
    if (isDrawingPhase && channel.wordLengths.length > 0) {
      const blanks = channel.wordLengths.map((n) => '_ '.repeat(n).trim()).join('   ');
      return { label: t('game.guessThis', 'Guess the word'), value: blanks, revealed: false };
    }
    return null;
  })();

  // ------------------------------------------------------------------
  // Overlays
  // ------------------------------------------------------------------
  const overlay = (() => {
    switch (state) {
      case 'countdown':
        return (
          <div className="game-overlay">
            <div className="game-overlay-card">
              <AlarmClock size={34} aria-hidden />
              <h3>{t('game.startingSoon', 'Game starting…')}</h3>
              <p className="game-overlay-big">{secondsLeft > 0 ? secondsLeft : '…'}</p>
            </div>
          </div>
        );
      case 'word_selection':
        return (
          <div className="game-overlay">
            <div className="game-overlay-card">
              <Palette size={34} aria-hidden />
              {isDrawer ? (
                <>
                  <h3>{t('game.chooseWord', 'Choose a word to draw')}</h3>
                  <div className="game-word-choices">
                    {channel.suggestions.map((w) => (
                      <button
                        key={w.group_id}
                        type="button"
                        className="game-word-choice"
                        onClick={() => channel.sendChooseWord(w.group_id)}
                      >
                        <span className="game-word-text">{w.text}</span>
                        <span className="game-word-points">
                          {w.points} {t('game.pts', 'pts')}
                        </span>
                      </button>
                    ))}
                  </div>
                  <p className="game-overlay-sub">
                    {t('game.autoPick', 'Auto-picks in {{s}}s', { s: secondsLeft })}
                  </p>
                </>
              ) : (
                <>
                  <h3>
                    {t('game.drawerChoosing', '{{name}} is choosing a word…', {
                      name: playerName(channel.drawerID),
                    })}
                  </h3>
                  <p className="game-overlay-big">{secondsLeft > 0 ? secondsLeft : '…'}</p>
                </>
              )}
            </div>
          </div>
        );
      case 'drawer_disconnected':
        return (
          <div className="game-overlay">
            <div className="game-overlay-card">
              <AlarmClock size={34} aria-hidden />
              <h3>{t('game.drawerDisconnected', 'The drawer disconnected — waiting for them to return…')}</h3>
            </div>
          </div>
        );
      case 'round_end':
        return (
          <div className="game-overlay game-overlay-light">
            <div className="game-overlay-card">
              <h3>{t('game.roundOver', 'Round over!')}</h3>
              {channel.wordRevealed && <p className="game-overlay-word">{channel.wordRevealed}</p>}
            </div>
          </div>
        );
      case 'leaderboard':
      case 'game_end': {
        const final = state === 'game_end';
        return (
          <div className="game-overlay">
            <div className="game-overlay-card game-overlay-wide">
              <Trophy size={34} aria-hidden />
              <h3>
                {final ? t('game.finalResults', 'Final results') : t('game.leaderboard', 'Leaderboard')}
              </h3>
              <ol className="game-final-list">
                {sortedPlayers.map((p, i) => (
                  <li key={p.user_id} className={p.user_id === currentUserID ? 'is-you' : ''}>
                    <span className="game-final-rank">{i + 1}</span>
                    <span className="game-final-name">
                      {playerName(p.user_id)}
                      {p.user_id === currentUserID ? ` (${t('rooms.you', 'You')})` : ''}
                    </span>
                    <span className="game-final-score">{p.score}</span>
                  </li>
                ))}
              </ol>
              {final && (
                <div className="game-final-actions">
                  {isOwner && (
                    <Button type="button" leftIcon={<RefreshCw size={15} />} onClick={channel.sendPlayAgain}>
                      {t('game.playAgain', 'Play again')}
                    </Button>
                  )}
                  <Button type="button" variant={isOwner ? 'secondary' : 'primary'} onClick={onExit}>
                    {t('game.backToDashboard', 'Back to dashboard')}
                  </Button>
                </div>
              )}
            </div>
          </div>
        );
      }
      default:
        return null;
    }
  })();

  return (
    <div className="game-view">
      {/* dir="ltr" pins round (left) · word (center) · timer (right) in
          BOTH languages — labels still translate. */}
      <header className="game-header" dir="ltr">
        <div className="game-round">
          {t('game.round', 'Round')} <strong>{Math.max(1, channel.round)}</strong>
          <span className="game-round-sep">/</span>
          {channel.maxRounds}
        </div>
        {wordDisplay && (
          <div className={`game-word ${wordDisplay.revealed ? 'is-revealed' : 'is-masked'}`}>
            <span className="game-word-label">{wordDisplay.label}</span>
            <span className="game-word-value" dir="auto">
              {wordDisplay.value}
            </span>
          </div>
        )}
        <div className={`game-timer ${secondsLeft > 0 && secondsLeft <= 10 ? 'is-urgent' : ''}`}>
          <AlarmClock size={15} aria-hidden />
          {secondsLeft > 0 ? `${secondsLeft}s` : '—'}
        </div>
      </header>

      {/* dir="ltr" pins the physical order (players | board | chat) so it
          NEVER flips when the UI language toggles between en and fa. Text
          inside still localizes via its own dir attributes. */}
      <div className="game-layout" dir="ltr">
        <aside className="game-players-panel" aria-label={t('rooms.players', 'Players')}>
          <ul className="game-players-list">
            {sortedPlayers.map((p) => (
              <li
                key={p.user_id}
                className={[
                  'game-player',
                  p.user_id === currentUserID ? 'is-you' : '',
                  p.is_drawer ? 'is-drawer' : '',
                  !p.is_online ? 'is-offline' : '',
                  p.guessed_word ? 'has-guessed' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
              >
                {/* Fixed row order: avatar → name → score → report flag */}
                <Avatar size="sm" alt={playerName(p.user_id)} />
                <span className="game-player-name" dir="auto">
                  {playerName(p.user_id)}
                  {p.is_owner && <Crown size={11} aria-label={t('rooms.owner', 'Owner')} />}
                  {p.is_drawer && <Palette size={11} aria-label={t('game.drawer', 'Drawer')} />}
                  {p.guessed_word && <Check size={12} className="game-player-check" aria-hidden />}
                </span>
                <span className="game-player-score">{p.score}</span>
                {p.user_id !== currentUserID && p.is_online ? (
                  <button
                    type="button"
                    className="game-player-report"
                    title={t('game.report', 'Report player')}
                    aria-label={t('game.report', 'Report player')}
                    onClick={() => setReportTarget(p.user_id)}
                  >
                    <Flag size={12} />
                  </button>
                ) : (
                  <span className="game-player-report-spacer" aria-hidden />
                )}
              </li>
            ))}
          </ul>
        </aside>

        <div className="game-board-wrap">
          <DrawingBoard engine={channel.engine} canDraw={canDraw} onSend={channel.sendDraw} />
          {overlay}
        </div>

        <section className="game-chat" aria-label={t('game.chatTitle', 'Chat & guesses')}>
          {/* Chat CONTENT aligns with the language: fa → right, en → left.
              The section itself stays docked on the right of the layout. */}
          <div className="game-chat-feed" dir={isRTL ? 'rtl' : 'ltr'}>
            {channel.chat.map((m) => (
              <p
                key={m.key}
                className={`game-chat-msg ${m.system ? 'is-system' : ''} ${
                  m.user_id === currentUserID ? 'is-you' : ''
                }`}
              >
                {m.system ? (
                  m.message
                ) : (
                  <>
                    <strong dir="auto">{playerName(m.user_id)}:</strong> <span dir="auto">{m.text}</span>
                  </>
                )}
              </p>
            ))}
            <div ref={chatEndRef} />
          </div>
          <form className="game-chat-input" onSubmit={submitGuess} dir={isRTL ? 'rtl' : 'ltr'}>
            <input
              value={guess}
              onChange={(e) => setGuess(e.target.value)}
              placeholder={
                isDrawer && isDrawingPhase
                  ? t('game.drawerNoChat', 'Drawers cannot chat')
                  : me?.guessed_word
                    ? t('game.alreadyGuessed', 'You guessed it!')
                    : t('game.guessPlaceholder', 'Type your guess…')
              }
              disabled={chatDisabled}
              maxLength={120}
              aria-label={t('game.guessPlaceholder', 'Type your guess…')}
            />
            <Button type="submit" size="sm" disabled={chatDisabled || !guess.trim()}>
              {t('game.send', 'Send')}
            </Button>
          </form>
        </section>
      </div>

      {reportTarget && (
        <div className="game-overlay" role="dialog" aria-modal="true">
          <div className="game-overlay-card">
            <Flag size={26} aria-hidden />
            <h3>{t('game.reportTitle', 'Report {{name}}', { name: playerName(reportTarget) })}</h3>
            <div className="game-report-reasons">
              {(
                [
                  ['inappropriate_drawing', t('game.reportDrawing', 'Inappropriate drawing')],
                  ['abusive_chat', t('game.reportChat', 'Abusive chat')],
                  ['cheating', t('game.reportCheating', 'Cheating')],
                  ['griefing', t('game.reportGriefing', 'Griefing')],
                ] as const
              ).map(([reason, label]) => (
                <button
                  key={reason}
                  type="button"
                  className="game-report-reason"
                  onClick={() => {
                    channel.sendReport(reportTarget, reason);
                    setReportTarget(null);
                    toast.success(t('game.reportSent', 'Report sent.'));
                  }}
                >
                  {label}
                </button>
              ))}
            </div>
            <Button type="button" variant="ghost" size="sm" onClick={() => setReportTarget(null)}>
              {t('common.cancel', 'Cancel')}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
