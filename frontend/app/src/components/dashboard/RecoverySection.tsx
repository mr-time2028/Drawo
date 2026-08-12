import { CheckCircle2, Loader2, Mail, Phone, Send, type LucideIcon } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { Button } from '@/components/ui/Button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input, Label } from '@/components/ui/Input';
import { cn } from '@/utils/cn';

import {
  confirmVerification,
  getProfile,
  requestVerification,
  updateProfile,
  type UserProfile,
  type VerifyType,
} from '@/api/user';

type ContactRowProps = {
  type: VerifyType;
  icon: LucideIcon;
  label: string;
  placeholder: string;
  initialValue: string;
  verified: boolean;
  onSaved: (next: string) => void;
};

function ContactRow({
  type,
  icon: Icon,
  label,
  placeholder,
  initialValue,
  verified,
  onSaved,
}: ContactRowProps) {
  const { t } = useTranslation();
  const [value, setValue] = useState(initialValue);
  const [requesting, setRequesting] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [saving, setSaving] = useState(false);
  const [codeOpen, setCodeOpen] = useState(false);
  const [code, setCode] = useState('');
  const trimmed = value.trim();
  const dirty = trimmed !== initialValue;

  async function handleSave() {
    if (!dirty || !trimmed) return;
    setSaving(true);
    try {
      const patch = type === 'email' ? { email: trimmed } : { phone: trimmed };
      await updateProfile(patch);
      toast.success(t('dashboard.recovery.saved'));
      onSaved(trimmed);
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('errors.fallbackError');
      toast.error(msg);
    } finally {
      setSaving(false);
    }
  }

  async function handleRequest() {
    setRequesting(true);
    try {
      await requestVerification(type);
      toast.success(t('dashboard.recovery.codeSent'));
      setCodeOpen(true);
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('errors.fallbackError');
      toast.error(msg);
    } finally {
      setRequesting(false);
    }
  }

  async function handleConfirm() {
    if (code.length !== 6) return;
    setConfirming(true);
    try {
      await confirmVerification(type, code);
      toast.success(t('dashboard.recovery.verified'));
      setCodeOpen(false);
      setCode('');
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('errors.fallbackError');
      toast.error(msg);
    } finally {
      setConfirming(false);
    }
  }

  return (
    <div className="recovery-contact">
      <div className="recovery-contact-row">
        <span className="recovery-contact-icon" aria-hidden="true">
          <Icon size={20} strokeWidth={2.2} />
        </span>
        <div className="recovery-contact-field">
          <Label htmlFor={`recovery-${type}`} className="recovery-contact-label">
            {label}
          </Label>
          <div className="recovery-contact-input">
            <Input
              id={`recovery-${type}`}
              type={type === 'email' ? 'email' : 'tel'}
              placeholder={placeholder}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              inputMode={type === 'phone' ? 'tel' : 'email'}
            />
            {verified && !dirty && (
              <span className="recovery-verified-badge" aria-label={t('dashboard.recovery.verified')}>
                <CheckCircle2 size={16} strokeWidth={2.4} aria-hidden="true" />
              </span>
            )}
          </div>
        </div>
        <div className="recovery-contact-actions">
          <Button
            variant="secondary"
            size="sm"
            onClick={handleSave}
            disabled={saving || !dirty || !trimmed}
            loading={saving}
            type="button"
          >
            {t('dashboard.recovery.save')}
          </Button>
          <Button
            variant={verified ? 'outline' : 'primary'}
            size="sm"
            onClick={handleRequest}
            disabled={requesting || dirty || !trimmed || (verified && dirty)}
            loading={requesting}
            leftIcon={<Send size={14} strokeWidth={2.4} />}
            type="button"
          >
            {verified ? t('dashboard.recovery.resendCode') : t('dashboard.recovery.verify')}
          </Button>
        </div>
      </div>
      {codeOpen && (
        <div className="recovery-code-row">
          <Input
            placeholder={t('dashboard.recovery.codePlaceholder')}
            value={code}
            onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
            maxLength={6}
            inputMode="numeric"
            className="recovery-code-input"
          />
          <Button
            size="sm"
            onClick={handleConfirm}
            disabled={code.length !== 6 || confirming}
            loading={confirming}
            type="button"
          >
            {t('dashboard.recovery.confirm')}
          </Button>
          <button
            type="button"
            className="recovery-code-cancel"
            onClick={() => {
              setCodeOpen(false);
              setCode('');
            }}
          >
            {t('common.cancel', 'Cancel')}
          </button>
        </div>
      )}
    </div>
  );
}

type RecoveryProps = {
  profile: UserProfile['profile'];
  onProfileUpdated: (p: UserProfile['profile']) => void;
};

export function RecoverySection({ profile, onProfileUpdated }: RecoveryProps) {
  const { t } = useTranslation();
  // Use a counter as key for the contact rows so their local state resets
  // whenever we get a fresh profile from the server (e.g. after Save).
  const [emailKey, setEmailKey] = useState(0);
  const [phoneKey, setPhoneKey] = useState(0);

  return (
    <div className="dashboard-section recovery-section">
      <Card padding="lg" className="recovery-card">
        <CardHeader>
          <div>
            <CardTitle>{t('dashboard.recovery.title')}</CardTitle>
            <CardDescription>{t('dashboard.recovery.subtitle')}</CardDescription>
          </div>
        </CardHeader>
        <div className="recovery-list">
          <ContactRow
            key={`email-${emailKey}-${profile.email}`}
            type="email"
            icon={Mail}
            label={t('dashboard.recovery.emailLabel')}
            placeholder={t('dashboard.recovery.emailPlaceholder')}
            initialValue={profile.email ?? ''}
            verified={profile.email_verified}
            onSaved={() => {
              // Refetch latest by signalling parent; bump key to reset draft.
              void refetchProfile(onProfileUpdated);
              setEmailKey((k) => k + 1);
            }}
          />
          <ContactRow
            key={`phone-${phoneKey}-${profile.phone}`}
            type="phone"
            icon={Phone}
            label={t('dashboard.recovery.phoneLabel')}
            placeholder={t('dashboard.recovery.phonePlaceholder')}
            initialValue={profile.phone ?? ''}
            verified={profile.phone_verified}
            onSaved={() => {
              void refetchProfile(onProfileUpdated);
              setPhoneKey((k) => k + 1);
            }}
          />
        </div>
        <p className={cn('recovery-hint')}>
          <span className="recovery-hint-icon" aria-hidden="true">
            <Loader2 size={14} />
          </span>
          {t('dashboard.recovery.hint')}
        </p>
      </Card>
    </div>
  );
}

// Best-effort refetch of the latest profile after a save. Errors are swallowed
// here because the parent already holds the optimistic value via onSaved, and
// a future refresh will reconcile the state.
async function refetchProfile(onProfileUpdated: (p: UserProfile['profile']) => void) {
  try {
    const fresh = await getProfile();
    onProfileUpdated(fresh.profile);
  } catch {
    /* ignore */
  }
}
