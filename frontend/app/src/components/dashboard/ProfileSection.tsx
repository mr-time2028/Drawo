import { Camera, Check, Edit3, Volume2, VolumeX } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { Avatar } from '@/components/ui/Avatar';
import { Button } from '@/components/ui/Button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input, Label } from '@/components/ui/Input';
import { cn } from '@/utils/cn';

import { changeUsername, getProfile, updateProfile, type UserProfile } from '@/api/user';

type ProfileProps = {
  profile: UserProfile['profile'];
  username: string;
  onProfileUpdated: (p: UserProfile['profile']) => void;
  onUsernameUpdated: (u: string) => void;
};

function UsernameEditor({ username, onSaved }: { username: string; onSaved: (u: string) => void }) {
  const { t } = useTranslation();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(username);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    const next = draft.trim();
    if (!next || next === username) {
      setEditing(false);
      return;
    }
    setSaving(true);
    try {
      await changeUsername({ username: next });
      onSaved(next);
      toast.success(t('dashboard.profile.usernameUpdated'));
      setEditing(false);
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('errors.fallbackError');
      toast.error(msg);
    } finally {
      setSaving(false);
    }
  }

  if (!editing) {
    return (
      <div className="profile-name-view">
        <h3 className="profile-name">{username}</h3>
        <button
          type="button"
          className="profile-name-edit"
          onClick={() => {
            setDraft(username);
            setEditing(true);
          }}
          aria-label={t('dashboard.profile.changeUsername')}
        >
          <Edit3 size={15} strokeWidth={2.4} aria-hidden="true" />
        </button>
      </div>
    );
  }

  return (
    <div className="profile-name-edit-row">
      <Input
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        maxLength={20}
        aria-label={t('auth.username')}
      />
      <Button
        size="sm"
        onClick={handleSave}
        loading={saving}
        disabled={!draft.trim() || draft.trim() === username}
        leftIcon={<Check size={15} strokeWidth={2.4} />}
        type="button"
      >
        {t('dashboard.profile.save')}
      </Button>
      <Button size="sm" variant="ghost" onClick={() => setEditing(false)} type="button">
        {t('common.cancel', 'Cancel')}
      </Button>
    </div>
  );
}

