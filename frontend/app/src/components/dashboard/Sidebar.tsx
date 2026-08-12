import {
  ChevronLeft,
  ChevronRight,
  LayoutDashboard,
  LogOut,
  ShieldCheck,
  User as UserIcon,
  type LucideIcon,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { Avatar } from '@/components/ui/Avatar';
import { cn } from '@/utils/cn';

export type DashboardSection = 'overview' | 'recovery' | 'profile';

type SidebarProps = {
  collapsed: boolean;
  onToggleCollapsed: () => void;
  active: DashboardSection;
  onSelect: (section: DashboardSection) => void;
  username: string;
  avatarUrl?: string;
  onLogout: () => void;
  logoutLoading?: boolean;
};

type ItemDef = {
  id: DashboardSection;
  icon: LucideIcon;
  labelKey: string;
};

const items: ItemDef[] = [
  { id: 'overview', icon: LayoutDashboard, labelKey: 'dashboard.nav.overview' },
  { id: 'recovery', icon: ShieldCheck, labelKey: 'dashboard.nav.recovery' },
  { id: 'profile', icon: UserIcon, labelKey: 'dashboard.nav.profile' },
];

export function Sidebar({
  collapsed,
  onToggleCollapsed,
  active,
  onSelect,
  username,
  avatarUrl,
  onLogout,
  logoutLoading,
}: SidebarProps) {
  const { t } = useTranslation();

  return (
    <aside
      className={cn('dashboard-sidebar', collapsed && 'is-collapsed')}
      aria-label={t('dashboard.nav.sidebarLabel')}
    >
      <div className="dashboard-sidebar-header">
        <div className="dashboard-sidebar-user">
          <Avatar size="md" src={avatarUrl} alt={username} fallbackName={username} />
          {!collapsed && (
            <div className="dashboard-sidebar-user-meta">
              <span className="dashboard-sidebar-username">{username}</span>
              <span className="dashboard-sidebar-sub">{t('dashboard.nav.playerLabel')}</span>
            </div>
          )}
        </div>
        <button
          type="button"
          className="dashboard-sidebar-collapse"
          onClick={onToggleCollapsed}
          aria-label={collapsed ? t('dashboard.nav.expand') : t('dashboard.nav.collapse')}
          title={collapsed ? t('dashboard.nav.expand') : t('dashboard.nav.collapse')}
        >
          {/* Chev direction flips with dir=rtl automatically because we rely
              on start/end semantics via rotate? Simpler: show chevron that points
              toward the edge the sidebar is attached to. */}
          <ChevronCondensed collapsed={collapsed} />
        </button>
      </div>

      <nav className="dashboard-sidebar-nav" aria-label={t('dashboard.nav.mainNav')}>
        <ul className="dashboard-sidebar-list">
          {items.map(({ id, icon: Icon, labelKey }) => {
            const isActive = active === id;
            return (
              <li key={id}>
                <button
                  type="button"
                  className={cn('dashboard-sidebar-link', isActive && 'is-active')}
                  onClick={() => onSelect(id)}
                  aria-current={isActive ? 'page' : undefined}
                  title={collapsed ? t(labelKey) : undefined}
                >
                  <span className="dashboard-sidebar-link-icon" aria-hidden="true">
                    <Icon size={20} strokeWidth={2.2} />
                  </span>
                  {!collapsed && <span className="dashboard-sidebar-link-label">{t(labelKey)}</span>}
                </button>
              </li>
            );
          })}
        </ul>
      </nav>

      <div className="dashboard-sidebar-footer">
        <button
          type="button"
          className="dashboard-sidebar-link dashboard-sidebar-logout"
          onClick={onLogout}
          disabled={logoutLoading}
          title={collapsed ? t('auth.logout') : undefined}
        >
          <span className="dashboard-sidebar-link-icon" aria-hidden="true">
            <LogOut size={20} strokeWidth={2.2} />
          </span>
          {!collapsed && (
            <span className="dashboard-sidebar-link-label">
              {logoutLoading ? t('auth.loggingOut') : t('auth.logout')}
            </span>
          )}
        </button>
      </div>
    </aside>
  );
}

/** Chevron points towards the nearest side edge. In LTR sidebar sits on the
 *  left: collapsed means chevron points LEFT (hide), expanded means chevron
 *  points RIGHT (hide becomes expand from the other side). We use
 *  logical-direction via two mirrored SVGs chosen by dir at runtime. */
function ChevronCondensed({ collapsed }: { collapsed: boolean }) {
  // When dir=ltr: collapsed → chevron-left (toward the left edge), else chevron-right.
  // When dir=rtl: collapsed → chevron-right (toward the right edge), else chevron-left.
  // We simply render both and toggle visibility via CSS that respects [dir].
  return (
    <>
      <ChevronLeft
        size={18}
        strokeWidth={2.4}
        className="dashboard-sidebar-chevron show-in-rtl-collapsed show-in-ltr-expanded"
        style={{ display: 'none' }}
        aria-hidden="true"
      />
      <ChevronRight
        size={18}
        strokeWidth={2.4}
        className="dashboard-sidebar-chevron show-in-ltr-collapsed show-in-rtl-expanded"
        aria-hidden="true"
      />
      <style>{`
        :where([dir='ltr']) .dashboard-sidebar-collapse .show-in-ltr-collapsed { display: ${collapsed ? 'none' : 'inline-block'}; }
        :where([dir='ltr']) .dashboard-sidebar-collapse .show-in-ltr-expanded { display: ${collapsed ? 'inline-block' : 'none'}; }
        :where([dir='rtl']) .dashboard-sidebar-collapse .show-in-rtl-collapsed { display: ${collapsed ? 'inline-block' : 'none'}; }
        :where([dir='rtl']) .dashboard-sidebar-collapse .show-in-rtl-expanded { display: ${collapsed ? 'none' : 'inline-block'}; }
      `}</style>
    </>
  );
}
