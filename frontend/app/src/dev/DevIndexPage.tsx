import { Copy, Palette, Terminal } from 'lucide-react';
import { toast } from 'sonner';
import { useTranslation } from 'react-i18next';

import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input, Label, Textarea } from '@/components/ui/Input';
import { Skeleton } from '@/components/ui/Skeleton';
import { Spinner } from '@/components/ui/Spinner';
import { Avatar } from '@/components/ui/Avatar';
import { mockMe } from '@/mock';

/**
 * Design-system preview page. This route only exists in development so we can
 * eyeball every primitive in one place during Phase 0.
 */
export function DevIndexPage() {
  const { i18n } = useTranslation();
  const isFa = i18n.language === 'fa';

  return (
    <main
      className="dev-page"
      style={{
        padding: '6rem 1.5rem 3rem',
        maxWidth: 1100,
        marginInline: 'auto',
        display: 'grid',
        gap: '1.5rem',
      }}
    >
      <header style={{ display: 'grid', gap: '0.5rem' }}>
        <Badge variant="default" style={{ width: 'fit-content' }}>
          <Terminal size={14} />
          {isFa ? 'صفحه توسعه' : 'Dev playground'}
        </Badge>
        <h1 style={{ fontSize: '2.4rem', fontWeight: 900, color: 'var(--ink)' }}>
          {isFa ? 'پیش‌نمایش طراحی' : 'Design system preview'}
        </h1>
        <p style={{ color: 'var(--muted)', maxWidth: 520 }}>
          {isFa
            ? 'این صفحه فقط در حالت توسعه در دسترس است و برای بررسی کامپوننت‌های پایه استفاده می‌شود.'
            : 'This page only exists in development. Preview primitives, theme, language, and toasts here.'}
        </p>
      </header>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{isFa ? 'دکمه‌ها' : 'Buttons'}</CardTitle>
            <CardDescription>{isFa ? 'انواع دکمه' : 'Variant / size / loading combinations'}</CardDescription>
          </div>
        </CardHeader>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.75rem' }}>
          <Button variant="primary">Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="danger">Danger</Button>
          <Button loading>Loading</Button>
          <Button size="sm">Small</Button>
          <Button size="lg">Large</Button>
          <Button variant="primary" onClick={() => toast.success(isFa ? 'سلام!' : 'Hello there 👋')}>
            {isFa ? 'نمایش اعلان' : 'Show toast'}
          </Button>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{isFa ? 'نشان‌ها' : 'Badges'}</CardTitle>
            <CardDescription>{isFa ? 'وضعیت‌ها' : 'Status badges'}</CardDescription>
          </div>
        </CardHeader>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
          <Badge>Default</Badge>
          <Badge variant="success">Success</Badge>
          <Badge variant="danger">Danger</Badge>
          <Badge variant="warning">Warning</Badge>
          <Badge variant="neutral">Neutral</Badge>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{isFa ? 'آواتارها و بارگذاری' : 'Avatars & loading'}</CardTitle>
            <CardDescription>{isFa ? 'آواتار و اسکلتون' : 'Avatar fallbacks and skeletons'}</CardDescription>
          </div>
        </CardHeader>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.85rem', flexWrap: 'wrap' }}>
          <Avatar size="sm" fallbackName="Hamid" />
          <Avatar size="md" fallbackName="Alice" />
          <Avatar size="lg" fallbackName="Babak" />
          <Avatar size="xl" fallbackName="Golnaz" />
          <Avatar size="md" src="/this-will-fail.png" fallbackName={mockMe.username} />
          <Spinner size="sm" />
          <Spinner />
          <Spinner size="lg" />
          <Skeleton className="h-10 w-32" />
          <Skeleton className="h-10 w-10 rounded-full" />
        </div>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>{isFa ? 'فرم' : 'Form'}</CardTitle>
            <CardDescription>{isFa ? 'ورودی‌ها' : 'Inputs and textareas'}</CardDescription>
          </div>
        </CardHeader>
        <form style={{ display: 'grid', gap: '0.75rem', maxWidth: 480 }} onSubmit={(e) => e.preventDefault()}>
          <div style={{ display: 'grid', gap: '0.35rem' }}>
            <Label htmlFor="dev-name">{isFa ? 'نام' : 'Name'}</Label>
            <Input id="dev-name" placeholder={isFa ? 'نام شما...' : 'Your name...'} />
          </div>
          <div style={{ display: 'grid', gap: '0.35rem' }}>
            <Label htmlFor="dev-bio">{isFa ? 'درباره' : 'Bio'}</Label>
            <Textarea
              id="dev-bio"
              placeholder={isFa ? 'کمی درباره خودتان...' : 'Tell us about yourself...'}
            />
          </div>
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            <Button type="submit">{isFa ? 'ذخیره' : 'Save'}</Button>
            <Button type="button" variant="ghost" onClick={() => toast.info(isFa ? 'لغو شد.' : 'Cancelled.')}>
              {isFa ? 'لغو' : 'Cancel'}
            </Button>
          </div>
        </form>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>
              <Palette size={18} style={{ display: 'inline', verticalAlign: -4, marginInlineEnd: 6 }} />
              {isFa ? 'توکن‌های رنگ' : 'Color tokens'}
            </CardTitle>
            <CardDescription>{isFa ? 'پالت رنگ' : 'Palette preview'}</CardDescription>
          </div>
        </CardHeader>
        <div style={{ display: 'grid', gap: '0.5rem' }}>
          {Object.entries({
            '--sky-50': 'var(--sky-50)',
            '--sky-100': 'var(--sky-100)',
            '--sky-300': 'var(--sky-300)',
            '--sky-500': 'var(--sky-500)',
            '--sky-600': 'var(--sky-600)',
            '--danger': 'var(--danger)',
            '--success': 'var(--success)',
            '--warning': 'var(--warning)',
            '--card-solid': 'var(--card-solid)',
          }).map(([name, value]) => (
            <div key={name} style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
              <span
                style={{
                  display: 'inline-block',
                  width: 28,
                  height: 28,
                  borderRadius: 8,
                  background: value,
                  border: '1px solid var(--border)',
                }}
              />
              <code style={{ fontSize: 'var(--fs-sm)', color: 'var(--muted)' }}>{name}</code>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  navigator.clipboard?.writeText(name);
                  toast.success(isFa ? 'کپی شد' : 'Copied');
                }}
              >
                <Copy size={14} />
              </Button>
            </div>
          ))}
        </div>
      </Card>
    </main>
  );
}
