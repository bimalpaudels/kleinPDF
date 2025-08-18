package application

import (
	"context"

	"kleinpdf/internal/services"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type StatsManager struct {
	ctx        context.Context
	stats      *AppStats
	pdfService *services.PDFService
}

func NewStatsManager(ctx context.Context, pdfService *services.PDFService) *StatsManager {
	return &StatsManager{
		ctx:        ctx,
		stats:      &AppStats{},
		pdfService: pdfService,
	}
}

func (m *StatsManager) UpdateStats(filesCompressed int, dataSaved int64) {
	m.stats.SessionFilesCompressed += filesCompressed
	m.stats.SessionDataSaved += dataSaved
	m.stats.TotalFilesCompressed += int64(filesCompressed)
	m.stats.TotalDataSaved += dataSaved

	// Emit stats update
	wailsruntime.EventsEmit(m.ctx, EventStatsUpdate, m.stats)
}

func (m *StatsManager) GetStats() *AppStats {
	return m.stats
}

func (m *StatsManager) GetAppStatus(workingDir string) map[string]interface{} {
	return map[string]interface{}{
		"status":                "running",
		"framework":             "Wails + Preact",
		"app_name":              "KleinPDF",
		"ghostscript_path":      m.pdfService.GetGhostscriptPath(),
		"ghostscript_available": m.pdfService.IsGhostscriptAvailable(),
		"working_directory":     workingDir,
	}
}