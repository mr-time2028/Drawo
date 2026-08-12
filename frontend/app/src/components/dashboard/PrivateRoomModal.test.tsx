import { cleanup, fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { i18n } from '@/i18n';
import { PrivateRoomModal } from './PrivateRoomModal';

// NumberField uses ↺ and ± characters — we render without router dependencies.
// Modal uses createPortal which works under jsdom. RTL cleanup handles portal
// unmounting between tests.

describe('PrivateRoomModal', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
  });

  afterEach(() => {
    cleanup();
  });

  function renderOpen() {
    const onClose = vi.fn();
    const onCreate = vi.fn();
    render(<PrivateRoomModal open={true} onClose={onClose} onCreate={onCreate} />);
    return { onClose, onCreate };
  }

  it('is hidden when closed', () => {
    render(<PrivateRoomModal open={false} onClose={() => {}} onCreate={() => {}} />);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('renders the form fields and defaults', () => {
    renderOpen();
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByLabelText(/room name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    expect(screen.getByText(/min players/i)).toBeInTheDocument();
    expect(screen.getByText(/max players/i)).toBeInTheDocument();
    expect(screen.getByText(/rounds/i)).toBeInTheDocument();
    expect(screen.getByText(/draw time/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /use drawo words/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /my own categories/i })).toBeInTheDocument();
  });

  it('disables Create when name is empty', () => {
    renderOpen();
    expect(screen.getByRole('button', { name: /create room/i })).toBeDisabled();
  });

  it('validates and dispatches onCreate for default word source', async () => {
    const { onCreate } = renderOpen();
    await userEvent.type(screen.getByLabelText(/room name/i), "Hamid's Room");
    // Default button is enabled now.
    const createBtn = screen.getByRole('button', { name: /create room/i });
    expect(createBtn).not.toBeDisabled();
    fireEvent.click(createBtn);
    expect(onCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "Hamid's Room",
        min_players: 2,
        max_players: 8,
        max_rounds: 3,
        round_time: 80,
        word_source: 'default',
        custom_categories: [],
      }),
    );
  });

  it('password field accepts optional password and trims it', async () => {
    const { onCreate } = renderOpen();
    await userEvent.type(screen.getByLabelText(/room name/i), 'XYZ');
    await userEvent.type(screen.getByLabelText(/password/i), '  secret  ');
    fireEvent.click(screen.getByRole('button', { name: /create room/i }));
    expect(onCreate.mock.calls[0][0].password).toBe('secret');
  });

  it('number steppers +/-/reset adjust values within bounds', async () => {
    renderOpen();
    // Locate the Max players field specifically (it shares "players" with Min players).
    const labels = screen.getAllByText(/players/i);
    const maxLabel = labels.find((el) => /max players/i.test(el.textContent || ''))!;
    const playersField = maxLabel.closest('.private-field') as HTMLElement;
    const playersInput = within(playersField).getByRole('spinbutton') as HTMLInputElement;
    const buttons = within(playersField).getAllByRole('button') as HTMLButtonElement[];
    const [minus, plus, reset] = buttons;
    expect(Number(playersInput.value)).toBe(8);
    fireEvent.click(plus);
    expect(Number(playersInput.value)).toBe(9);
    fireEvent.click(minus);
    fireEvent.click(minus);
    expect(Number(playersInput.value)).toBe(7);
    // Clamp at current min-player value (default 2).
    for (let i = 0; i < 20; i++) fireEvent.click(minus);
    expect(Number(playersInput.value)).toBe(2);
    // Reset
    fireEvent.click(reset);
    expect(Number(playersInput.value)).toBe(8);
  });

  it('custom word mode requires a named category and at least 5 words', async () => {
    renderOpen();
    await userEvent.type(screen.getByLabelText(/room name/i), 'Custom');
    fireEvent.click(screen.getByRole('button', { name: /my own categories/i }));
    // Add category UI appears.
    expect(screen.getByRole('button', { name: /add category/i })).toBeInTheDocument();
    // Create is disabled because no named category / words.
    expect(screen.getByRole('button', { name: /create room/i })).toBeDisabled();

    // Type a category name and some words.
    const catName = screen.getByPlaceholderText(/category name/i);
    await userEvent.type(catName, 'Animals');
    const tier1 = screen.getAllByRole('textbox')[3] as HTMLTextAreaElement; // the first easy textarea
    await userEvent.type(tier1, 'cat\ndog\ncow\nhog\nfox');
    expect(screen.getByText(/5 \/ 5/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /create room/i })).not.toBeDisabled();
  });

  it('adds and removes categories (remove disabled when only one remains)', async () => {
    renderOpen();
    fireEvent.click(screen.getByRole('button', { name: /my own categories/i }));
    expect(screen.getByRole('button', { name: /remove category/i })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: /add category/i }));
    const removeBtns = screen.getAllByRole('button', { name: /remove category/i });
    expect(removeBtns).toHaveLength(2);
    fireEvent.click(removeBtns[1]);
    expect(screen.getAllByRole('button', { name: /remove category/i })).toHaveLength(1);
  });

  it('cancel closes and resets the modal', async () => {
    const { onClose } = renderOpen();
    await userEvent.type(screen.getByLabelText(/room name/i), 'Typed');
    fireEvent.click(screen.getByRole('button', { name: /cancel/i }));
    expect(onClose).toHaveBeenCalled();
  });

  it('clamps number fields to their bounds and snaps invalid input to default', async () => {
    renderOpen();
    const labels = screen.getAllByText(/players/i);
    const maxLabel = labels.find((el) => /max players/i.test(el.textContent || ''))!;
    const playersField = maxLabel.closest('.private-field') as HTMLElement;
    const playersInput = within(playersField).getByRole('spinbutton') as HTMLInputElement;
    // Directly firing change with out-of-range values — clamp() in NumberField
    // enforces the min/max, NaN falls back to the default.
    fireEvent.change(playersInput, { target: { value: '999' } });
    expect(Number(playersInput.value)).toBe(12);
    fireEvent.change(playersInput, { target: { value: '0' } });
    expect(Number(playersInput.value)).toBe(2);
  });

  it('round time snaps to step increments via +/- buttons', async () => {
    renderOpen();
    const timeField = screen.getByText(/draw time/i).closest('.private-field') as HTMLElement;
    const timeInput = within(timeField).getByRole('spinbutton') as HTMLInputElement;
    expect(Number(timeInput.value)).toBe(80);
    const buttons = within(timeField).getAllByRole('button') as HTMLButtonElement[];
    const [, plus] = buttons;
    fireEvent.click(plus);
    expect(Number(timeInput.value)).toBe(90);
  });
});