function AvatarEditor({ initialUrl, onSaved }: { initialUrl: string; onSaved: (url: string) => void }) {
  const { t } = useTranslation();
  const [draft, setDraft] = useState(initialUrl);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    const url = draft.trim();
    if (url === initialUrl) return;
    setSaving(true);
    try {
      const updated = await updateProfile({ avatar_url: url });
      onSaved(url);
      toast.success(t('dashboard.profile.avatarUpdated'));
      if (updated.avatar_url !== url) {
        setDraft(updated.avatar_url ?? '');
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('errors.fallbackError');
      toast.error(msg);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="profile-field">
      <Label htmlFor="profile-avatar-url">{t('dashboard.profile.avatarUrlLabel')}</Label>
      <div className="profile-field-row">
        <Input
          id="profile-avatar-url"
          placeholder={t('dashboard.profile.avatarUrlPlaceholder')}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
        />
        <Button
          size="sm"
          onClick={handleSave}
          loading={saving}
          disabled={draft.trim() === initialUrl}
          type="button"
        >
          {t('dashboard.profile.save')}
        </Button>
      </div>
      <p className="profile-field-hint">{t('dashboard.profile.avatarUrlHint')}</p>
    </div>
  );
}

function SoundToggles({
  initialBg,
  initialTool,
  onSaved,
}: {
  initialBg: boolean;
  initialTool: boolean;
  onSaved: (p: { bg: boolean; tool: boolean }) => void;
}) {
  const { t } = useTranslation();
  const [bg, setBg] = useState(initialBg);
  const [tool, setTool] = useState(initialTool);
  const [saving, setSaving] = useState(false);

  async function toggle(which: 'bg' | 'tool') {
    const nextBg = which === 'bg' ? !bg : bg;
    const nextTool = which === 'tool' ? !tool : tool;
    setBg(nextBg);
    setTool(nextTool);
    setSaving(true);
    try {
      const updated = await updateProfile({
        background_sound: nextBg,
        tool_sound: nextTool,
      });
      onSaved({ bg: updated.background_sound, tool: updated.tool_sound });
    } catch (err) {
      setBg(initialBg);
      setTool(initialTool);
      const msg = err instanceof Error ? err.message : t('errors.fallbackError');
      toast.error(msg);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="profile-toggles">
      <SoundToggle
        label={t('dashboard.profile.backgroundSound')}
        active={bg}
        onToggle={() => toggle('bg')}
        disabled={saving}
      />
      <SoundToggle
        label={t('dashboard.profile.toolSound')}
        active={tool}
        onToggle={() => toggle('tool')}
        disabled={saving}
      />
    </div>
  );
}

export function ProfileSection({ profile, username, onProfileUpdated, onUsernameUpdated }: ProfileProps) {
  const { t } = useTranslation();

  return (
    <div className="dashboard-section profile-section">
      <Card padding="lg" className="profile-card">
        <CardHeader>
          <div>
            <CardTitle>{t('dashboard.profile.title')}</CardTitle>
            <CardDescription>{t('dashboard.profile.subtitle')}</CardDescription>
          </div>
        </CardHeader>

        <div className="profile-hero">
          <div className="profile-avatar-wrap">
            <Avatar
              size="xl"
              src={profile.avatar_url}
              alt={username}
              fallbackName={username}
              className="profile-avatar"
            />
            <span className="profile-avatar-badge" aria-hidden="true">
              <Camera size={14} strokeWidth={2.4} />
            </span>
          </div>

          <div className="profile-name-block">
            <UsernameEditor key={`name-${username}`} username={username} onSaved={onUsernameUpdated} />
            <p className="profile-member-since">
              {t('dashboard.profile.memberSince', {
                date: new Date(profile.created_at).toLocaleDateString(),
              })}
            </p>
          </div>
        </div>

        <div className="profile-fields">
          <AvatarEditor
            key={`avatar-${profile.avatar_url ?? ''}`}
            initialUrl={profile.avatar_url ?? ''}
            onSaved={() => refetchProfile(onProfileUpdated)}
          />
        </div>
      </Card>

      <Card padding="lg" className="settings-card">
        <CardHeader>
          <div>
            <CardTitle>{t('dashboard.settings.title')}</CardTitle>
            <CardDescription>{t('dashboard.settings.subtitle')}</CardDescription>
          </div>
        </CardHeader>
        <SoundToggles
          key={`toggles-${profile.background_sound}-${profile.tool_sound}`}
          initialBg={profile.background_sound}
          initialTool={profile.tool_sound}
          onSaved={({ bg, tool: toolVal }) => {
            onProfileUpdated({
              ...profile,
              background_sound: bg,
              tool_sound: toolVal,
            });
            refetchProfile(onProfileUpdated);
          }}
        />
      </Card>
    </div>
  );
}

function SoundToggle({
  label,
  active,
  onToggle,
  disabled,
}: {
  label: string;
  active: boolean;
  onToggle: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={active}
      onClick={onToggle}
      disabled={disabled}
      className={cn('sound-toggle', active && 'is-active')}
    >
      <span className="sound-toggle-icon" aria-hidden="true">
        {active ? <Volume2 size={20} strokeWidth={2.2} /> : <VolumeX size={20} strokeWidth={2.2} />}
      </span>
      <span className="sound-toggle-label">{label}</span>
      <span className="sound-toggle-track" aria-hidden>
        <span className="sound-toggle-thumb" />
      </span>
    </button>
  );
}

async function refetchProfile(onProfileUpdated: (p: UserProfile['profile']) => void) {
  try {
    const fresh = await getProfile();
    onProfileUpdated(fresh.profile);
  } catch {
    /* ignore */
  }
}
