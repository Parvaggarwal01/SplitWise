package store

import (
	"sync"
	"time"

	"splitwise-assignment/api/internal/balance"
	"splitwise-assignment/api/internal/domain"
)

type Memory struct {
	mu     sync.RWMutex
	report domain.ImportReport
}

func NewMemory() *Memory {
	return &Memory{
		report: domain.ImportReport{
			ID:         "empty",
			ImportedAt: time.Now().UTC(),
			Members: []domain.Member{
				{ID: "aisha", Name: "Aisha", JoinedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
				{ID: "rohan", Name: "Rohan", JoinedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
				{ID: "priya", Name: "Priya", JoinedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
	}
}

func (m *Memory) ReplaceImport(report domain.ImportReport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.report = report
}

func (m *Memory) Report() domain.ImportReport {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.report
}

func (m *Memory) Expenses() []domain.Expense {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]domain.Expense(nil), m.report.Expenses...)
}

func (m *Memory) Members() []domain.Member {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]domain.Member(nil), m.report.Members...)
}

func (m *Memory) Balances() domain.BalanceSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return balance.Summarize(m.report.Expenses, m.report.Settlements)
}
