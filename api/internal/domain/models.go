package domain

import "time"

type Money struct {
	AmountPaise int64  `json:"amountPaise"`
	Currency    string `json:"currency"`
}

type Member struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	JoinedAt  time.Time  `json:"joinedAt"`
	LeftAt    *time.Time `json:"leftAt,omitempty"`
	IsVisitor bool       `json:"isVisitor"`
}

type Expense struct {
	ID          string          `json:"id"`
	Date        time.Time       `json:"date"`
	Description string          `json:"description"`
	PaidBy      string          `json:"paidBy"`
	Amount      Money           `json:"amount"`
	BaseAmount  Money           `json:"baseAmount"`
	SplitType   string          `json:"splitType"`
	Shares      []ExpenseShare  `json:"shares"`
	SourceRow   int             `json:"sourceRow"`
	Notes       string          `json:"notes,omitempty"`
	Anomalies   []ImportAnomaly `json:"anomalies,omitempty"`
}

type ExpenseShare struct {
	MemberName  string `json:"memberName"`
	AmountPaise int64  `json:"amountPaise"`
}

type Settlement struct {
	ID          string    `json:"id"`
	Date        time.Time `json:"date"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	AmountPaise int64     `json:"amountPaise"`
	SourceRow   int       `json:"sourceRow"`
	Notes       string    `json:"notes,omitempty"`
}

type ImportAnomaly struct {
	RowNumber int    `json:"rowNumber"`
	Code      string `json:"code"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Policy    string `json:"policy"`
	Action    string `json:"action"`
}

type ImportReport struct {
	ID          string          `json:"id"`
	ImportedAt  time.Time       `json:"importedAt"`
	RowsRead    int             `json:"rowsRead"`
	Expenses    []Expense       `json:"expenses"`
	Settlements []Settlement    `json:"settlements"`
	Anomalies   []ImportAnomaly `json:"anomalies"`
	Members     []Member        `json:"members"`
}

type BalanceLine struct {
	MemberName  string `json:"memberName"`
	NetPaise    int64  `json:"netPaise"`
	PaidPaise   int64  `json:"paidPaise"`
	SharePaise  int64  `json:"sharePaise"`
	DetailCount int    `json:"detailCount"`
}

type Debt struct {
	From        string `json:"from"`
	To          string `json:"to"`
	AmountPaise int64  `json:"amountPaise"`
}

type BalanceSummary struct {
	Currency string        `json:"currency"`
	Lines    []BalanceLine `json:"lines"`
	Debts    []Debt        `json:"debts"`
}
