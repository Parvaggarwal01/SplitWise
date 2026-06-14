package store

import (
	"fmt"
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
	return &Memory{report: emptyReport()}
}

func (m *Memory) ReplaceImport(report domain.ImportReport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	report = normalizeReport(report)
	m.report = report
}

func (m *Memory) ClearImport() domain.ImportReport {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.report = emptyReport()
	return m.report
}

func (m *Memory) ReviewAnomaly(rowNumber int, code string, decision string) (domain.ImportReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	action := ""
	switch decision {
	case "approve":
		action = "approved"
	case "keep_skipped":
		action = "kept_skipped"
	default:
		return domain.ImportReport{}, fmt.Errorf("unsupported decision %q", decision)
	}

	found := false
	for i := range m.report.Anomalies {
		if m.report.Anomalies[i].RowNumber == rowNumber && m.report.Anomalies[i].Code == code {
			m.report.Anomalies[i].Action = action
			m.report.Anomalies[i].Severity = "reviewed"
			found = true
		}
	}
	if !found {
		return domain.ImportReport{}, fmt.Errorf("anomaly not found")
	}
	m.report = normalizeReport(m.report)
	return m.report, nil
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

func emptyReport() domain.ImportReport {
	return domain.ImportReport{
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
	}
}
