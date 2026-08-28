import type { ScannerRule } from '@/libs/api';

export type SignalsRulesTableProps = {
  items: ScannerRule[];
  loading?: boolean;
  deleteLoading?: boolean;
  toggleLoading?: boolean;
  onDelete: (id: string) => void;
  onToggle: (id: string, enabled: boolean) => void;
  onEdit: (rule: ScannerRule) => void;
};
