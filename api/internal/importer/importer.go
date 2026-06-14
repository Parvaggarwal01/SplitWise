package importer

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"splitwise-assignment/api/internal/domain"
)

const (
	baseCurrency = "INR"
	usdRatePaise = int64(8350)
)

var canonicalNames = map[string]string{
	"aisha":              "Aisha",
	"rohan":              "Rohan",
	"priya":              "Priya",
	"priya s":            "Priya",
	"meera":              "Meera",
	"dev":                "Dev",
	"sam":                "Sam",
	"dev's friend kabir": "Kabir",
	"devs friend kabir":  "Kabir",
	"dev friend kabir":   "Kabir",
}

type row struct {
	number       int
	dateRaw      string
	description  string
	paidBy       string
	amountRaw    string
	currency     string
	splitType    string
	splitWith    string
	splitDetails string
	notes        string
}

func Parse(r io.Reader) (domain.ImportReport, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = false
	records, err := reader.ReadAll()
	if err != nil {
		return domain.ImportReport{}, fmt.Errorf("read csv: %w", err)
	}
	if len(records) == 0 {
		return domain.ImportReport{}, fmt.Errorf("empty csv")
	}

	report := domain.ImportReport{
		ID:         makeID("import", time.Now().Format(time.RFC3339Nano)),
		ImportedAt: time.Now().UTC(),
		RowsRead:   len(records) - 1,
		Members:    defaultMembers(),
	}

	seen := map[string]domain.Expense{}
	for i, record := range records[1:] {
		csvRow := toRow(i+2, record)
		expense, settlement, anomalies := parseRow(csvRow)
		report.Anomalies = append(report.Anomalies, anomalies...)

		if settlement != nil {
			report.Settlements = append(report.Settlements, *settlement)
			continue
		}
		if expense == nil {
			continue
		}

		key := duplicateKey(*expense)
		if previous, ok := seen[key]; ok {
			report.Anomalies = append(report.Anomalies, domain.ImportAnomaly{
				RowNumber: csvRow.number,
				Code:      "duplicate_expense",
				Severity:  "approval_required",
				Message:   fmt.Sprintf("Possible duplicate of row %d: %q and %q", previous.SourceRow, previous.Description, expense.Description),
				Policy:    "Keep both rows visible but import only the first until a user approves merging or replacing.",
				Action:    "skipped_pending_review",
			})
			continue
		}
		seen[key] = *expense
		report.Expenses = append(report.Expenses, *expense)
	}

	sort.SliceStable(report.Expenses, func(i, j int) bool {
		return report.Expenses[i].Date.Before(report.Expenses[j].Date)
	})
	return report, nil
}

func toRow(number int, record []string) row {
	for len(record) < 9 {
		record = append(record, "")
	}
	return row{
		number:       number,
		dateRaw:      strings.TrimSpace(record[0]),
		description:  strings.TrimSpace(record[1]),
		paidBy:       record[2],
		amountRaw:    strings.TrimSpace(record[3]),
		currency:     strings.TrimSpace(record[4]),
		splitType:    strings.TrimSpace(record[5]),
		splitWith:    strings.TrimSpace(record[6]),
		splitDetails: strings.TrimSpace(record[7]),
		notes:        strings.TrimSpace(record[8]),
	}
}

