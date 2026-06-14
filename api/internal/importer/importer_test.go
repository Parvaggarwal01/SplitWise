package importer

import (
	"os"
	"strings"
	"testing"

	"splitwise-assignment/api/internal/domain"
)

func TestParseAssignmentCSVDetectsExpectedAnomalies(t *testing.T) {
	csvFixture := assignmentFixture
	if path := os.Getenv("ASSIGNMENT_CSV_PATH"); path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		csvFixture = string(content)
	}

	report, err := Parse(strings.NewReader(csvFixture))
	if err != nil {
		t.Fatal(err)
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
	assertAnomaly(t, report.Anomalies, "non_shared_transfer")
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

const assignmentFixture = `date,description,paid_by,amount,currency,split_type,split_with,split_details,notes
08-02-2026,Dinner at Marina Bites,Dev,3200,INR,equal,Aisha;Rohan;Priya;Dev,,
08-02-2026,dinner - marina bites,Dev,3200,INR,equal,Aisha;Rohan;Priya;Dev,,
10-02-2026,Electricity Feb,Aisha,"1,200",INR,equal,Aisha;Rohan;Priya;Meera,,
14-02-2026,Movie night snacks,priya,640,INR,equal,Aisha;Rohan;Priya,,Meera skipped
15-02-2026,Cylinder refill,Rohan,899.995,INR,equal,Aisha;Rohan;Priya;Meera,,
18-02-2026,Groceries DMart,Priya S,1875,INR,equal,Aisha;Rohan;Priya;Meera,,
22-02-2026,House cleaning supplies,,780,INR,equal,Aisha;Rohan;Priya;Meera,,can't remember who paid
25-02-2026,Rohan paid Aisha back,Rohan,5000,INR,,Aisha,,this is a settlement not an expense??
28-02-2026,Pizza Friday,Aisha,1440,INR,percentage,Aisha;Rohan;Priya;Meera,Aisha 30%; Rohan 30%; Priya 30%; Meera 20%,percentages might be off
09-03-2026,Goa villa booking,Dev,540,USD,equal,Aisha;Rohan;Priya;Dev,,booked on intl site
Mar-14,Airport cab,rohan ,1100,INR,equal,Aisha;Rohan;Priya;Dev,,
15-03-2026,Groceries DMart,Priya,2105,,equal,Aisha;Rohan;Priya;Meera,,forgot to set currency
22-03-2026,Dinner order Swiggy,Priya,0,INR,equal,Aisha;Rohan;Priya;Meera,,counted twice earlier - fixing later
04-05-2026,Deep cleaning service,Rohan,2500,INR,equal,Aisha;Rohan;Priya,,is this April 5 or May 4? format is a mess
02-04-2026,Groceries BigBasket,Priya,2640,INR,equal,Aisha;Rohan;Priya;Meera,,oops Meera still in the group list
08-04-2026,Sam deposit share,Sam,15000,INR,equal,Aisha,,Sam moving in! paid Aisha his deposit
`
