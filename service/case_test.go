package service

import (
	"context"
	"errors"
	"testing"

	"tibsphereai/internal/ai"
	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCaseGen - ai.CaseGeneratorAI ning test uchun soxta implementatsiyasi.
type fakeCaseGen struct {
	res *m.CaseCreateReq
	err error
}

func (f fakeCaseGen) Generate(ctx context.Context, req *m.CaseGenReq) (*m.CaseCreateReq, *ai.Usage, error) {
	return f.res, nil, f.err
}

func newCaseTestService(cm *mocks.CaseMock, caseGen ai.CaseGeneratorAI) *CaseService {
	return NewCaseService(&st.Storage{CaseS: cm}, &ai.Client{CaseGen: caseGen})
}

func TestCaseService_Create(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.CaseCreateReq{}
	want := &m.ById{Id: "case-1"}
	cm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// GenerateWithAI - AI muvaffaqiyatli generatsiya qilsa, IsAIGenerated=true
// qilib storage.Create'ga uzatilishi kerak (BACKEND_STRUCTURE.md 7-bo'lim).
func TestCaseService_GenerateWithAI_Success(t *testing.T) {
	cm := mocks.NewCaseMock()
	aiReq := &m.CaseCreateReq{Difficulty: "medium"}
	svc := newCaseTestService(cm, fakeCaseGen{res: aiReq})

	want := &m.ById{Id: "case-ai-1"}
	cm.On("Create", aiReq).Return(want, nil)

	got, err := svc.GenerateWithAI(context.Background(), &m.CaseGenReq{Topic: "cardiology", Difficulty: "medium"})
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.True(t, aiReq.IsAIGenerated)
	cm.AssertExpectations(t)
}

// GenerateWithAI - AI generatsiya xato bersa, storage.Create umuman chaqirilmasligi kerak.
func TestCaseService_GenerateWithAI_AIError(t *testing.T) {
	cm := mocks.NewCaseMock()
	wantErr := errors.New("GEMINI_API_KEY is not configured")
	svc := newCaseTestService(cm, fakeCaseGen{err: wantErr})

	got, err := svc.GenerateWithAI(context.Background(), &m.CaseGenReq{Topic: "cardiology", Difficulty: "medium"})
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
	cm.AssertNotCalled(t, "Create")
}

func TestCaseService_GetById(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.ById{Id: "case-1"}
	want := &m.CaseRes{}
	cm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCaseService_GetAll(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.CaseGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.CaseList{}
	cm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestCaseService_Update(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.CaseUpdateReq{}
	cm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestCaseService_Delete(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.ById{Id: "case-1"}
	cm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestCaseService_Publish(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.ById{Id: "case-1"}
	cm.On("Publish", req).Return(&m.Void{}, nil)

	_, err := svc.Publish(context.Background(), req)
	require.NoError(t, err)
}

func TestCaseService_GetByIdMobile(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.ByIdMobile{Id: "case-1"}
	want := &m.CaseMobileRes{}
	cm.On("GetByIdMobile", req).Return(want, nil)

	got, err := svc.GetByIdMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCaseService_GetAllMobile(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.CaseGetListMobileReq{}
	wantPag := &m.PaginationResponse{Total: 1}
	wantList := &m.CaseMobileList{}
	cm.On("GetAllMobile", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAllMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestCaseService_GetRandom(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := newCaseTestService(cm, nil)

	req := &m.CaseRandomReq{}
	want := &m.CaseMobileRes{}
	cm.On("GetRandom", req).Return(want, nil)

	got, err := svc.GetRandom(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
