package service

import (
	"bytes"
	"context"
	"testing"

	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestParseCaseImportFile_CSV(t *testing.T) {
	csvData := "\xef\xbb\xbfTitle UZ;difficulty;hr\nBirinchi;hard;90\n;;\nIkkinchi;;\n"
	rows, err := ParseCaseImportFile("cases.csv", []byte(csvData))
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "Birinchi", rows[0]["title_uz"])
	assert.Equal(t, "hard", rows[0]["difficulty"])
	assert.Equal(t, "2", rows[0]["_row"])
	assert.Equal(t, "4", rows[1]["_row"]) // bo'sh qator o'tkazib yuborilgan, raqam fayldagi o'rniga mos
}

func TestParseCaseImportFile_XLSX(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	require.NoError(t, f.SetSheetRow(sheet, "A1", &[]string{"title_uz", "difficulty", "temp"}))
	require.NoError(t, f.SetSheetRow(sheet, "A2", &[]string{"Case 1", "easy", "36,6"}))
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))

	rows, err := ParseCaseImportFile("cases.xlsx", buf.Bytes())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Case 1", rows[0]["title_uz"])
}

func TestParseCaseImportFile_BadExtension(t *testing.T) {
	_, err := ParseCaseImportFile("cases.txt", []byte("x"))
	assert.Error(t, err)
}

func TestCaseService_ImportCases(t *testing.T) {
	cm := mocks.NewCaseMock()
	svc := &CaseService{storage: st.Storage{CaseS: cm}}

	cm.On("Create", mock.MatchedBy(func(r *m.CaseCreateReq) bool {
		return r.Title["uz"] == "A" && r.Difficulty == "medium" && r.ExpectedDurationMinutes == 10 && r.InitialVitals.Temp == 36.6
	})).Return(&m.ById{Id: "id-1"}, nil)
	cm.On("Publish", &m.ById{Id: "id-1"}).Return(&m.Void{}, nil)

	rows := []map[string]string{
		{"_row": "2", "title_uz": "A", "temp": "36,6"},
		{"_row": "3", "difficulty": "hard"},                        // title yo'q
		{"_row": "4", "title_uz": "B", "difficulty": "impossible"}, // yaroqsiz difficulty
	}
	res := svc.ImportCases(context.Background(), rows, true)

	assert.Equal(t, 3, res.Total)
	assert.Equal(t, 1, res.Created)
	assert.Equal(t, 1, res.Published)
	assert.Equal(t, 2, res.Failed)
	require.Len(t, res.Errors, 2)
	assert.Equal(t, 3, res.Errors[0].Row)
	assert.Equal(t, 4, res.Errors[1].Row)
}