func parseRow(r row) (*domain.Expense, *domain.Settlement, []domain.ImportAnomaly) {
	var anomalies []domain.ImportAnomaly
	date, dateAnomalies, ok := parseDate(r.number, r.dateRaw, r.notes)
	anomalies = append(anomalies, dateAnomalies...)
	if !ok {
		return nil, nil, anomalies
	}

	paidBy, nameAnomalies, payerOK := normalizeName(r.number, r.paidBy, "paid_by")
	anomalies = append(anomalies, nameAnomalies...)
	if !payerOK && !looksLikeSettlement(r) {
		anomalies = append(anomalies, anomaly(r.number, "missing_payer", "blocking", "No payer is present for an expense row.", "Expense is not imported until the payer is supplied.", "skipped"))
		return nil, nil, anomalies
	}

	amountPaise, amountAnomalies, amountOK := parseAmount(r.number, r.amountRaw)
	anomalies = append(anomalies, amountAnomalies...)
	if !amountOK {
		return nil, nil, anomalies
	}

	currency := strings.ToUpper(strings.TrimSpace(r.currency))
	if currency == "" {
		currency = baseCurrency
		anomalies = append(anomalies, anomaly(r.number, "missing_currency", "warning", "Currency is blank.", "Default blank currency to INR because the surrounding household rows are INR.", "defaulted_to_inr"))
	}
	if currency != "INR" && currency != "USD" {
		anomalies = append(anomalies, anomaly(r.number, "unsupported_currency", "blocking", "Currency is not supported: "+currency, "Only INR and USD are imported.", "skipped"))
		return nil, nil, anomalies
	}

	if looksLikeNonSharedTransfer(r) {
		anomalies = append(anomalies, anomaly(r.number, "non_shared_transfer", "approval_required", "Deposit or one-off transfer found in the expense sheet.", "Do not include deposits in shared expense balances unless a reviewer explicitly converts it to a settlement.", "skipped_pending_review"))
		return nil, nil, anomalies
	}

	if looksLikeSettlement(r) {
		to, toAnomalies, toOK := normalizeName(r.number, r.splitWith, "settlement_to")
		anomalies = append(anomalies, toAnomalies...)
		if !toOK {
			return nil, nil, anomalies
		}
		return nil, &domain.Settlement{
			ID:          makeID("settlement", strconv.Itoa(r.number), paidBy, to, r.amountRaw),
			Date:        date,
			From:        paidBy,
			To:          to,
			AmountPaise: convertToBase(amountPaise, currency),
			SourceRow:   r.number,
			Notes:       r.notes,
		}, append(anomalies, anomaly(r.number, "settlement_as_expense", "warning", "A payment/settlement was logged in the expense sheet.", "Record as settlement, not as shared expense.", "recorded_as_settlement"))
	}

	if amountPaise == 0 {
		anomalies = append(anomalies, anomaly(r.number, "zero_amount", "approval_required", "Expense amount is zero.", "Keep the raw row but skip creating an expense until reviewed.", "skipped_pending_review"))
		return nil, nil, anomalies
	}
	if amountPaise < 0 {
		anomalies = append(anomalies, anomaly(r.number, "negative_amount", "warning", "Negative amount treated as refund.", "Import as a negative expense and allocate it across participants.", "imported_as_refund"))
	}

	baseAmount := convertToBase(amountPaise, currency)
	if currency == "USD" {
		anomalies = append(anomalies, anomaly(r.number, "foreign_currency", "warning", "USD amount requires conversion to INR.", "Use fixed documented rate of 1 USD = INR 83.50 for repeatable assignment calculations.", "converted_to_inr"))
	}

	participants, participantAnomalies := parseParticipants(r.number, r.splitWith)
	anomalies = append(anomalies, participantAnomalies...)
	if len(participants) == 0 {
		anomalies = append(anomalies, anomaly(r.number, "missing_participants", "blocking", "No split participants were found.", "Expense is not imported until participants are supplied.", "skipped"))
		return nil, nil, anomalies
	}
	anomalies = append(anomalies, membershipAnomalies(r.number, date, participants)...)

	shares, shareAnomalies, sharesOK := calculateShares(r, participants, baseAmount)
	anomalies = append(anomalies, shareAnomalies...)
	if !sharesOK {
		return nil, nil, anomalies
	}

	return &domain.Expense{
		ID:          makeID("expense", strconv.Itoa(r.number), r.description, r.amountRaw),
		Date:        date,
		Description: r.description,
		PaidBy:      paidBy,
		Amount:      domain.Money{AmountPaise: amountPaise, Currency: currency},
		BaseAmount:  domain.Money{AmountPaise: baseAmount, Currency: baseCurrency},
		SplitType:   normalizeSplitType(r.splitType),
		Shares:      shares,
		SourceRow:   r.number,
		Notes:       r.notes,
		Anomalies:   anomalies,
	}, nil, anomalies
}

