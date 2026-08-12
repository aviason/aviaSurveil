package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"archive/zip"
	"github.com/aviason/aviaSurveil/internal/checklistintake"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/platform/database"
)

func TestCanonicalAPIDefaultIntakeUsesPostgresStoreWhenPoolIsConfigured(t *testing.T) {
	api := NewCanonicalAPI(CanonicalAPIDependencies{Pool: &database.Pool{}})
	if api.checklistIntake == nil {
		t.Fatal("canonical API did not configure checklist intake")
	}
	store, ok := api.checklistIntake.Store.(*checklistintake.PostgresStore)
	if !ok {
		t.Fatalf("default checklist intake store = %T, want *checklistintake.PostgresStore", api.checklistIntake.Store)
	}
	if store.Pool == nil {
		t.Fatal("default PostgreSQL checklist intake store lost the configured pool")
	}
}

type trackingBody struct {
	reads int
}

func (body *trackingBody) Read([]byte) (int, error) {
	body.reads++
	return 0, errors.New("body must not be read")
}

func (body *trackingBody) Close() error { return nil }

type oneByteMultipartBody struct {
	data         []byte
	archiveStart int
	position     int
	archiveReads int
}

func (body *oneByteMultipartBody) Read(buffer []byte) (int, error) {
	if body.position >= len(body.data) {
		return 0, io.EOF
	}
	if body.position >= body.archiveStart {
		body.archiveReads++
	}
	buffer[0] = body.data[body.position]
	body.position++
	return 1, nil
}

func (body *oneByteMultipartBody) Close() error { return nil }

