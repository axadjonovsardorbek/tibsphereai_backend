package service

import (
	"context"
	"errors"
	"testing"

	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRoleTestService(rm *mocks.RoleMock) *RoleService {
	return NewRoleService(&st.Storage{RoleS: rm})
}

func TestRoleService_Create(t *testing.T) {
	rm := mocks.NewRoleMock()
	svc := newRoleTestService(rm)

	req := &m.RoleCreateReq{Name: "Kontent menejer", Slug: "content_manager"}
	want := &m.ById{Id: "role-1"}
	rm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestRoleService_GetById(t *testing.T) {
	rm := mocks.NewRoleMock()
	svc := newRoleTestService(rm)

	req := &m.ById{Id: "role-1"}
	want := &m.RoleRes{}
	rm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestRoleService_GetAll(t *testing.T) {
	rm := mocks.NewRoleMock()
	svc := newRoleTestService(rm)

	req := &m.RoleGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.RoleList{}
	rm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestRoleService_Update(t *testing.T) {
	rm := mocks.NewRoleMock()
	svc := newRoleTestService(rm)

	req := &m.RoleUpdateReq{Id: "role-1", Name: "Yangi nom"}
	rm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

// Update - tizim roli (superadmin) himoyalangan, storage xato qaytaradi, service uni tarqatadi.
func TestRoleService_Update_ProtectedSystemRole(t *testing.T) {
	rm := mocks.NewRoleMock()
	svc := newRoleTestService(rm)

	req := &m.RoleUpdateReq{Id: "role-superadmin", Name: "x"}
	wantErr := errors.New("role with id role-superadmin not found or is a protected system role")
	rm.On("Update", req).Return(nil, wantErr)

	_, err := svc.Update(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

func TestRoleService_Delete(t *testing.T) {
	rm := mocks.NewRoleMock()
	svc := newRoleTestService(rm)

	req := &m.ById{Id: "role-1"}
	rm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}
