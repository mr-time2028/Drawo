import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { i18n } from '@/i18n';
import { StartMatchDrawer } from './StartMatchDrawer';

const GAME_LANG_KEY = 'drawo.gameLanguage';

describe('StartMatchDrawer', () => {
  beforeEach(async () => {
    localStorage.clear();
    // Default game language when nothing is stored is 'fa' (matching the
    // component). Tests can pre-seed GAME_LANG_KEY to change this.
    await i18n.changeLanguage('en');
  });

  afterEach(() => {
    localStorage.clear();
    document.body.innerHTML = '';
  });

  it('is hidden when closed', () => {
    render(<StartMatchDrawer open={false} onClose={() => {}} onOpenPrivate={() => {}} onStartPublic={() => {}} anchorSide="end" />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('renders both language and game-type sections when open and uses persisted game language', () => {
    localStorage.setItem(GAME_LANG_KEY, 'en');
    render(<StartMatchDrawer open={true} onClose={() => {}} onOpenPrivate={() => {}} onStartPublic={() => {}} anchorSide="end" />);
    expect(screen.getByRole('dialog', { name: /start match/i })).toBeInTheDocument();
    expect(screen.getByText(/game language/i)).toBeInTheDocument();
    expect(screen.getByText(/game type/i)).toBeInTheDocument();
    // Game language is independent from i18n; we seeded it to 'en'.
    const enBtn = screen.getByRole('radio', { name: /english/i });
    expect(enBtn).toHaveAttribute('aria-checked', 'true');
  });

  it('closes via X, backdrop click, Escape key, and Cancel button', () => {
    const onClose = vi.fn();
    render(<StartMatchDrawer open={true} onClose={onClose} onOpenPrivate={() => {}} onStartPublic={() => {}} anchorSide="end" />);
    fireEvent.click(screen.getByRole('button', { name: /cancel/i }));
    expect(onClose).toHaveBeenCalledTimes(1);
    fireEvent.click(document.querySelector('.start-match-backdrop')!);
    expect(onClose).toHaveBeenCalledTimes(2);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(3);
  });

  it('switches GAME language independently from UI language and persists it', () => {
    // Start with UI language English and no stored game language (default 'fa').
    render(<StartMatchDrawer open={true} onClose={() => {}} onOpenPrivate={() => {}} onStartPublic={() => {}} anchorSide="end" />);
    const faBtn = screen.getByRole('radio', { name: /فارسی/i });
    const enBtn = screen.getByRole('radio', { name: /english/i });
    // Default is fa.
    expect(faBtn).toHaveAttribute('aria-checked', 'true');
    expect(enBtn).toHaveAttribute('aria-checked', 'false');

    // Switch to English game language.
    fireEvent.click(enBtn);
    expect(enBtn).toHaveAttribute('aria-checked', 'true');
    expect(faBtn).toHaveAttribute('aria-checked', 'false');
    expect(localStorage.getItem(GAME_LANG_KEY)).toBe('en');

    // Switch back to Persian game language.
    fireEvent.click(faBtn);
    expect(faBtn).toHaveAttribute('aria-checked', 'true');
    expect(localStorage.getItem(GAME_LANG_KEY)).toBe('fa');

    // UI/site language MUST NOT have been changed by game language toggles.
    expect(i18n.language).toBe('en');
    expect(document.documentElement.lang).toBe('en');
  });

  it('private room CTA fires openPrivate(lang) + onClose; public is disabled', () => {
    const onClose = vi.fn();
    const onOpenPrivate = vi.fn();
    const onStartPublic = vi.fn();
    localStorage.setItem(GAME_LANG_KEY, 'en');
    render(
      <StartMatchDrawer
        open={true}
        onClose={onClose}
        onOpenPrivate={onOpenPrivate}
        onStartPublic={onStartPublic}
        anchorSide="end"
      />,
    );
    const publicBtn = screen.getByRole('button', { name: /public match/i });
    expect(publicBtn).toBeDisabled();
    expect(screen.getByText(/soon/i)).toBeInTheDocument();
    // The footer CTA reads "Create room" because private is the default selection.
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    expect(onClose).toHaveBeenCalled();
    expect(onOpenPrivate).toHaveBeenCalledWith('en');
    expect(onStartPublic).not.toHaveBeenCalled();
  });

  it('no escape listener while closed', () => {
    const onClose = vi.fn();
    render(<StartMatchDrawer open={false} onClose={onClose} onOpenPrivate={() => {}} onStartPublic={() => {}} anchorSide="end" />);
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(onClose).not.toHaveBeenCalled();
  });
});
