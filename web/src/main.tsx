import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { AlertTriangle, ArrowRight, CheckCircle2, Database, FileUp, LogOut, Trash2, UsersRound } from 'lucide-react';
import { api, labelize, money, shortDate } from './lib/api';
import type { BalanceSummary, Expense, ImportReport, Member } from './lib/types';
import './styles/app.css';

function App() {
  const [user, setUser] = useState<{ name: string; email: string; token: string } | null>(() => {
    const stored = window.localStorage.getItem('flat-ledger-user');
    return stored ? JSON.parse(stored) : null;
  });
  const [report, setReport] = useState<ImportReport | null>(null);
  const [balances, setBalances] = useState<BalanceSummary | null>(null);
  const [expenses, setExpenses] = useState<Expense[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [error, setError] = useState('');
  const [usdRate, setUsdRate] = useState(83.5);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const refresh = async () => {
    const [latest, summary, expenseRows, memberRows] = await Promise.all([
      api.latestImport(),
      api.balances(),
      api.expenses(),
      api.members()
    ]);
    setReport(latest);
    setBalances(summary);
    setExpenses(expenseRows ?? []);
    setMembers(memberRows ?? []);
  };

  useEffect(() => {
    refresh().catch((err: Error) => setError(err.message));
  }, []);

  const anomalyCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const anomaly of report?.anomalies ?? []) {
      counts.set(anomaly.severity, (counts.get(anomaly.severity) ?? 0) + 1);
    }
    return Array.from(counts.entries());
  }, [report]);

  const reportExpenses = report?.expenses ?? [];
  const reportAnomalies = report?.anomalies ?? [];
  const balanceDebts = balances?.debts ?? [];
  const balanceLines = balances?.lines ?? [];
  const expenseRows = expenses ?? [];

  if (!user) {
    return <LoginScreen onLogin={setUser} />;
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Shared expenses</p>
          <h1>Flat Ledger</h1>
        </div>
        <div className="actions">
          <div className="userBadge">
            <span>{user.name}</span>
            <button
              type="button"
              title="Sign out"
              onClick={() => {
                window.localStorage.removeItem('flat-ledger-user');
                setUser(null);
              }}
            >
              <LogOut size={16} />
            </button>
          </div>
          <label className="rateInput">
            <span>USD rate</span>
            <input
              min="0.01"
              step="0.01"
              type="number"
              value={usdRate}
              onChange={(event) => setUsdRate(Number(event.target.value))}
            />
          </label>
          <label className="upload">
            <FileUp size={18} />
            Import CSV
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,text/csv"
              onChange={async (event) => {
                const file = event.target.files?.[0];
                if (!file) return;
                setError('');
                try {
                  await api.importFile(file, usdRate);
                  await refresh();
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Import failed');
                }
              }}
            />
          </label>
          <button
            className="clearButton"
            type="button"
            onClick={async () => {
              setError('');
              try {
                await api.clearImport();
                if (fileInputRef.current) fileInputRef.current.value = '';
                await refresh();
              } catch (err) {
                setError(err instanceof Error ? err.message : 'Clear failed');
              }
            }}
          >
            <Trash2 size={18} />
            Clear import
          </button>
        </div>
      </header>

      {error && <div className="error">{error}</div>}

      <section className="metrics">
        <Metric icon={<Database />} label="Rows read" value={String(report?.rowsRead ?? 0)} />
        <Metric icon={<CheckCircle2 />} label="Imported expenses" value={String(reportExpenses.length)} />
        <Metric icon={<AlertTriangle />} label="Anomalies" value={String(reportAnomalies.length)} />
        <Metric icon={<UsersRound />} label="People" value={String(members.length)} />
      </section>

      <section className="grid">
        <Panel title="Who Pays Whom">
          <div className="debtList">
            {balanceDebts.map((debt) => (
              <div className="debt" key={`${debt.from}-${debt.to}-${debt.amountPaise}`}>
                <span>{debt.from}</span>
                <ArrowRight size={16} />
                <span>{debt.to}</span>
                <strong>{money(debt.amountPaise)}</strong>
              </div>
            ))}
            {balanceDebts.length === 0 && <p className="muted">No balances yet.</p>}
          </div>
        </Panel>

        <Panel title="Individual Balances">
          <table>
            <thead>
              <tr>
                <th>Person</th>
                <th>Paid</th>
                <th>Share</th>
                <th>Net</th>
              </tr>
            </thead>
            <tbody>
              {balanceLines.map((line) => (
                <tr key={line.memberName}>
                  <td>{line.memberName}</td>
                  <td>{money(line.paidPaise)}</td>
                  <td>{money(line.sharePaise)}</td>
                  <td className={line.netPaise >= 0 ? 'positive' : 'negative'}>{money(line.netPaise)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>
      </section>

      <section className="grid">
        <Panel title="Import Report">
          <div className="chips">
            {anomalyCounts.map(([severity, count]) => (
              <span className="chip" key={severity}>{labelize(severity)}: {count}</span>
            ))}
          </div>
          <div className="anomalies">
            {reportAnomalies.map((anomaly, index) => (
              <article className="anomaly" key={`${anomaly.rowNumber}-${anomaly.code}-${index}`}>
                <div>
                  <strong>Row {anomaly.rowNumber}: {labelize(anomaly.code)}</strong>
                  <p>{anomaly.message}</p>
                  <p className="muted">{anomaly.policy}</p>
                </div>
                <div className="reviewColumn">
                  <span>{labelize(anomaly.action)}</span>
                  {anomaly.severity === 'approval_required' && (
                    <div className="reviewActions">
                      <button
                        type="button"
                        onClick={async () => {
                          setError('');
                          try {
                            await api.reviewAnomaly(anomaly.rowNumber, anomaly.code, 'approve');
                            await refresh();
                          } catch (err) {
                            setError(err instanceof Error ? err.message : 'Approval failed');
                          }
                        }}
                      >
                        Approve
                      </button>
                      <button
                        type="button"
                        onClick={async () => {
                          setError('');
                          try {
                            await api.reviewAnomaly(anomaly.rowNumber, anomaly.code, 'keep_skipped');
                            await refresh();
                          } catch (err) {
                            setError(err instanceof Error ? err.message : 'Review failed');
                          }
                        }}
                      >
                        Keep skipped
                      </button>
                    </div>
                  )}
                </div>
              </article>
            ))}
          </div>
        </Panel>

        <Panel title="Membership Timeline">
          <div className="members">
            {members.map((member) => (
              <div className="member" key={member.id}>
                <strong>{member.name}</strong>
                <span>{shortDate(member.joinedAt)} {member.leftAt ? `to ${shortDate(member.leftAt)}` : 'onward'}</span>
                {member.isVisitor && <em>visitor</em>}
              </div>
            ))}
          </div>
        </Panel>
      </section>

      <Panel title="Expense Trace">
        <table>
          <thead>
            <tr>
              <th>Row</th>
              <th>Date</th>
              <th>Description</th>
              <th>Paid by</th>
              <th>Amount</th>
              <th>Split</th>
            </tr>
          </thead>
          <tbody>
            {expenseRows.map((expense) => (
              <tr key={expense.id}>
                <td>{expense.sourceRow}</td>
                <td>{shortDate(expense.date)}</td>
                <td>{expense.description}</td>
                <td>{expense.paidBy}</td>
                <td>{money(expense.baseAmount.amountPaise)}</td>
                <td>{expense.splitType}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Panel>
    </main>
  );
}

function LoginScreen({ onLogin }: { onLogin: (user: { name: string; email: string; token: string }) => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [name, setName] = useState('Aisha');
  const [email, setEmail] = useState('aisha@example.com');
  const [password, setPassword] = useState('demo123');
  const [error, setError] = useState('');

  return (
    <main className="loginShell">
      <form
        className="loginPanel"
        onSubmit={async (event) => {
          event.preventDefault();
          setError('');
          const submitter = event.nativeEvent instanceof SubmitEvent ? event.nativeEvent.submitter : null;
          const action = submitter instanceof HTMLButtonElement ? submitter.dataset.authAction : mode;
          try {
            const user = action === 'login'
              ? await api.login(email, password)
              : await api.register(name, email, password);
            window.localStorage.setItem('flat-ledger-user', JSON.stringify(user));
            onLogin(user);
          } catch (err) {
            setError(err instanceof Error ? err.message : 'Authentication failed');
          }
        }}
      >
        <p className="eyebrow">Flat Ledger</p>
        <h1>{mode === 'login' ? 'Sign in' : 'Create account'}</h1>
        <div className="authTabs">
          <button
            className={mode === 'login' ? 'active' : ''}
            type="button"
            onClick={() => {
              setError('');
              setMode('login');
            }}
          >
            Login
          </button>
          <button
            className={mode === 'register' ? 'active' : ''}
            type="button"
            onClick={() => {
              setError('');
              setMode('register');
            }}
          >
            Register
          </button>
        </div>
        {mode === 'register' && (
          <label>
            Name
            <input value={name} onChange={(event) => setName(event.target.value)} type="text" />
          </label>
        )}
        <label>
          Email
          <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" />
        </label>
        <label>
          Password
          <input value={password} onChange={(event) => setPassword(event.target.value)} type="password" />
        </label>
        {error && <div className="error">{error}</div>}
        <button data-auth-action={mode} type="submit">{mode === 'login' ? 'Continue' : 'Create account'}</button>
      </form>
    </main>
  );
}

function Metric({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="metric">
      {icon}
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="panel">
      <h2>{title}</h2>
      {children}
    </section>
  );
}

createRoot(document.getElementById('root')!).render(<App />);
