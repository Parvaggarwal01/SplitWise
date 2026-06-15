package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"splitwise-assignment/api/internal/auth"
	"splitwise-assignment/api/internal/domain"
	"splitwise-assignment/api/internal/importer"
	"splitwise-assignment/api/internal/store"

	"github.com/jung-kurt/gofpdf"
)

type Server struct {
	store     *store.Memory
	authStore auth.Store
	mux       *http.ServeMux
}

func New(memory *store.Memory, authStore auth.Store) *Server {
	server := &Server{store: memory, authStore: authStore, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("POST /api/imports", s.importCSV)
	s.mux.HandleFunc("POST /api/login", s.login)
	s.mux.HandleFunc("POST /api/register", s.register)
	s.mux.HandleFunc("GET /api/imports/latest", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.Report())
	})
	s.mux.HandleFunc("GET /api/imports/latest/report.md", s.downloadImportReport)
	s.mux.HandleFunc("GET /api/imports/latest/report.pdf", s.downloadImportReportPDF)
	s.mux.HandleFunc("DELETE /api/imports/latest", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.ClearImport())
	})
	s.mux.HandleFunc("PATCH /api/imports/latest/anomalies", s.reviewAnomaly)
	s.mux.HandleFunc("GET /api/groups/default/members", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.Members())
	})
	s.mux.HandleFunc("GET /api/groups/default/expenses", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.Expenses())
	})
	s.mux.HandleFunc("GET /api/groups/default/balances", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.Balances())
	})
}

func (s *Server) downloadImportReport(w http.ResponseWriter, r *http.Request) {
	report := s.store.Report()
	content := renderImportReport(report)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="import-report.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(content))
}

