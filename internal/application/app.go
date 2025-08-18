package application

import (
	"context"

	"kleinpdf/internal/config"
	"kleinpdf/internal/database"
	model "kleinpdf/internal/models"
	"kleinpdf/internal/services"
)

type App struct {
	ctx context.Context

	// Handlers
	compressionHandler *CompressionHandler
	preferencesHandler *PreferencesHandler
	dialogsHandler     *DialogsHandler
	filesHandler       *FilesHandler
	statsManager       *StatsManager

	// Core services
	pdfService   *services.PDFService
	prefsService *services.PreferencesService
	config       *config.Config
}

func NewApp() *App {
	return &App{}
}

func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx

	// Initialize configuration
	cfg := config.New()
	a.config = cfg

	// Initialize database
	db, err := database.Initialize(cfg.DatabasePath)
	if err != nil {
		cfg.Logger.Error("Failed to initialize database", "error", err)
		return
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&model.UserPreferences{})
	if err != nil {
		cfg.Logger.Error("Failed to migrate database", "error", err)
		return
	}

	a.pdfService = services.NewPDFService(cfg)
	a.prefsService = services.NewPreferencesService(db)

	a.statsManager = NewStatsManager(ctx, a.pdfService)
	a.filesHandler = NewFilesHandler(ctx, cfg, a.prefsService)
	a.compressionHandler = NewCompressionHandler(ctx, cfg, a.pdfService, a.prefsService, a.filesHandler, a.statsManager)
	a.preferencesHandler = NewPreferencesHandler(a.prefsService)
	a.dialogsHandler = NewDialogsHandler(ctx)

	cfg.Logger.Info("Wails app initialized successfully")
	cfg.Logger.Info("Application configuration", 
		"working_directory", cfg.WorkingDir,
		"database_path", cfg.DatabasePath,
		"ghostscript_available", a.pdfService.IsGhostscriptAvailable())
}

func (a *App) CompressPDF(request CompressionRequest) CompressionResponse {
	return a.compressionHandler.CompressPDF(request)
}

func (a *App) ProcessFileData(fileData []FileUpload) CompressionResponse {
	return a.compressionHandler.ProcessFileData(fileData)
}

func (a *App) GetPreferences() (*model.UserPreferencesData, error) {
	return a.preferencesHandler.GetPreferences()
}

func (a *App) UpdatePreferences(data map[string]interface{}) error {
	return a.preferencesHandler.UpdatePreferences(data)
}

func (a *App) OpenFileDialog() ([]string, error) {
	return a.dialogsHandler.OpenFileDialog()
}

func (a *App) OpenDirectoryDialog() (string, error) {
	return a.dialogsHandler.OpenDirectoryDialog()
}

func (a *App) ShowSaveDialog(filename string) (string, error) {
	return a.dialogsHandler.ShowSaveDialog(filename)
}

func (a *App) OpenFile(filePath string) error {
	return a.dialogsHandler.OpenFile(filePath)
}

func (a *App) GetAppStatus() map[string]interface{} {
	return a.statsManager.GetAppStatus(a.config.WorkingDir)
}

func (a *App) GetStats() *AppStats {
	return a.statsManager.GetStats()
}