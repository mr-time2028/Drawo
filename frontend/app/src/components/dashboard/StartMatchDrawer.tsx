import { Gamepad2, Globe, Lock, Users, X } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui/Button';
import { cn } from '@/utils/cn';
import { isSupportedLanguage, type SupportedLanguage } from '@/i18n';

// Key under which the user's chosen GAME language is persisted in localStorage.
// This is separate from the site/UI language (i18n) — the two can be changed
// independently: UI language controls menu/header text, game language controls
// the word bank/system prompts used in a match.
const GAME_LANG_KEY = 'drawo.gameLanguage';

function loadGameLanguage(): SupportedLanguage {
  if (typeof window === 'undefined') return 'fa';
  const v = window.localStorage.getItem(GAME_LANG_KEY);
  return isSupportedLanguage(v) ? v : 'fa';
}

function persistGameLanguage(lang: SupportedLanguage) {
  try {
    window.localStorage.setItem(GAME_LANG_KEY, lang);
  } catch {
    /* ignore quota / storage-disabled errors */
  }
}

type MatchDrawerProps = {
  open: boolean;
  onClose: () => void;
  onStartPublic: (lang: SupportedLanguage) => void;
  onOpenPrivate: (lang: SupportedLanguage) => void;
  anchorSide: 'end' | 'start';
  // DOMRect of the button that opened the drawer — used to anchor the popup
  // directly above the Start Match FAB.
  anchorRect?: DOMRect | null;
};

type MatchChoice = 'public' | 'private' | null;