func TestGovernedChecklistIntakeRejectsMismatchedHeaderBeforeArchiveRead(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	receipt := `{"operationId":"OP-MISMATCH","idempotencyKey":"RECEIPT-KEY","expectedArchiveSha256":"sha256:0000000000000000000000000000000000000000000000000000000000000000","reason":"candidate-only"}`
	part, err := form.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="receipt"; filename="receipt.json"`},
		"Content-Type":        []string{"application/json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(receipt)); err != nil {
		t.Fatal(err)
	}
	archivePayload := []byte("UNIQUE-ARCHIVE-CONTENT-MUST-NOT-BE-READ")
	part, err = form.CreateFormFile("archive", "archive.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(archivePayload); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}
	archiveStart := bytes.Index(body.Bytes(), archivePayload)
	if archiveStart < 0 {
		t.Fatal("archive payload marker was not found")
	}
	requestBody := &oneByteMultipartBody{data: body.Bytes(), archiveStart: archiveStart}
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/governed-checklist/import-batches", requestBody)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("Idempotency-Key", "HEADER-KEY")
	request = request.WithContext(context.WithValue(request.Context(), principalContextKey{}, identity.Principal{SubjectID: "admin", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}))
	response := httptest.NewRecorder()
	NewCanonicalAPI(CanonicalAPIDependencies{}).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("mismatched header status = %d body=%s, want %d", response.Code, response.Body.String(), http.StatusBadRequest)
	}
	if requestBody.archiveReads != 0 {
		t.Fatalf("mismatched header archive-content reads = %d, want 0", requestBody.archiveReads)
	}
}

func TestGovernedChecklistAdminOrganizationIsCheckedBeforeReadingArchive(t *testing.T) {
	body := &trackingBody{}
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/governed-checklist/import-batches", body)
	request.Header.Set("Content-Type", "multipart/form-data; boundary=unused")
	request = request.WithContext(context.WithValue(request.Context(), principalContextKey{}, identity.Principal{SubjectID: "admin", OrganizationID: "ORG-OTHER", Roles: []identity.Role{identity.RoleAdmin}}))
	response := httptest.NewRecorder()
	NewCanonicalAPI(CanonicalAPIDependencies{}).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-CAA Admin status = %d body=%s, want %d", response.Code, response.Body.String(), http.StatusForbidden)
	}
	if body.reads != 0 {
		t.Fatalf("non-CAA Admin body reads = %d, want 0", body.reads)
	}
}

func TestGovernedChecklistIntakeRejectsDuplicateMultipartPartsAndMissingHeader(t *testing.T) {
	data := syntheticArchiveBytes(t)
	makeRequest := func(duplicateArchive, duplicateReceipt bool, header string) *http.Request {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		writeArchive := func() {
			part, err := form.CreateFormFile("archive", "archive.zip")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(data); err != nil {
				t.Fatal(err)
			}
		}
		writeReceipt := func() {
			receipt := `{"operationId":"OP-DUP","idempotencyKey":"IDEM-DUP","expectedArchiveSha256":"sha256:0000000000000000000000000000000000000000000000000000000000000000","reason":"candidate-only"}`
			if err := form.WriteField("receipt", receipt); err != nil {
				t.Fatal(err)
			}
		}
		writeReceipt()
		if duplicateReceipt {
			writeReceipt()
		}
		writeArchive()
		if duplicateArchive {
			writeArchive()
		}
		if err := form.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/v1/admin/governed-checklist/import-batches", &body)
		request.Header.Set("Content-Type", form.FormDataContentType())
		if header != "" {
			request.Header.Set("Idempotency-Key", header)
		}
		return request.WithContext(context.WithValue(request.Context(), principalContextKey{}, identity.Principal{SubjectID: "admin", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}))
	}
	for name, request := range map[string]*http.Request{
		"duplicate archive":          makeRequest(true, false, "IDEM-DUP"),
		"duplicate receipt":          makeRequest(false, true, "IDEM-DUP"),
		"missing idempotency header": makeRequest(false, false, ""),
	} {
		response := httptest.NewRecorder()
		NewCanonicalAPI(CanonicalAPIDependencies{}).Handler().ServeHTTP(response, request)
		if response.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s status = %d body=%s, want %d", name, response.Code, response.Body.String(), http.StatusUnprocessableEntity)
		}
	}
}

func TestGovernedChecklistIntakeRoutesAuthenticateBeforeReadingArchive(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/governed-checklist/import-batches", bytes.NewReader([]byte("not multipart")))
	response := httptest.NewRecorder()
	NewCanonicalAPI(CanonicalAPIDependencies{}).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated intake status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestGovernedChecklistAdminInventoryDoesNotExposeToDepartmentManager(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/governed-checklist/import-batches/BATCH-1/files", nil).WithContext(context.WithValue(context.Background(), principalContextKey{}, identity.Principal{SubjectID: "manager", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}}))
	response := httptest.NewRecorder()
	NewCanonicalAPI(CanonicalAPIDependencies{}).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("manager inventory status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func syntheticArchiveBytes(t *testing.T) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	body := []byte("%PDF-1.7\nsynthetic")
	header := &zip.FileHeader{Name: "form.pdf", Method: zip.Store, CRC32: crc32.ChecksumIEEE(body), CompressedSize64: uint64(len(body)), UncompressedSize64: uint64(len(body))}
	entry, err := writer.CreateRaw(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func TestGovernedChecklistIntakeReceiptIsCandidateOnlyAndReplayStable(t *testing.T) {
	archive := bytes.NewBuffer(syntheticArchiveBytes(t))
	digest := sha256.Sum256(archive.Bytes())
	expectedArchiveSHA := fmt.Sprintf("sha256:%x", digest[:])
	if _, err := checklistintake.NewService(nil).ReceiveArchive(context.Background(), identity.Principal{SubjectID: "admin", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}, "OP-DIRECT", "IDEM-DIRECT", expectedArchiveSHA, "candidate-only", archive.Bytes()); err != nil {
		t.Fatalf("direct intake failed: %v", err)
	}
	makeRequest := func() *http.Request {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		receipt := map[string]string{"operationId": "OP-HTTP", "idempotencyKey": "IDEM-HTTP", "expectedArchiveSha256": expectedArchiveSHA, "reason": "candidate-only"}
		part, err := form.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": []string{`form-data; name="receipt"; filename="receipt.json"`},
			"Content-Type":        []string{"application/json"},
		})
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(receipt)
		_, _ = part.Write(encoded)
		part, err = form.CreateFormFile("archive", "archive.zip")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(archive.Bytes()); err != nil {
			t.Fatal(err)
		}
		_ = form.Close()
		request := httptest.NewRequest(http.MethodPost, "/v1/admin/governed-checklist/import-batches", &body)
		request.Header.Set("Content-Type", form.FormDataContentType())
		request.Header.Set("Idempotency-Key", "IDEM-HTTP")
		return request.WithContext(context.WithValue(request.Context(), principalContextKey{}, identity.Principal{SubjectID: "admin", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleAdmin}}))
	}
	api := NewCanonicalAPI(CanonicalAPIDependencies{})
	first := httptest.NewRecorder()
	api.Handler().ServeHTTP(first, makeRequest())
	if first.Code != http.StatusCreated {
		t.Fatalf("first intake status = %d body=%s", first.Code, first.Body.String())
	}
	second := httptest.NewRecorder()
	api.Handler().ServeHTTP(second, makeRequest())
	if second.Code != http.StatusCreated || !bytes.Contains(second.Body.Bytes(), []byte(`"replayed":true`)) {
		t.Fatalf("replay intake response = %d body=%s", second.Code, second.Body.String())
	}
}
