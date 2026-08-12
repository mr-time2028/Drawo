import { History, Medal, Star, Trophy, type LucideIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui/Button';
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';

type OverviewProps = {
  username: string;
  gamesPlayed: number;
  mvps: number;
  wordScore: number;
  reputationScore: number;
  rank: string;
  onShowHistory: () => void;
};

function Stat({
  icon: Icon,
  label,
  value,
  tint = 'blue',
}: {
  icon: LucideIcon;
  label: string;
  value: string | number;
  tint?: 'blue' | 'gold' | 'green' | 'purple';
}) {
  return (
    <div className={`stat-card stat-tint-${tint}`}>
      <span className="stat-icon" aria-hidden="true">
        <Icon size={20} strokeWidth={2.2} />
      </span>
      <div className="stat-text">
        <span className="stat-value">{value}</span>
        <span className="stat-label">{label}</span>
      </div>
    </div>
  );
}

export function OverviewSection({
  username,
  gamesPlayed,
  mvps,
  wordScore,
  reputationScore,
  rank,
  onShowHistory,
}: OverviewProps) {
  const { t } = useTranslation();

  return (
    <div className="dashboard-section overview-section">
      <div className="overview-header">
        <div>
          <h2 className="overview-title">{t('dashboard.overview.hello', { name: username })}</h2>
          <p className="overview-subtitle">{t('dashboard.overview.summary')}</p>
        </div>
        {rank && (
          <span className="overview-rank-badge" aria-label={t('dashboard.overview.rank')}>
            {rank}
          </span>
        )}
      </div>

      <div className="stats-grid" aria-label={t('dashboard.overview.statsLabel')}>
        <Stat icon={Trophy} label={t('dashboard.overview.gamesPlayed')} value={gamesPlayed} />
        <Stat icon={Medal} label={t('dashboard.overview.mvps')} value={mvps} tint="gold" />
        <Stat icon={Star} label={t('dashboard.overview.wordScore')} value={wordScore} tint="green" />
        <Stat icon={Star} label={t('dashboard.overview.reputation')} value={reputationScore} tint="purple" />
      </div>

      <Card padding="lg" className="history-card">
        <CardHeader className="history-header">
          <div>
            <CardTitle className="history-title">
              <span className="history-title-icon" aria-hidden="true">
                <History size={22} strokeWidth={2.2} />
              </span>
              {t('dashboard.history.title')}
            </CardTitle>
            <CardDescription>{t('dashboard.history.recentGames')}</CardDescription>
          </div>
          <Button variant="ghost" size="sm" onClick={onShowHistory} type="button">
            {t('dashboard.history.seeMore')}
          </Button>
        </CardHeader>
        <div className="history-list" role="list" aria-label={t('dashboard.history.recentGames')}>
          <p className="history-empty">{t('dashboard.history.empty')}</p>
        </div>
      </Card>
    </div>
  );
}
