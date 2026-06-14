import type { BalanceSummary, Expense, ImportReport, Member } from './types';

const json = async <T>(url: string): Promise<T> => {
  const response = await fetch(url);
  if (!response.ok) throw new Error(await response.text());
  return response.json() as Promise<T>;
};

export const api = {
  login: async (email: string, password: string) => {
    const response = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });
    if (!response.ok) throw new Error(await response.text());
    return response.json() as Promise<{ name: string; email: string; token: string }>;
  },
  register: async (name: string, email: string, password: string) => {
    const response = await fetch('/api/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, email, password })
    });
    if (!response.ok) throw new Error(await response.text());
    return response.json() as Promise<{ name: string; email: string; token: string }>;
  },
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
  },
  clearImport: async () => {
    const response = await fetch('/api/imports/latest', { method: 'DELETE' });
    if (!response.ok) throw new Error(await response.text());
    return response.json() as Promise<ImportReport>;
  },
  reviewAnomaly: async (rowNumber: number, code: string, decision: 'approve' | 'keep_skipped') => {
    const response = await fetch('/api/imports/latest/anomalies', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rowNumber, code, decision })
    });
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

export const labelize = (value: string) =>
  value
    .replace(/_/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
