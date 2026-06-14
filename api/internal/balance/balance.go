package balance

import (
	"sort"

	"splitwise-assignment/api/internal/domain"
)

func Summarize(expenses []domain.Expense, settlements []domain.Settlement) domain.BalanceSummary {
	lines := map[string]*domain.BalanceLine{}
	ensure := func(name string) *domain.BalanceLine {
		if _, ok := lines[name]; !ok {
			lines[name] = &domain.BalanceLine{MemberName: name}
		}
		return lines[name]
	}

	for _, expense := range expenses {
		payer := ensure(expense.PaidBy)
		payer.PaidPaise += expense.BaseAmount.AmountPaise
		payer.DetailCount++
		for _, share := range expense.Shares {
			line := ensure(share.MemberName)
			line.SharePaise += share.AmountPaise
			line.DetailCount++
		}
	}

	for _, settlement := range settlements {
		ensure(settlement.From).PaidPaise += settlement.AmountPaise
		ensure(settlement.To).SharePaise += settlement.AmountPaise
	}

	result := domain.BalanceSummary{Currency: "INR"}
	for _, line := range lines {
		line.NetPaise = line.PaidPaise - line.SharePaise
		result.Lines = append(result.Lines, *line)
	}
	sort.Slice(result.Lines, func(i, j int) bool {
		return result.Lines[i].MemberName < result.Lines[j].MemberName
	})
	result.Debts = simplify(result.Lines)
	return result
}

func simplify(lines []domain.BalanceLine) []domain.Debt {
	var debtors []domain.BalanceLine
	var creditors []domain.BalanceLine
	for _, line := range lines {
		if line.NetPaise < 0 {
			line.NetPaise = -line.NetPaise
			debtors = append(debtors, line)
		} else if line.NetPaise > 0 {
			creditors = append(creditors, line)
		}
	}
	sort.Slice(debtors, func(i, j int) bool { return debtors[i].NetPaise > debtors[j].NetPaise })
	sort.Slice(creditors, func(i, j int) bool { return creditors[i].NetPaise > creditors[j].NetPaise })

	var debts []domain.Debt
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		amount := debtors[i].NetPaise
		if creditors[j].NetPaise < amount {
			amount = creditors[j].NetPaise
		}
		if amount > 0 {
			debts = append(debts, domain.Debt{From: debtors[i].MemberName, To: creditors[j].MemberName, AmountPaise: amount})
		}
		debtors[i].NetPaise -= amount
		creditors[j].NetPaise -= amount
		if debtors[i].NetPaise == 0 {
			i++
		}
		if creditors[j].NetPaise == 0 {
			j++
		}
	}
	return debts
}