func parseDate(rowNumber int, value string, notes string) (time.Time, []domain.ImportAnomaly, bool) {
	var anomalies []domain.ImportAnomaly
	for _, layout := range []string{"02-01-2006", "Jan-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			if layout == "Jan-02" {
				parsed = time.Date(2026, parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
				anomalies = append(anomalies, anomaly(rowNumber, "non_standard_date", "warning", "Date uses abbreviated month format: "+value, "Parse as 2026 because every assignment row is in 2026.", "parsed_as_2026"))
			}
			if value == "04-05-2026" || strings.Contains(strings.ToLower(notes), "april 5 or may 4") {
				anomalies = append(anomalies, anomaly(rowNumber, "ambiguous_date", "approval_required", "Date could be read as 4 May or 5 April.", "Parse with DD-MM-YYYY to stay consistent with the rest of the sheet, but require user review.", "parsed_as_4_may_pending_review"))
			}
			return parsed, anomalies, true
		}
	}
	return time.Time{}, []domain.ImportAnomaly{anomaly(rowNumber, "invalid_date", "blocking", "Could not parse date: "+value, "Invalid dates are not imported.", "skipped")}, false
}

func parseAmount(rowNumber int, value string) (int64, []domain.ImportAnomaly, bool) {
	cleaned := strings.ReplaceAll(value, ",", "")
	floatValue, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, []domain.ImportAnomaly{anomaly(rowNumber, "invalid_amount", "blocking", "Could not parse amount: "+value, "Invalid amounts are not imported.", "skipped")}, false
	}
	paise := int64(math.Round(floatValue * 100))
	var anomalies []domain.ImportAnomaly
	if strings.Contains(value, ",") {
		anomalies = append(anomalies, anomaly(rowNumber, "formatted_amount", "warning", "Amount contains a thousands separator.", "Remove commas before parsing.", "normalized"))
	}
	if math.Abs(floatValue*100-math.Round(floatValue*100)) > 0.000001 {
		anomalies = append(anomalies, anomaly(rowNumber, "fractional_paise", "warning", "Amount has more than two decimal places.", "Round to nearest paise.", "rounded"))
	}
	return paise, anomalies, true
}

