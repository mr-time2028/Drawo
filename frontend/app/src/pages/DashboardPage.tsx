import { useNavigate } from '@tanstack/react-router';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { logout as apiLogout } from '@/api/auth';
import { wsReleaseToken } from '@/api/authTokenManager';
import { getProfile, type UserProfile } from '@/api/user';
import { buildInviteURL, createRoom } from '@/api/room';
import { HistoryModal } from '@/components/dashboard/HistoryModal';
import { OverviewSection } from '@/components/dashboard/OverviewSection';
import { PrivateRoomModal } from '@/components/dashboard/PrivateRoomModal';
import { ProfileSection } from '@/components/dashboard/ProfileSection';
import { RecoverySection } from '@/components/dashboard/RecoverySection';
import { Sidebar, type DashboardSection } from '@/components/dashboard/Sidebar';
import { StartMatchDrawer } from '@/components/dashboard/StartMatchDrawer';
import { isSupportedLanguage, type SupportedLanguage } from '@/i18n';
import { useAuthStore } from '@/stores/authStore';

export function DashboardPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const accessToken = useAuthStore((state) => state.accessToken);
  const clearTokens = useAuthStore((state) => state.clearTokens);

  const [collapsed, setCollapsed] = useState(false);
  const [section, setSection] = useState<DashboardSection>('overview');
  const [historyOpen, setHistoryOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [privateOpen, setPrivateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [logoutLoading, setLogoutLoading] = useState(false);
  const [fabRect, setFabRect] = useState<DOMRect | null>(null);
  const fabRef = useRef<HTMLButtonElement | null>(null);
  // Game language is picked in the Start Match drawer (independent of UI
  // language) and carried through to PrivateRoomModal + the createRoom API.
  const [gameLanguage, setGameLanguage] = useState<SupportedLanguage>(() => {
    const v = typeof window !== 'undefined' ? localStorage.getItem('drawo.gameLanguage') : null;
    return isSupportedLanguage(v) ? v : 'fa';
  });

  const openStartDrawer = useCallback(() => {
    if (fabRef.current) setFabRect(fabRef.current.getBoundingClientRect());
    setDrawerOpen(true);
  }, []);

  const [data, setData] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    getProfile()
      .then((p) => {
        if (cancelled) return;
        // Reconcile locale with backend (source of truth). If server differs
        // from our cached localStorage value, switch to it.
        if (isSupportedLanguage(p.profile.locale) && i18n.language !== p.profile.locale) {
          void i18n.changeLanguage(p.profile.locale);
        }
        setData(p);
      })
      .catch((err) => {
        if (cancelled) return;
        console.error('[dashboard] profile load failed:', err);
        const msg = err instanceof Error ? err.message : t('errors.fallbackError');
        toast.error(msg);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [t, i18n]);

  const username = useMemo(() => data?.user.username ?? '', [data]);
  const profile = data?.profile;
  const isRTL = i18n.dir() === 'rtl';

  async function handleLogout() {
    setLogoutLoading(true);
    try {
      if (accessToken) {
        await apiLogout();
      }
    } catch (err) {
      console.warn('[dashboard] backend logout failed:', err);
      toast.error(t('auth.logoutBackendFailed'));
    } finally {
      wsReleaseToken();
      clearTokens();
      setLogoutLoading(false);
      void navigate({ to: '/login', replace: true });
    }
  }

  async function handleCreateRoom(input: {
    name: string;
    password: string;
    language: SupportedLanguage;
    min_players: number;
    max_players: number;
    max_rounds: number;
    round_time: number;
    word_source: 'default' | 'custom';
    custom_categories: { name: string; words: Record<number, string[]> }[];
  }) {
    setCreating(true);
    try {
      const lang = isSupportedLanguage(input.language) ? input.language : gameLanguage;
      // Persist the user's chosen game language so the drawer remembers it
      // next time they open it. This is SEPARATE from i18n/site language.
      localStorage.setItem('drawo.gameLanguage', lang);
      setGameLanguage(lang);
      const room = await createRoom({
        name: input.name,
        password: input.password || undefined,
        language: lang,
        min_players: input.min_players,
        max_players: input.max_players,
        max_rounds: input.max_rounds,
        round_time: input.round_time,
        word_source: input.word_source,
        room_type: 'private',
        custom_categories: input.custom_categories,
      });
      const url = buildInviteURL(room.invite_code);
      try {
        await navigator.clipboard.writeText(url);
        toast.success(t('dashboard.privateRoom.createdAndCopied', { url }));
      } catch {
        toast.success(t('dashboard.privateRoom.created', { code: room.invite_code }));
      }
      setPrivateOpen(false);
      void navigate({ to: `/rooms/${room.id}`, replace: false });
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('errors.fallbackError');
      toast.error(msg);
    } finally {
      setCreating(false);
    }
  }

  return (
    <main className="dashboard-shell-page" aria-busy={loading}>
      <Sidebar
        collapsed={collapsed}
        onToggleCollapsed={() => setCollapsed((c) => !c)}
        active={section}
        onSelect={setSection}
        username={username}
        avatarUrl={profile?.avatar_url}
        onLogout={handleLogout}
        logoutLoading={logoutLoading}
      />

      <div className="dashboard-content">
        {loading || !data || !profile ? (
          <div className="dashboard-loading">
            <span className="dashboard-spinner" aria-hidden="true" />
            <p>{t('common.loading', 'Loading…')}</p>
          </div>
        ) : (
          <>
            {section === 'overview' && (
              <OverviewSection
                username={username}
                gamesPlayed={profile.games_played}
                mvps={profile.mvps}
                wordScore={profile.word_score}
                reputationScore={profile.reputation_score}
                rank={profile.rank}
                onShowHistory={() => setHistoryOpen(true)}
              />
            )}
            {section === 'recovery' && (
              <RecoverySection
                profile={profile}
                onProfileUpdated={(up) => setData((prev) => (prev ? { ...prev, profile: up } : prev))}
              />
            )}
            {section === 'profile' && (
              <ProfileSection
                profile={profile}
                username={username}
                onProfileUpdated={(up) => setData((prev) => (prev ? { ...prev, profile: up } : prev))}
                onUsernameUpdated={(u) =>
                  setData((prev) => (prev ? { ...prev, user: { ...prev.user, username: u } } : prev))
                }
              />
            )}
          </>
        )}

        <button
          ref={fabRef}
          type="button"
          className="start-match-button"
          onClick={openStartDrawer}
          title={t('dashboard.startMatch')}
        >
          <span className="start-match-label">{t('dashboard.startMatch')}</span>
        </button>
      </div>

      <HistoryModal open={historyOpen} onClose={() => setHistoryOpen(false)} />

      <StartMatchDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        onStartPublic={(lang) => {
          setGameLanguage(lang);
          localStorage.setItem('drawo.gameLanguage', lang);
          toast.info(t('dashboard.startMatch.publicSoon', 'Public matchmaking is coming soon!'));
        }}
        onOpenPrivate={(lang) => {
          setGameLanguage(lang);
          localStorage.setItem('drawo.gameLanguage', lang);
          setPrivateOpen(true);
        }}
        anchorSide={isRTL ? 'start' : 'end'}
        anchorRect={fabRect}
      />

      <PrivateRoomModal
        open={privateOpen}
        onClose={() => setPrivateOpen(false)}
        onCreate={handleCreateRoom}
        loading={creating}
        initialLanguage={gameLanguage}
      />
    </main>
  );
}
