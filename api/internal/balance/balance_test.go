package balance

import (
	"testing"

	"splitwise-assignment/api/internal/domain"
)

func TestSimplifyDoesNotTreatDebtorsAsCreditors(t *testing.T) {
	summary := Summarize(nil, nil)
	summary.Debts = simplify([]domain.BalanceLine{
		{MemberName: "Aisha", NetPaise: 9060419},
		{MemberName: "Dev", NetPaise: 3243025},
		{MemberName: "Priya", NetPaise: -5966582},
		{MemberName: "Rohan", NetPaise: -5515481},
		{MemberName: "Meera", NetPaise: -2043131},
		{MemberName: "Kabir", NetPaise: -250500},
		{MemberName: "Sam", NetPaise: 1472250},
	})

	for _, debt := range summary.Debts {
		if debt.To == "Priya" {
			t.Fatalf("Priya is a debtor and should not receive settlement money: %+v", summary.Debts)
		}
	}
	if !hasPayee(summary.Debts, "Dev") {
		t.Fatalf("expected Dev to receive money, got %+v", summary.Debts)
	}
}

func hasPayee(debts []domain.Debt, payee string) bool {
	for _, debt := range debts {
		if debt.To == payee {
			return true
		}
	}
	return false
}