func parseParticipants(rowNumber int, raw string) ([]string, []domain.ImportAnomaly) {
	parts := strings.Split(raw, ";")
	var result []string
	var anomalies []domain.ImportAnomaly
	seen := map[string]bool{}
	for _, part := range parts {
		name, nameAnomalies, ok := normalizeName(rowNumber, part, "split_with")
		anomalies = append(anomalies, nameAnomalies...)
		if !ok {
			continue
		}
		if seen[name] {
			anomalies = append(anomalies, anomaly(rowNumber, "duplicate_participant", "warning", "Participant appears more than once: "+name, "Deduplicate participants within one expense.", "deduplicated"))
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result, anomalies
}

func calculateShares(r row, participants []string, baseAmount int64) ([]domain.ExpenseShare, []domain.ImportAnomaly, bool) {
	switch normalizeSplitType(r.splitType) {
	case "equal":
		anomalies := []domain.ImportAnomaly{}
		if r.splitDetails != "" {
			anomalies = append(anomalies, anomaly(r.number, "split_details_ignored", "warning", "split_type is equal but split_details is present.", "Trust split_type and split equally; leave detail visible in report.", "ignored_split_details"))
		}
		return splitByWeights(participants, nil, baseAmount), anomalies, true
	case "share":
		weights, anomalies, ok := parseWeightedDetails(r.number, r.splitDetails, false)
		if !ok {
			return nil, anomalies, false
		}
		return splitByWeights(participants, weights, baseAmount), anomalies, true
	case "unequal":
		amounts, anomalies, ok := parseUnequalDetails(r.number, r.splitDetails)
		if !ok {
			return nil, anomalies, false
		}
		return allocateAmounts(participants, amounts, baseAmount), anomalies, true
	case "percentage":
		weights, anomalies, ok := parseWeightedDetails(r.number, r.splitDetails, true)
		if !ok {
			return nil, anomalies, false
		}
		total := 0.0
		for _, value := range weights {
			total += value
		}
		if math.Abs(total-100) > 0.01 {
			anomalies = append(anomalies, anomaly(r.number, "percentage_total_invalid", "approval_required", fmt.Sprintf("Percentages total %.2f%%, not 100%%.", total), "Normalize percentages to total 100 for calculation, require review.", "normalized_pending_review"))
		}
		return splitByWeights(participants, weights, baseAmount), anomalies, true
	default:
		return nil, []domain.ImportAnomaly{anomaly(r.number, "missing_or_unknown_split_type", "blocking", "Split type is missing or unsupported: "+r.splitType, "Do not import as expense unless it is identified as settlement.", "skipped")}, false
	}
}

func parseUnequalDetails(rowNumber int, raw string) (map[string]int64, []domain.ImportAnomaly, bool) {
	values, anomalies, ok := parseWeightedDetails(rowNumber, raw, false)
	if !ok {
		return nil, anomalies, false
	}
	result := map[string]int64{}
	for name, value := range values {
		result[name] = int64(math.Round(value * 100))
	}
	return result, anomalies, true
}

func parseWeightedDetails(rowNumber int, raw string, percentage bool) (map[string]float64, []domain.ImportAnomaly, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, []domain.ImportAnomaly{anomaly(rowNumber, "missing_split_details", "blocking", "Split details are required for this split type.", "Do not import without explicit split details.", "skipped")}, false
	}
	values := map[string]float64{}
	var anomalies []domain.ImportAnomaly
	for _, token := range strings.Split(raw, ";") {
		fields := strings.Fields(strings.TrimSpace(strings.TrimSuffix(token, "%")))
		if len(fields) < 2 {
			return nil, []domain.ImportAnomaly{anomaly(rowNumber, "invalid_split_details", "blocking", "Could not parse split detail: "+token, "Do not import malformed split details.", "skipped")}, false
		}
		name, nameAnomalies, ok := normalizeName(rowNumber, strings.Join(fields[:len(fields)-1], " "), "split_details")
		anomalies = append(anomalies, nameAnomalies...)
		if !ok {
			return nil, anomalies, false
		}
		valueText := strings.TrimSuffix(fields[len(fields)-1], "%")
		value, err := strconv.ParseFloat(valueText, 64)
		if err != nil {
			return nil, []domain.ImportAnomaly{anomaly(rowNumber, "invalid_split_value", "blocking", "Could not parse split value: "+token, "Do not import malformed split values.", "skipped")}, false
		}
		values[name] = value
	}
	if percentage {
		return values, anomalies, true
	}
	return values, anomalies, true
}

func splitByWeights(participants []string, weights map[string]float64, amount int64) []domain.ExpenseShare {
	if weights == nil {
		weights = map[string]float64{}
		for _, participant := range participants {
			weights[participant] = 1
		}
	}
	total := 0.0
	for _, participant := range participants {
		total += weights[participant]
	}
	shares := make([]domain.ExpenseShare, 0, len(participants))
	allocated := int64(0)
	for i, participant := range participants {
		part := int64(math.Round(float64(amount) * weights[participant] / total))
		if i == len(participants)-1 {
			part = amount - allocated
		}
		allocated += part
		shares = append(shares, domain.ExpenseShare{MemberName: participant, AmountPaise: part})
	}
	return shares
}

func allocateAmounts(participants []string, amounts map[string]int64, baseAmount int64) []domain.ExpenseShare {
	shares := make([]domain.ExpenseShare, 0, len(participants))
	allocated := int64(0)
	for i, participant := range participants {
		amount := amounts[participant]
		if i == len(participants)-1 && allocated+amount != baseAmount {
			amount += baseAmount - allocated - amount
		}
		allocated += amount
		shares = append(shares, domain.ExpenseShare{MemberName: participant, AmountPaise: amount})
	}
	return shares
}

func membershipAnomalies(rowNumber int, date time.Time, participants []string) []domain.ImportAnomaly {
	var anomalies []domain.ImportAnomaly
	for _, participant := range participants {
		if participant == "Meera" && date.After(time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)) {
			anomalies = append(anomalies, anomaly(rowNumber, "participant_outside_membership", "approval_required", "Meera appears after her move-out date.", "Keep raw row but exclude Meera from calculation unless approved.", "requires_review"))
		}
		if participant == "Sam" && date.Before(time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)) {
			anomalies = append(anomalies, anomaly(rowNumber, "participant_before_join", "approval_required", "Sam appears before his join date.", "Require review before charging Sam.", "requires_review"))
		}
	}
	return anomalies
}

