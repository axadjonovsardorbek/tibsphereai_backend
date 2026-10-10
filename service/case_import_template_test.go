package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shablon o'zi import qila olishi kerak (sarlavhalar va namuna qator to'g'ri o'qiladi).
func TestCaseImportTemplate_RoundTrips(t *testing.T) {
	data, err := CaseImportTemplateXLSX()
	require.NoError(t, err)

	rows, err := ParseCaseImportFile("cases-template.xlsx", data)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Ko'krak qafasidagi og'riq", rows[0]["title_uz"])
	assert.Equal(t, "medium", rows[0]["difficulty"])
	assert.Equal(t, "Kardiologiya", rows[0]["topic"])
}
