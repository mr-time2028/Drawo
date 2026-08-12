import { Gamepad2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { Modal } from '@/components/ui/Modal';

type HistoryModalProps = {
  open: boolean;
  onClose: () => void;
};

export function HistoryModal({ open, onClose }: HistoryModalProps) {
  const { t } = useTranslation();
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t('dashboard.history.modalTitle')}
      description={t('dashboard.history.modalDescription')}
      className="history-modal"
    >
      <div className="history-modal-empty">
        <span className="history-modal-icon" aria-hidden="true">
          <Gamepad2 size={26} strokeWidth={2.2} />
        </span>
        <p className="history-modal-text">{t('dashboard.history.empty')}</p>
        <p className="history-modal-sub">{t('dashboard.history.comingSoon')}</p>
      </div>
    </Modal>
  );
}