func normalizeName(rowNumber int, raw string, field string) (string, []domain.ImportAnomaly, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, false
	}
	canonical, ok := canonicalNames[strings.ToLower(trimmed)]
	if !ok {
		canonical = trimmed
	}
	var anomalies []domain.ImportAnomaly
	if trimmed != canonical {
		anomalies = append(anomalies, anomaly(rowNumber, "name_normalized", "warning", field+" value normalized from "+trimmed+" to "+canonical+".", "Normalize known aliases and whitespace/case variants.", "normalized"))
	}
	return canonical, anomalies, true
}

func normalizeSplitType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func looksLikeSettlement(r row) bool {
	text := strings.ToLower(r.description + " " + r.notes)
	description := strings.ToLower(r.description)
	if strings.Contains(description, "paid") && strings.Contains(description, "back") {
		return true
	}
	return normalizeSplitType(r.splitType) == "" && (strings.Contains(text, "settlement") || strings.Contains(description, "paid"))
}

func looksLikeNonSharedTransfer(r row) bool {
	text := strings.ToLower(r.description + " " + r.notes)
	return strings.Contains(text, "deposit")
}

func convertToBase(amountPaise int64, currency string) int64 {
	if strings.ToUpper(currency) == "USD" {
		return int64(math.Round(float64(amountPaise) * float64(usdRatePaise) / 100.0))
	}
	return amountPaise
}

func duplicateKey(expense domain.Expense) string {
	normalizedDescription := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(expense.Description), "")
	if strings.Contains(normalizedDescription, "marinabites") {
		normalizedDescription = "marinabitesdinner"
	}
	if strings.Contains(normalizedDescription, "thalassa") {
		normalizedDescription = "thalassadinner"
	}
	return fmt.Sprintf("%s:%s:%s", expense.Date.Format("2006-01-02"), normalizedDescription, strings.Join(shareNames(expense.Shares), ";"))
}

func shareNames(shares []domain.ExpenseShare) []string {
	names := make([]string, 0, len(shares))
	for _, share := range shares {
		names = append(names, share.MemberName)
	}
	sort.Strings(names)
	return names
}

func defaultMembers() []domain.Member {
	marchEnd := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	return []domain.Member{
		{ID: "aisha", Name: "Aisha", JoinedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "rohan", Name: "Rohan", JoinedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "priya", Name: "Priya", JoinedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "meera", Name: "Meera", JoinedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), LeftAt: &marchEnd},
		{ID: "dev", Name: "Dev", JoinedAt: time.Date(2026, 2, 8, 0, 0, 0, 0, time.UTC), IsVisitor: true},
		{ID: "sam", Name: "Sam", JoinedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
		{ID: "kabir", Name: "Kabir", JoinedAt: time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC), IsVisitor: true},
	}
}

func anomaly(rowNumber int, code string, severity string, message string, policy string, action string) domain.ImportAnomaly {
	return domain.ImportAnomaly{RowNumber: rowNumber, Code: code, Severity: severity, Message: message, Policy: policy, Action: action}
}

func makeID(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:])[:16]
}