export function StartMatchDrawer({
  open,
  onClose,
  onStartPublic,
  onOpenPrivate,
  anchorSide,
  anchorRect,
}: MatchDrawerProps) {
  const { t } = useTranslation();
  const containerRef = useRef<HTMLDivElement | null>(null);

  // Game language is INDEPENDENT from the UI/site language (i18n). It controls
  // which word bank/system language is used during the match. It starts from
  // localStorage and toggles instantly — no side effects on i18n.
  const [lang, setLang] = useState<SupportedLanguage>(() => loadGameLanguage());
  const [choice, setChoice] = useState<MatchChoice>('private');

  // Reset choice when the drawer closes but keep remembered game language.
  useEffect(() => {
    if (!open) setChoice('private');
  }, [open]);

  const isFa = lang === 'fa';

  // Close on Escape — stopPropagation so that when the drawer is open it
  // doesn't accidentally close OTHER modals mounted later on the page
  // (for example a parent modal listening on the window). Also guard so we
  // don't react if a nested modal (e.g. PrivateRoomModal) is on top — in
  // that case the drawer is closed anyway when the private modal is opened
  // via handleCtaClick, but we keep the guard as defense-in-depth.
  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  function chooseLanguage(next: SupportedLanguage) {
    setLang(next);
    persistGameLanguage(next);
  }

  // Position the popup so it sits directly above the Start Match button,
  // clamped to the viewport. Anchor rect is provided by the parent; we fall
  // back to the viewport corner if unavailable.
  const popupStyle = useMemo<React.CSSProperties>(() => {
    const margin = 16;
    const popupWidth = 380;
    const popupHeight = 560; // max height via CSS; we only use it for bottom clamp
    if (!anchorRect) {
      return anchorSide === 'end'
        ? { insetInlineEnd: margin, insetBlockEnd: margin }
        : { insetInlineStart: margin, insetBlockEnd: margin };
    }
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    // Horizontally align to the button's center, then clamp within viewport.
    const btnCenterX = anchorRect.left + anchorRect.width / 2;
    let left: number;
    if (anchorSide === 'end') {
      // Align the popup's right edge with the button's right edge in LTR.
      left = anchorRect.right - popupWidth;
    } else {
      left = anchorRect.left;
    }
    left = Math.max(margin, Math.min(left, vw - popupWidth - margin));
    const bottom = Math.max(margin, vh - anchorRect.top + 12); // 12px gap above button
    return {
      position: 'fixed',
      left,
      bottom: Math.min(bottom, vh - margin),
      width: popupWidth,
      maxHeight: Math.min(popupHeight, vh - bottom - margin),
    };
  }, [anchorRect, anchorSide, open]);

  function handleCtaClick() {
    if (choice === 'public') {
      onStartPublic(lang);
      onClose();
    } else if (choice === 'private') {
      onOpenPrivate(lang);
      onClose();
    }
  }

  return (
    <div
      ref={containerRef}
      className={cn(
        'start-match-drawer-root',
        open && 'is-open',
      )}
      aria-hidden={!open}
    >
      <div className="start-match-backdrop" onClick={onClose} aria-hidden="true" />
      <div
        className="start-match-drawer"
        role="dialog"
        aria-modal="true"
        aria-label={t('dashboard.startMatch')}
        style={open ? popupStyle : undefined}
      >
        <div className="start-match-drawer-header">
          <div className="start-match-drawer-title">
            <Gamepad2 size={20} strokeWidth={2.3} aria-hidden="true" />
            <span>{t('dashboard.startMatch.title', 'Start a match')}</span>
          </div>
          <button
            type="button"
            className="start-match-drawer-close"
            onClick={onClose}
            aria-label={t('common.close', 'Close')}
          >
            <X size={18} strokeWidth={2.4} aria-hidden="true" />
          </button>
        </div>

        <div className="start-match-drawer-body">
          {/* Game language */}
          <section className="start-match-section">
            <h4 className="start-match-section-title">
              <Globe size={16} strokeWidth={2.3} aria-hidden="true" />
              {t('dashboard.startMatch.language', 'Game language')}
            </h4>
            <p className="start-match-section-hint">
              {t(
                'dashboard.startMatch.languageHint',
                'Words, prompts and the chat filter will use this language.',
              )}
            </p>
            <div
              className="start-match-lang-grid"
              role="radiogroup"
              aria-label={t('dashboard.startMatch.language')}
            >
              <button
                type="button"
                role="radio"
                aria-checked={!isFa}
                className={cn('start-match-lang-btn', !isFa && 'is-active')}
                onClick={() => chooseLanguage('en')}
              >
                <span className="start-match-lang-flag" aria-hidden>
                  EN
                </span>
                <span>English</span>
              </button>
              <button
                type="button"
                role="radio"
                aria-checked={isFa}
                className={cn('start-match-lang-btn', isFa && 'is-active')}
                onClick={() => chooseLanguage('fa')}
              >
                <span className="start-match-lang-flag" aria-hidden>
                  فا
                </span>
                <span>فارسی</span>
              </button>
            </div>
          </section>

          {/* Game type */}
          <section className="start-match-section">
            <h4 className="start-match-section-title">
              <Users size={16} strokeWidth={2.3} aria-hidden="true" />
              {t('dashboard.startMatch.gameType', 'Game type')}
            </h4>
            <p className="start-match-section-hint">
              {t(
                'dashboard.startMatch.gameTypeHint',
                'Play with anyone in a public queue, or set up a private room with friends.',
              )}
            </p>
            <div className="start-match-type-grid">
              <button
                type="button"
                className={cn('start-match-type-btn', choice === 'public' && 'is-active')}
                onClick={() => setChoice('public')}
                disabled
                aria-disabled="true"
                title={t('dashboard.startMatch.publicSoon')}
              >
                <span className="start-match-type-icon" aria-hidden>
                  <Users size={22} strokeWidth={2.2} />
                </span>
                <span className="start-match-type-name">
                  {t('dashboard.startMatch.public', 'Public match')}
                </span>
                <span className="start-match-type-desc">
                  {t('dashboard.startMatch.publicDesc', 'Matchmaking is coming soon')}
                </span>
                <span className="start-match-soon-badge">{t('dashboard.startMatch.soon', 'Soon')}</span>
              </button>
              <button
                type="button"
                className={cn('start-match-type-btn', choice === 'private' && 'is-active')}
                onClick={() => setChoice('private')}
              >
                <span className="start-match-type-icon" aria-hidden>
                  <Lock size={22} strokeWidth={2.2} />
                </span>
                <span className="start-match-type-name">
                  {t('dashboard.startMatch.private', 'Private room')}
                </span>
                <span className="start-match-type-desc">
                  {t(
                    'dashboard.startMatch.privateDesc',
                    'Create a room, share the link, and play with friends.',
                  )}
                </span>
              </button>
            </div>
          </section>
        </div>

        <div className="start-match-drawer-footer">
          <Button variant="outline" onClick={onClose} type="button" fullWidth>
            {t('common.cancel', 'Cancel')}
          </Button>
          <Button
            onClick={handleCtaClick}
            type="button"
            size="lg"
            fullWidth
            disabled={choice === null || choice === 'public'}
          >
            {choice === 'public'
              ? t('dashboard.startMatch.startPublic', 'Find a match')
              : t('dashboard.privateRoom.create', 'Create room')}
          </Button>
        </div>
      </div>
    </div>
  );
}
