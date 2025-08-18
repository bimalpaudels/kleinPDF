package application

import (
	"context"

	"kleinpdf/internal/config"
	"kleinpdf/internal/container"
	"kleinpdf/internal/database"
	"kleinpdf/internal/models"
	"kleinpdf/internal/transport"
)

type App struct {
	ctx       context.Context
	container *container.Container
	wailsApp  *transport.WailsApp
	config    *config.Config
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
	err = db.AutoMigrate(&models.UserPreferences{})
	if err != nil {
		cfg.Logger.Error("Failed to migrate database", "error", err)
		return
	}

	// Initialize dependency container
	a.container = container.New(ctx, cfg, db)
	
	// Initialize transport layer
	a.wailsApp = transport.NewWailsApp(
		ctx,
		a.container.GetCompressionService(),
		a.container.GetPreferencesRepository(),
		a.container.GetStatisticsService(),
	)

	cfg.Logger.Info("Wails app initialized successfully")
	cfg.Logger.Info("Application configuration", 
		"working_directory", cfg.WorkingDir,
		"database_path", cfg.DatabasePath,
		"ghostscript_available", true) // We'll get this from container later
}

func (a *App) CompressPDF(request transport.CompressionRequest) transport.CompressionResponse {
	return a.wailsApp.CompressPDF(request)
}

func (a *App) ProcessFileData(fileData []transport.FileUpload) transport.CompressionResponse {
	return a.wailsApp.ProcessFileData(fileData)
}

func (a *App) GetPreferences() (*models.UserPreferencesData, error) {
	return a.wailsApp.GetPreferences()
}

func (a *App) UpdatePreferences(data map[string]interface{}) error {
	return a.wailsApp.UpdatePreferences(data)
}

func (a *App) OpenFileDialog() ([]string, error) {
	return a.wailsApp.OpenFileDialog()
}

func (a *App) OpenDirectoryDialog() (string, error) {
	return a.wailsApp.OpenDirectoryDialog()
}

func (a *App) ShowSaveDialog(filename string) (string, error) {
	return a.wailsApp.ShowSaveDialog(filename)
}

func (a *App) OpenFile(filePath string) error {
	return a.wailsApp.OpenFile(filePath)
}

func (a *App) GetAppStatus() map[string]interface{} {
	return a.wailsApp.GetAppStatus()
}

func (a *App) GetStats() *transport.AppStats {
	return a.wailsApp.GetStats()
}