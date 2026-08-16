import { Globe, Plus, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { parseWordList } from '@/api/room';
import { Button } from '@/components/ui/Button';
import { Input, Label } from '@/components/ui/Input';
import { Modal } from '@/components/ui/Modal';
import { cn } from '@/utils/cn';
import { isSupportedLanguage, type SupportedLanguage } from '@/i18n';

type WordTier = 1 | 2 | 3;

type CustomCategory = {
  id: string; // local id for React keys
  name: string;
  words: Record<WordTier, string>; // textarea content per tier
};

type PrivateRoomData = {
  name: string;
  password: string;
  language: SupportedLanguage;
  min_players: number;
  max_players: number;
  max_rounds: number;
  round_time: number;
  word_source: 'default' | 'custom';
  custom_categories: CustomCategory[];
};

type PrivateRoomModalProps = {
  open: boolean;
  onClose: () => void;
  onCreate: (
    data: Omit<PrivateRoomData, 'custom_categories'> & {
      custom_categories: { name: string; words: Record<number, string[]> }[];
    },
  ) => void;
  loading?: boolean;
  /** Game language chosen in the Start Match drawer (controls word bank &
   *  system messages). Stored separately from the UI/site language. */
  initialLanguage?: SupportedLanguage;
};

const PLAYERS_MIN = 2;
const PLAYERS_MAX = 12;
const PLAYERS_MIN_DEFAULT = 2;
const PLAYERS_MAX_DEFAULT = 8;
const ROUNDS_MIN = 1;
const ROUNDS_MAX = 10;
const ROUNDS_DEFAULT = 3;
const TIME_MIN = 30;
const TIME_MAX = 180;
const TIME_STEP = 10;
const TIME_DEFAULT = 80;
const MIN_TOTAL_WORDS = 5;
const PASSWORD_MIN = 4;
const PASSWORD_MAX = 32;

function newCategory(id: string): CustomCategory {
  return { id, name: '', words: { 1: '', 2: '', 3: '' } };
}

function countTotalWords(cats: CustomCategory[]): number {
  let n = 0;
  for (const c of cats) {
    for (const tier of [1, 2, 3] as WordTier[]) {
      n += parseWordList(c.words[tier]).length;
    }
  }
  return n;
}

export function PrivateRoomModal({
  open,
  onClose,
  onCreate,
  loading,
  initialLanguage = 'fa',
}: PrivateRoomModalProps) {
  const { t } = useTranslation();
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [minPlayers, setMinPlayers] = useState<number>(PLAYERS_MIN_DEFAULT);
  const [maxPlayers, setMaxPlayers] = useState<number>(PLAYERS_MAX_DEFAULT);
  const [maxRounds, setMaxRounds] = useState<number>(ROUNDS_DEFAULT);
  const [roundTime, setRoundTime] = useState<number>(TIME_DEFAULT);
  // Game language: defaults to the language selected in the Start Match drawer.
  // When custom words are used, the word bank language is irrelevant (owner's
  // words are used directly); system/game-flow messages still follow this.
  const [lang, setLang] = useState<SupportedLanguage>(
    isSupportedLanguage(initialLanguage) ? initialLanguage : 'fa',
  );
  const [wordSource, setWordSource] = useState<'default' | 'custom'>('default');
  const [categories, setCategories] = useState<CustomCategory[]>([newCategory('c1')]);

  const totalWords = useMemo(() => countTotalWords(categories), [categories]);
  const trimmedPassword = password.trim();
  const passwordError = useMemo(() => {
    if (trimmedPassword.length === 0) return '';
    if (trimmedPassword.length < PASSWORD_MIN) {
      return t('dashboard.privateRoom.passwordTooShort', 'Password must be at least {{min}} characters.', {
        min: PASSWORD_MIN,
      });
    }
    if (trimmedPassword.length > PASSWORD_MAX) {
      return t('dashboard.privateRoom.passwordTooLong', 'Password must be at most {{max}} characters.', {
        max: PASSWORD_MAX,
      });
    }
    return '';
  }, [trimmedPassword, t]);
  // Keep min ≤ max.
  useEffect(() => {
    if (minPlayers > maxPlayers) setMinPlayers(maxPlayers);
    if (maxPlayers < minPlayers) setMaxPlayers(minPlayers);
  }, [minPlayers, maxPlayers]);

  const canCreate = useMemo(() => {
    if (!name.trim() || name.trim().length < 3) return false;
    if (passwordError) return false;
    if (minPlayers < PLAYERS_MIN || minPlayers > PLAYERS_MAX) return false;
    if (maxPlayers < minPlayers || maxPlayers > PLAYERS_MAX) return false;
    if (maxRounds < ROUNDS_MIN || maxRounds > ROUNDS_MAX) return false;
    if (roundTime < TIME_MIN || roundTime > TIME_MAX) return false;
    if (wordSource === 'custom') {
      const named = categories.filter((c) => c.name.trim().length > 0);
      if (named.length === 0) return false;
      if (totalWords < MIN_TOTAL_WORDS) return false;
    }
    return true;
  }, [name, passwordError, minPlayers, maxPlayers, maxRounds, roundTime, wordSource, categories, totalWords]);

  function addCategory() {
    setCategories((cs) => [...cs, newCategory('c' + (cs.length + 1) + '_' + Date.now())]);
  }
  function removeCategory(id: string) {
    setCategories((cs) => cs.filter((c) => c.id !== id));
  }
  function updateCategory(id: string, patch: Partial<CustomCategory>) {
    setCategories((cs) => cs.map((c) => (c.id === id ? { ...c, ...patch } : c)));
  }
  function updateTier(id: string, tier: WordTier, value: string) {
    setCategories((cs) => cs.map((c) => (c.id === id ? { ...c, words: { ...c.words, [tier]: value } } : c)));
  }

  function reset() {
    setName('');
    setPassword('');
    setMinPlayers(PLAYERS_MIN_DEFAULT);
    setMaxPlayers(PLAYERS_MAX_DEFAULT);
    setMaxRounds(ROUNDS_DEFAULT);
    setRoundTime(TIME_DEFAULT);
    setLang(isSupportedLanguage(initialLanguage) ? initialLanguage : 'fa');
    setWordSource('default');
    setCategories([newCategory('c1')]);
  }

  // Sync incoming initial language (e.g. user changed it in the drawer and
  // re-opened the modal during the same session).
  useEffect(() => {
    if (isSupportedLanguage(initialLanguage)) setLang(initialLanguage);
  }, [initialLanguage]);

  function handleCreate() {
    const custom = categories
      .filter((c) => c.name.trim().length > 0)
      .map((c) => ({
        name: c.name.trim(),
        words: {
          1: parseWordList(c.words[1]),
          2: parseWordList(c.words[2]),
          3: parseWordList(c.words[3]),
        },
      }))
      .filter((c) => c.words[1].length + c.words[2].length + c.words[3].length > 0);

    onCreate({
      name: name.trim(),
      password: password.trim(),
      language: lang,
      min_players: minPlayers,
      max_players: maxPlayers,
      max_rounds: maxRounds,
      round_time: roundTime,
      word_source: wordSource,
      custom_categories: wordSource === 'custom' ? custom : [],
    });
    reset();
  }

  function handleClose() {
    reset();
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={t('dashboard.privateRoom.title', 'Create private room')}
      description={t(
        'dashboard.privateRoom.subtitle',
        'Configure your game and share the invite link with friends.',
      )}
      className="private-room-modal"
      footer={
        <div className="private-room-footer">
          <Button variant="outline" onClick={handleClose} type="button" className="private-footer-cancel">
            {t('common.cancel', 'Cancel')}
          </Button>
          <Button
            onClick={handleCreate}
            loading={loading}
            disabled={!canCreate || loading}
            type="button"
            size="lg"
          >
            {t('dashboard.privateRoom.create', 'Create room')}
          </Button>
        </div>
      }
    >
      <div className="private-room-form">
        <div className="private-field">
          <Label htmlFor="room-name">{t('dashboard.privateRoom.nameLabel', 'Room name')}</Label>
          <Input
            id="room-name"
            value={name}
            maxLength={50}
            placeholder={t('dashboard.privateRoom.namePlaceholder', "Hamid's room")}
            onChange={(e) => setName(e.target.value)}
          />
        </div>

        <div className="private-field">
          <Label>
            <Globe
              size={15}
              strokeWidth={2.3}
              aria-hidden="true"
              style={{ display: 'inline', verticalAlign: '-2px', marginInlineEnd: 4 }}
            />
            {t('dashboard.privateRoom.language', 'Game language')}
          </Label>
          <p className="field-hint">
            {wordSource === 'custom'
              ? t(
                  'dashboard.privateRoom.languageHintCustom',
                  'With custom words the word bank is your own; system messages and game flow still follow this language.',
                )
              : t(
                  'dashboard.privateRoom.languageHint',
                  'Determines the word bank and in-game system messages. Independent of the site UI language.',
                )}
          </p>
          <div
            className="start-match-lang-grid"
            role="radiogroup"
            aria-label={t('dashboard.privateRoom.language', 'Game language')}
          >
            <button
              type="button"
              role="radio"
              aria-checked={lang === 'en'}
              className={cn('start-match-lang-btn', lang === 'en' && 'is-active')}
              onClick={() => setLang('en')}
            >
              <span className="start-match-lang-flag" aria-hidden="true">
                EN
              </span>
              <span>English</span>
            </button>
            <button
              type="button"
              role="radio"
              aria-checked={lang === 'fa'}
              className={cn('start-match-lang-btn', lang === 'fa' && 'is-active')}
              onClick={() => setLang('fa')}
            >
              <span className="start-match-lang-flag" aria-hidden="true">
                فا
              </span>
              <span>فارسی</span>
            </button>
          </div>
        </div>

        <div className="private-field">
          <Label htmlFor="room-password">
            {t('dashboard.privateRoom.passwordLabel', 'Password (optional)')}
          </Label>
          <Input
            id="room-password"
            type="text"
            value={password}
            maxLength={PASSWORD_MAX}
            placeholder={t('dashboard.privateRoom.passwordPlaceholder', 'Leave empty for an open room')}
            onChange={(e) => setPassword(e.target.value)}
          />
          <p className={cn('field-hint', passwordError && 'field-error')}>
            {passwordError ||
              t(
                'dashboard.privateRoom.passwordHint',
                'If set, anyone joining via the invite link must enter this password. Leave empty for an open room.',
              )}
          </p>
        </div>

        <div className="private-grid-2">
          <NumberField
            label={t('dashboard.privateRoom.minPlayers', 'Min players')}
            value={minPlayers}
            min={PLAYERS_MIN}
            max={Math.min(maxPlayers, PLAYERS_MAX)}
            defaultValue={PLAYERS_MIN_DEFAULT}
            onChange={setMinPlayers}
          />
          <NumberField
            label={t('dashboard.privateRoom.maxPlayers', 'Max players')}
            value={maxPlayers}
            min={Math.max(minPlayers, PLAYERS_MIN)}
            max={PLAYERS_MAX}
            defaultValue={PLAYERS_MAX_DEFAULT}
            onChange={setMaxPlayers}
          />
        </div>

        <div className="private-grid-2">
          <NumberField
            label={t('dashboard.privateRoom.rounds', 'Rounds')}
            value={maxRounds}
            min={ROUNDS_MIN}
            max={ROUNDS_MAX}
            defaultValue={ROUNDS_DEFAULT}
            onChange={setMaxRounds}
          />
          <NumberField
            label={t('dashboard.privateRoom.drawTime', 'Draw time (seconds)')}
            value={roundTime}
            min={TIME_MIN}
            max={TIME_MAX}
            step={TIME_STEP}
            defaultValue={TIME_DEFAULT}
            onChange={setRoundTime}
          />
        </div>

        <div className="private-field">
          <Label>{t('dashboard.privateRoom.wordSourceLabel', 'Words')}</Label>
          <div className="word-source-grid">
            <button
              type="button"
              className={cn('word-source-btn', wordSource === 'default' && 'is-active')}
              onClick={() => setWordSource('default')}
            >
              <span className="word-source-name">
                {t('dashboard.privateRoom.useDefault', 'Use Drawo words')}
              </span>
              <span className="word-source-desc">
                {t(
                  'dashboard.privateRoom.useDefaultDesc',
                  'Our curated categories, words, and point tiers — same as public matches.',
                )}
              </span>
            </button>
            <button
              type="button"
              className={cn('word-source-btn', wordSource === 'custom' && 'is-active')}
              onClick={() => setWordSource('custom')}
            >
              <span className="word-source-name">
                {t('dashboard.privateRoom.useCustom', 'My own categories & words')}
              </span>
              <span className="word-source-desc">
                {t(
                  'dashboard.privateRoom.useCustomDesc',
                  'Type your own categories and words per difficulty tier (easy/medium/hard).',
                )}
              </span>
            </button>
          </div>
        </div>

        {wordSource === 'custom' && (
          <div className="private-field custom-words-panel">
            <div className="custom-words-header">
              <div>
                <Label>{t('dashboard.privateRoom.customCategoriesLabel', 'Custom categories')}</Label>
                <p className="field-hint">
                  {t(
                    'dashboard.privateRoom.customHint',
                    'Create at least one category. Add words to the difficulty buckets you want; empty buckets are skipped. Easy = 1 point, Medium = 2 points, Hard = 3 points.',
                  )}
                </p>
              </div>
              <Button
                variant="secondary"
                size="sm"
                type="button"
                leftIcon={<Plus size={15} />}
                onClick={addCategory}
              >
                {t('dashboard.privateRoom.addCategory', 'Add category')}
              </Button>
            </div>

            <div className="custom-cats-list">
              {categories.map((cat, idx) => (
                <div className="custom-cat" key={cat.id}>
                  <div className="custom-cat-head">
                    <Input
                      value={cat.name}
                      placeholder={t('dashboard.privateRoom.categoryNamePlaceholder', 'Category name')}
                      onChange={(e) => updateCategory(cat.id, { name: e.target.value })}
                    />
                    <button
                      type="button"
                      className="custom-cat-remove"
                      onClick={() => removeCategory(cat.id)}
                      aria-label={t('dashboard.privateRoom.removeCategory', 'Remove category')}
                      disabled={categories.length <= 1}
                    >
                      <Trash2 size={16} strokeWidth={2.2} aria-hidden />
                    </button>
                  </div>
                  <div className="custom-tiers-grid">
                    {([1, 2, 3] as WordTier[]).map((tier) => (
                      <div className="custom-tier" key={tier}>
                        <div className={cn('custom-tier-label', `tier-${tier}`)}>
                          {t(`dashboard.privateRoom.tier${tier}`, `Tier ${tier} (${tier}pt)`)}
                        </div>
                        <textarea
                          className="custom-tier-input"
                          value={cat.words[tier]}
                          rows={3}
                          placeholder={t('dashboard.privateRoom.wordPlaceholder', 'One word per line')}
                          onChange={(e) => updateTier(cat.id, tier, e.target.value)}
                        />
                        <div className="custom-tier-count">{parseWordList(cat.words[tier]).length}</div>
                      </div>
                    ))}
                  </div>
                  {idx < categories.length - 1 && <div className="custom-cat-divider" />}
                </div>
              ))}
            </div>

            <div className={cn('custom-words-counter', totalWords >= MIN_TOTAL_WORDS && 'is-ok')}>
              {t('dashboard.privateRoom.wordCount', { count: totalWords, min: MIN_TOTAL_WORDS })}
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
}

function NumberField({
  label,
  value,
  min,
  max,
  step = 1,
  defaultValue,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  defaultValue: number;
  onChange: (n: number) => void;
}) {
  function clamp(n: number) {
    if (Number.isNaN(n)) return defaultValue;
    return Math.max(min, Math.min(max, n));
  }
  return (
    <div className="private-field">
      <Label>{label}</Label>
      <div className="number-field">
        <button
          type="button"
          className="number-btn"
          onClick={() => onChange(clamp(value - step))}
          aria-label="−"
        >
          −
        </button>
        <input
          type="number"
          className="number-input"
          value={value}
          min={min}
          max={max}
          step={step}
          onChange={(e) => onChange(clamp(Number(e.target.value)))}
        />
        <button
          type="button"
          className="number-btn"
          onClick={() => onChange(clamp(value + step))}
          aria-label="+"
        >
          +
        </button>
        <button
          type="button"
          className="number-reset"
          onClick={() => onChange(defaultValue)}
          title="Reset to default"
        >
          ↺
        </button>
      </div>
    </div>
  );
}
