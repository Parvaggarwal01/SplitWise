import type { BalanceSummary, Expense, ImportReport, Member } from './types';

const json = async <T>(url: string): Promise<T> => {
  const response = await fetch(url);
  if (!response.ok) throw new Error(await response.text());
  return response.json() as Promise<T>;
};

export const api = {
  latestImport: () => json<ImportReport>('/api/imports/latest'),
  balances: () => json<BalanceSummary>('/api/groups/default/balances'),
  expenses: () => json<Expense[]>('/api/groups/default/expenses'),
  members: () => json<Member[]>('/api/groups/default/members'),
  importFile: async (file: File) => {
    const form = new FormData();
    form.append('file', file);
    const response = await fetch('/api/imports', { method: 'POST', body: form });
    if (!response.ok) throw new Error(await response.text());
    return response.json() as Promise<ImportReport>;
  }
};

export const money = (paise: number, currency = 'INR') =>
  new Intl.NumberFormat('en-IN', {
    style: 'currency',
    currency,
    maximumFractionDigits: 2
  }).format(paise / 100);

export const shortDate = (value: string) =>
  new Intl.DateTimeFormat('en-IN', { day: '2-digit', month: 'short' }).format(new Date(value));
