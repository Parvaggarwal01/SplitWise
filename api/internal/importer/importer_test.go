package importer

import (
	"os"
	"testing"

	"splitwise-assignment/api/internal/domain"
)

func TestParseAssignmentCSVDetectsExpectedAnomalies(t *testing.T) {
	file, err := os.Open("../../../sample-data/expenses_export.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	report, err := Parse(file)
	if err != nil {
		t.Fatal(err)
	}
	if report.RowsRead != 42 {
		t.Fatalf("RowsRead = %d, want 42", report.RowsRead)
	}
	if len(report.Expenses) == 0 {
		t.Fatal("expected imported expenses")
	}
	if len(report.Anomalies) < 12 {
		t.Fatalf("expected at least 12 anomalies, got %d", len(report.Anomalies))
	}

	assertAnomaly(t, report.Anomalies, "duplicate_expense")
	assertAnomaly(t, report.Anomalies, "settlement_as_expense")
	assertAnomaly(t, report.Anomalies, "foreign_currency")
	assertAnomaly(t, report.Anomalies, "participant_outside_membership")
	assertAnomaly(t, report.Anomalies, "missing_currency")
	assertAnomaly(t, report.Anomalies, "ambiguous_date")
	assertAnomaly(t, report.Anomalies, "zero_amount")
}

func assertAnomaly(t *testing.T, anomalies []domain.ImportAnomaly, code string) {
	t.Helper()
	for _, anomaly := range anomalies {
		if anomaly.Code == code {
			return
		}
	}
	t.Fatalf("missing anomaly code %s", code)
}
