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
			ID:          "empty",
			ImportedAt:  time.Now().UTC(),
			Expenses:    []domain.Expense{},
			Settlements: []domain.Settlement{},
			Anomalies:   []domain.ImportAnomaly{},
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
	report = normalizeReport(report)
	m.report = report
}

func (m *Memory) Report() domain.ImportReport {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return normalizeReport(m.report)
}

func (m *Memory) Expenses() []domain.Expense {
	m.mu.RLock()
	defer m.mu.RUnlock()
	report := normalizeReport(m.report)
	return append([]domain.Expense{}, report.Expenses...)
}

func (m *Memory) Members() []domain.Member {
	m.mu.RLock()
	defer m.mu.RUnlock()
	report := normalizeReport(m.report)
	return append([]domain.Member{}, report.Members...)
}

func (m *Memory) Balances() domain.BalanceSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	report := normalizeReport(m.report)
	summary := balance.Summarize(report.Expenses, report.Settlements)
	if summary.Lines == nil {
		summary.Lines = []domain.BalanceLine{}
	}
	if summary.Debts == nil {
		summary.Debts = []domain.Debt{}
	}
	return summary
}

func normalizeReport(report domain.ImportReport) domain.ImportReport {
	if report.Expenses == nil {
		report.Expenses = []domain.Expense{}
	}
	if report.Settlements == nil {
		report.Settlements = []domain.Settlement{}
	}
	if report.Anomalies == nil {
		report.Anomalies = []domain.ImportAnomaly{}
	}
	if report.Members == nil {
		report.Members = []domain.Member{}
	}
	return report
}
