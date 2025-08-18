package container

import (
	"context"
	"log/slog"

	"kleinpdf/internal/config"
	compressionDomain "kleinpdf/internal/domain/compression"
	preferencesDomain "kleinpdf/internal/domain/preferences"
	statisticsDomain "kleinpdf/internal/domain/statistics"
	"kleinpdf/internal/services"
	
	"gorm.io/gorm"
)

// Container holds all dependencies for the application
type Container struct {
	config   *config.Config
	db       *gorm.DB
	logger   *slog.Logger
	
	// Services
	pdfProcessor        compressionDomain.PDFProcessor
	preferencesRepo     preferencesDomain.Repository
	compressionService  compressionDomain.Service
	statisticsService   statisticsDomain.Service
}

// New creates a new dependency injection container
func New(ctx context.Context, cfg *config.Config, db *gorm.DB) *Container {
	c := &Container{
		config: cfg,
		db:     db,
		logger: cfg.Logger,
	}
	
	c.initServices(ctx)
	return c
}

// initServices initializes all services with their dependencies
func (c *Container) initServices(ctx context.Context) {
	// Create infrastructure services
	c.pdfProcessor = &PDFProcessorAdapter{service: services.NewPDFService(c.config)}
	c.preferencesRepo = &PreferencesRepositoryAdapter{service: services.NewPreferencesService(c.db)}
	
	// Create domain services
	c.compressionService = &CompressionServiceImpl{
		processor:    c.pdfProcessor,
		prefsRepo:    c.preferencesRepo,
		config:       c.config,
		ctx:          ctx,
	}
	
	c.statisticsService = &StatisticsServiceImpl{
		processor: c.pdfProcessor,
		ctx:       ctx,
	}
}

// GetCompressionService returns the compression service
func (c *Container) GetCompressionService() compressionDomain.Service {
	return c.compressionService
}

// GetStatisticsService returns the statistics service  
func (c *Container) GetStatisticsService() statisticsDomain.Service {
	return c.statisticsService
}

// GetPreferencesRepository returns the preferences repository
func (c *Container) GetPreferencesRepository() preferencesDomain.Repository {
	return c.preferencesRepo
}

// GetConfig returns the application configuration
func (c *Container) GetConfig() *config.Config {
	return c.config
}