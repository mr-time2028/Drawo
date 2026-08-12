import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { i18n } from '@/i18n';
import { OverviewSection } from './OverviewSection';

describe('OverviewSection', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('greets the user, shows stats, and fires onShowHistory', () => {
    const onShowHistory = vi.fn();
    render(
      <OverviewSection
        username="Hamid"
        gamesPlayed={12}
        mvps={3}
        wordScore={450}
        reputationScore={1200}
        rank="Gold"
        onShowHistory={onShowHistory}
      />,
    );
    expect(screen.getByRole('heading', { name: /hello,\s*hamid/i })).toBeInTheDocument();
    expect(screen.getByText('12')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText('450')).toBeInTheDocument();
    expect(screen.getByText('1200')).toBeInTheDocument();
    expect(screen.getByText('Gold')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /see more/i }));
    expect(onShowHistory).toHaveBeenCalled();
  });

  it('hides the rank badge when rank is empty', () => {
    render(
      <OverviewSection
        username="X"
        gamesPlayed={0}
        mvps={0}
        wordScore={0}
        reputationScore={0}
        rank=""
        onShowHistory={() => {}}
      />,
    );
    expect(screen.queryByLabelText(/rank/i)).not.toBeInTheDocument();
  });
});
