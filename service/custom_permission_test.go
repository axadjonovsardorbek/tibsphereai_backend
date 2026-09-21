package service

import (
	"context"
	"testing"

	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCustomPermissionTestService(pm *mocks.CustomPermissionMock) *CustomPermissionService {
	return NewCustomPermissionService(&st.Storage{CustomPermissionS: pm})
}

func TestCustomPermissionService_Create(t *testing.T) {
	pm := mocks.NewCustomPermissionMock()
	svc := newCustomPermissionTestService(pm)

	req := &m.CustomPermissionCreateReq{Title: "Bo'limlar"}
	want := &m.ById{Id: "perm-1"}
	pm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCustomPermissionService_Update(t *testing.T) {
	pm := mocks.NewCustomPermissionMock()
	svc := newCustomPermissionTestService(pm)

	req := &m.CustomPermissionUpdateReq{Id: "perm-1", Title: "Bo'limlar v2"}
	pm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestCustomPermissionService_Delete(t *testing.T) {
	pm := mocks.NewCustomPermissionMock()
	svc := newCustomPermissionTestService(pm)

	req := &m.ById{Id: "perm-1"}
	pm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestCustomPermissionService_GetRoots(t *testing.T) {
	pm := mocks.NewCustomPermissionMock()
	svc := newCustomPermissionTestService(pm)

	want := &m.CustomPermissionList{Count: 2}
	pm.On("GetRoots").Return(want, nil)

	got, err := svc.GetRoots(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCustomPermissionService_GetChildren(t *testing.T) {
	pm := mocks.NewCustomPermissionMock()
	svc := newCustomPermissionTestService(pm)

	want := &m.CustomPermissionList{Count: 1}
	pm.On("GetChildren", "perm-1").Return(want, nil)

	got, err := svc.GetChildren(context.Background(), "perm-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// GetAccesses - ParentId bo'sh bo'lsa root, berilsa shu modulning bo'lim-modullari
// so'ralishi kerak (storage'ga aynan shu ikki argument uzatiladi).
func TestCustomPermissionService_GetAccesses_Root(t *testing.T) {
	pm := mocks.NewCustomPermissionMock()
	svc := newCustomPermissionTestService(pm)

	want := &m.RoleAccessList{Items: []m.RoleAccessItem{{CustomPermissionId: "perm-1", CanRead: true}}}
	pm.On("GetAccesses", "role-1", "").Return(want, nil)

	got, err := svc.GetAccesses(context.Background(), &m.RoleAccessGetReq{RoleId: "role-1"})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCustomPermissionService_GetAccesses_Children(t *testing.T) {
	pm := mocks.NewCustomPermissionMock()
	svc := newCustomPermissionTestService(pm)

	want := &m.RoleAccessList{}
	pm.On("GetAccesses", "role-1", "perm-1").Return(want, nil)

	got, err := svc.GetAccesses(context.Background(), &m.RoleAccessGetReq{RoleId: "role-1", ParentId: "perm-1"})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCustomPermissionService_UpsertAccess(t *testing.T) {
	pm := mocks.NewCustomPermissionMock()
	svc := newCustomPermissionTestService(pm)

	req := &m.RoleAccessUpsertReq{RoleId: "role-1", CustomPermissionId: "perm-1", CanRead: true, CanWrite: true}
	pm.On("UpsertAccess", req).Return(&m.Void{}, nil)

	_, err := svc.UpsertAccess(context.Background(), req)
	require.NoError(t, err)
}