func (s *Server) downloadImportReportPDF(w http.ResponseWriter, r *http.Request) {
	report := s.store.Report()
	content, err := renderImportReportPDF(report)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="import-report.pdf"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid login payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Password) == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}
	user, err := s.authStore.Login(r.Context(), payload.Email, payload.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid register payload", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(payload.Name)
	email := strings.TrimSpace(payload.Email)
	if name == "" || email == "" || strings.TrimSpace(payload.Password) == "" {
		http.Error(w, "name, email and password are required", http.StatusBadRequest)
		return
	}
	user, err := s.authStore.Register(r.Context(), name, email, payload.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) reviewAnomaly(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		RowNumber int    `json:"rowNumber"`
		Code      string `json:"code"`
		Decision  string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid review payload", http.StatusBadRequest)
		return
	}
	report, err := s.store.ReviewAnomaly(payload.RowNumber, payload.Code, payload.Decision)
	if err != nil {
		http.Error(w, fmt.Sprintf("review failed: %s", err), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) importCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var reader = r.Body
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			http.Error(w, "invalid multipart body", http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()
		reader = file
	}

	options := importer.Options{USDRatePaise: parseUSDRatePaise(r)}
	report, err := importer.ParseWithOptions(reader, options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.store.ReplaceImport(report)
	writeJSON(w, http.StatusCreated, report)
}

func parseUSDRatePaise(r *http.Request) int64 {
	value := strings.TrimSpace(r.FormValue("usdRate"))
	if value == "" {
		value = strings.TrimSpace(r.URL.Query().Get("usdRate"))
	}
	if value == "" {
		return 0
	}
	rate, err := strconv.ParseFloat(value, 64)
	if err != nil || rate <= 0 {
		return 0
	}
	return int64(math.Round(rate * 100))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderImportReport(report domain.ImportReport) string {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# Import Report\n\n")
	fmt.Fprintf(&buffer, "- Import ID: `%s`\n", report.ID)
	fmt.Fprintf(&buffer, "- Imported at: %s\n", report.ImportedAt.Format(time.RFC3339))
	fmt.Fprintf(&buffer, "- Rows read: %d\n", report.RowsRead)
	fmt.Fprintf(&buffer, "- Imported expenses: %d\n", len(report.Expenses))
	fmt.Fprintf(&buffer, "- Settlements recorded: %d\n", len(report.Settlements))
	fmt.Fprintf(&buffer, "- Anomalies detected: %d\n\n", len(report.Anomalies))

	fmt.Fprintf(&buffer, "## Anomalies\n\n")
	if len(report.Anomalies) == 0 {
		fmt.Fprintf(&buffer, "No anomalies detected.\n")
		return buffer.String()
	}

	fmt.Fprintf(&buffer, "| Row | Code | Severity | Message | Policy | Action |\n")
	fmt.Fprintf(&buffer, "| --- | --- | --- | --- | --- | --- |\n")
	for _, anomaly := range report.Anomalies {
		fmt.Fprintf(
			&buffer,
			"| %d | %s | %s | %s | %s | %s |\n",
			anomaly.RowNumber,
			escapeMarkdownCell(labelize(anomaly.Code)),
			escapeMarkdownCell(labelize(anomaly.Severity)),
			escapeMarkdownCell(anomaly.Message),
			escapeMarkdownCell(anomaly.Policy),
			escapeMarkdownCell(labelize(anomaly.Action)),
		)
	}
	return buffer.String()
}

func renderImportReportPDF(report domain.ImportReport) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Import Report", false)
	pdf.SetAuthor("Flat Ledger", false)
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	pdf.SetFillColor(15, 107, 95)
	pdf.Rect(0, 0, 210, 32, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 22)
	pdf.SetXY(14, 10)
	pdf.CellFormat(120, 8, "Flat Ledger Import Report", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.SetXY(14, 21)
	pdf.CellFormat(120, 5, "Generated from the latest CSV import", "", 0, "L", false, 0, "")

	pdf.SetTextColor(29, 36, 40)
	pdf.SetY(42)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 7, "Summary", "", 1, "L", false, 0, "")

	cardY := pdf.GetY() + 2
	addPDFCard(pdf, 14, cardY, "Rows read", strconv.Itoa(report.RowsRead))
	addPDFCard(pdf, 62, cardY, "Imported expenses", strconv.Itoa(len(report.Expenses)))
	addPDFCard(pdf, 110, cardY, "Settlements", strconv.Itoa(len(report.Settlements)))
	addPDFCard(pdf, 158, cardY, "Anomalies", strconv.Itoa(len(report.Anomalies)))

	pdf.SetY(cardY + 25)
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(93, 104, 111)
	pdf.CellFormat(0, 5, "Import ID: "+report.ID, "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, "Imported at: "+report.ImportedAt.Format(time.RFC3339), "", 1, "L", false, 0, "")

	pdf.Ln(6)
	pdf.SetTextColor(29, 36, 40)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 7, "Anomalies and Actions", "", 1, "L", false, 0, "")

	if len(report.Anomalies) == 0 {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 7, "No anomalies detected.", "", 1, "L", false, 0, "")
	} else {
		for _, anomaly := range report.Anomalies {
			addPDFAnomaly(pdf, anomaly)
		}
	}

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func addPDFCard(pdf *gofpdf.Fpdf, x float64, y float64, label string, value string) {
	pdf.SetDrawColor(216, 221, 212)
	pdf.SetFillColor(248, 250, 248)
	pdf.RoundedRect(x, y, 38, 18, 2, "1234", "FD")
	pdf.SetXY(x+3, y+3)
	pdf.SetTextColor(93, 104, 111)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(32, 4, label, "", 1, "L", false, 0, "")
	pdf.SetX(x + 3)
	pdf.SetTextColor(29, 36, 40)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(32, 7, value, "", 0, "L", false, 0, "")
}

func addPDFAnomaly(pdf *gofpdf.Fpdf, anomaly domain.ImportAnomaly) {
	if pdf.GetY() > 250 {
		pdf.AddPage()
	}
	x := 14.0
	y := pdf.GetY() + 2
	width := 182.0

	pdf.SetDrawColor(228, 230, 224)
	pdf.SetFillColor(255, 255, 255)
	pdf.RoundedRect(x, y, width, 31, 2, "1234", "FD")

	pdf.SetXY(x+4, y+4)
	pdf.SetTextColor(29, 36, 40)
	pdf.SetFont("Arial", "B", 10)
	title := fmt.Sprintf("Row %d: %s", anomaly.RowNumber, labelize(anomaly.Code))
	pdf.CellFormat(112, 5, title, "", 0, "L", false, 0, "")

	pdf.SetTextColor(118, 90, 0)
	pdf.SetFillColor(255, 246, 215)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(x+134, y+4)
	pdf.CellFormat(42, 5, labelize(anomaly.Action), "", 0, "C", true, 0, "")

	pdf.SetTextColor(93, 104, 111)
	pdf.SetFont("Arial", "", 8)
	pdf.SetXY(x+4, y+11)
	pdf.MultiCell(172, 4, "Severity: "+labelize(anomaly.Severity), "", "L", false)
	pdf.SetX(x + 4)
	pdf.MultiCell(172, 4, "Issue: "+anomaly.Message, "", "L", false)
	pdf.SetX(x + 4)
	pdf.MultiCell(172, 4, "Policy: "+anomaly.Policy, "", "L", false)
	pdf.SetY(y + 34)
}

func labelize(value string) string {
	parts := strings.Fields(strings.ReplaceAll(value, "_", " "))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}
