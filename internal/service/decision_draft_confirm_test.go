package service_test

// Захирамжийг баталгаажуулах: хавсралт файл ЗААВАЛ; файл «Захирамж» төрлөөр
// захирамж ба түүнд холбогдсон нэгж талбаруудын хавсралт болно (repo Confirm
// нэг гүйлгээнд холбоно). DB алдаа гарвал хадгалсан файл устгагдана.

import (
	"context"
	"errors"
	"io"
	"testing"

	"acquisition_backend/internal/domain"
	"acquisition_backend/internal/pkg/filestorage"
	"acquisition_backend/internal/repository"
	"acquisition_backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type confirmStubRepo struct {
	repository.DecisionDraftRepository

	draft      *domain.DecisionDraft
	confirmErr error
	confirmDoc *domain.Document
	confirmed  bool
}

func (s *confirmStubRepo) GetByID(context.Context, string) (*domain.DecisionDraft, error) {
	return s.draft, nil
}
func (s *confirmStubRepo) DecreeDocumentTypeID(context.Context) (int, error) { return 42, nil }
func (s *confirmStubRepo) Confirm(_ context.Context, _, _, _, _, _ string, doc *domain.Document) error {
	s.confirmed = true
	s.confirmDoc = doc
	return s.confirmErr
}
func (s *confirmStubRepo) ListDraftFundingSources(context.Context, string) ([]*domain.DecisionDraftFundingLink, error) {
	return nil, nil
}
func (s *confirmStubRepo) GetDocument(context.Context, string) (*domain.DecisionDraftDocument, error) {
	return nil, nil
}

// fakeFiles — хадгалсан/устгасан файлыг тэмдэглэнэ.
type fakeFiles struct {
	saved     []string
	deleted   []string
	namespace string
	ownerID   string
}

func (f *fakeFiles) Save(_ context.Context, namespace, ownerID, filename, contentType string, data []byte, _ bool) (*filestorage.StoredFile, error) {
	f.namespace, f.ownerID = namespace, ownerID
	url := "/api/files/" + namespace + "/" + ownerID + "/" + filename
	f.saved = append(f.saved, url)
	return &filestorage.StoredFile{URL: url, Name: filename, Type: contentType, SizeBytes: int64(len(data))}, nil
}
func (f *fakeFiles) SaveStream(context.Context, string, string, string, string, io.Reader, int64, bool) (*filestorage.StoredFile, error) {
	return nil, errors.New("not used")
}
func (f *fakeFiles) Read(context.Context, string) ([]byte, error)              { return nil, nil }
func (f *fakeFiles) OpenStream(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (f *fakeFiles) Delete(_ context.Context, url string) error {
	f.deleted = append(f.deleted, url)
	return nil
}

func confirmReq(att *service.DecisionAttachment) service.ConfirmDecisionDraftRequest {
	return service.ConfirmDecisionDraftRequest{DecreeNumber: "А/123", DecisionDate: "2026-10-29", Attachment: att}
}

func TestDecisionConfirm_RequiresAttachment(t *testing.T) {
	repo := &confirmStubRepo{draft: &domain.DecisionDraft{ID: "d1", Status: domain.DecisionDraftStatusDraft}}
	files := &fakeFiles{}
	svc := service.NewDecisionDraftService(repo, files)

	_, err := svc.Confirm(context.Background(), "d1", confirmReq(nil), "user-1")

	assert.ErrorIs(t, err, domain.ErrDecisionAttachmentRequired)
	assert.False(t, repo.confirmed, "хавсралтгүй үед баталгаажих ёсгүй")
	assert.Empty(t, files.saved)
}

func TestDecisionConfirm_StoresAttachmentAsDecreeDocument(t *testing.T) {
	repo := &confirmStubRepo{draft: &domain.DecisionDraft{ID: "d1", Status: domain.DecisionDraftStatusDraft}}
	files := &fakeFiles{}
	svc := service.NewDecisionDraftService(repo, files)

	_, err := svc.Confirm(context.Background(), "d1", confirmReq(&service.DecisionAttachment{
		Filename: "decree.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.4 test"),
	}), "user-1")

	require.NoError(t, err)
	require.True(t, repo.confirmed)
	require.NotNil(t, repo.confirmDoc, "хавсралт repo.Confirm-д дамжих ёстой")
	assert.Equal(t, 42, repo.confirmDoc.DocumentTypeID, "«Захирамж» төрлөөр")
	assert.Equal(t, "Захирамж А/123", repo.confirmDoc.Name)
	assert.Equal(t, files.saved[0], repo.confirmDoc.FileURL)
	assert.Equal(t, "decision", files.namespace)
	assert.Equal(t, "d1", files.ownerID)
	assert.Empty(t, files.deleted)
}

func TestDecisionConfirm_DeletesFileWhenDBFails(t *testing.T) {
	repo := &confirmStubRepo{
		draft:      &domain.DecisionDraft{ID: "d1", Status: domain.DecisionDraftStatusDraft},
		confirmErr: errors.New("db down"),
	}
	files := &fakeFiles{}
	svc := service.NewDecisionDraftService(repo, files)

	_, err := svc.Confirm(context.Background(), "d1", confirmReq(&service.DecisionAttachment{
		Filename: "decree.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.4 test"),
	}), "user-1")

	require.Error(t, err)
	assert.Equal(t, files.saved, files.deleted, "DB алдаа гарвал хадгалсан файл устгагдана")
}

func TestDecisionConfirm_AlreadyConfirmed(t *testing.T) {
	repo := &confirmStubRepo{draft: &domain.DecisionDraft{ID: "d1", Status: domain.DecisionDraftStatusConfirmed}}
	files := &fakeFiles{}
	svc := service.NewDecisionDraftService(repo, files)

	_, err := svc.Confirm(context.Background(), "d1", confirmReq(&service.DecisionAttachment{
		Filename: "decree.pdf", ContentType: "application/pdf", Data: []byte("%PDF-1.4 test"),
	}), "user-1")

	assert.ErrorIs(t, err, domain.ErrDecisionConfirmed)
	assert.Empty(t, files.saved, "баталгаажсан бол файл хадгалахгүй")
}
