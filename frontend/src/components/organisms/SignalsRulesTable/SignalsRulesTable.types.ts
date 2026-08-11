import type { ScannerRule } from '@/libs/api';

export type SignalsRulesTableProps = {
  items: ScannerRule[];
  loading?: boolean;
  deleteLoading?: boolean;
  onDelete: (id: string) => void;
};
