export type ImportAnomaly = {
  rowNumber: number;
  code: string;
  severity: string;
  message: string;
  policy: string;
  action: string;
};

export type Member = {
  id: string;
  name: string;
  joinedAt: string;
  leftAt?: string;
  isVisitor?: boolean;
};

export type ExpenseShare = {
  memberName: string;
  amountPaise: number;
};

export type Expense = {
  id: string;
  date: string;
  description: string;
  paidBy: string;
  amount: { amountPaise: number; currency: string };
  baseAmount: { amountPaise: number; currency: string };
  splitType: string;
  shares: ExpenseShare[];
  sourceRow: number;
  notes?: string;
  anomalies?: ImportAnomaly[];
};

export type Settlement = {
  id: string;
  date: string;
  from: string;
  to: string;
  amountPaise: number;
  sourceRow: number;
  notes?: string;
};

export type ImportReport = {
  id: string;
  importedAt: string;
  rowsRead: number;
  expenses: Expense[] | null;
  settlements: Settlement[] | null;
  anomalies: ImportAnomaly[] | null;
  members: Member[] | null;
};

export type BalanceLine = {
  memberName: string;
  netPaise: number;
  paidPaise: number;
  sharePaise: number;
  detailCount: number;
};

export type Debt = {
  from: string;
  to: string;
  amountPaise: number;
};

export type BalanceSummary = {
  currency: string;
  lines: BalanceLine[] | null;
  debts: Debt[] | null;
};
