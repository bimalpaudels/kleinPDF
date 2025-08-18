package application

import (
	"kleinpdf/internal/models"
	"kleinpdf/internal/services"
)

type PreferencesHandler struct {
	prefsService *services.PreferencesService
}

func NewPreferencesHandler(prefsService *services.PreferencesService) *PreferencesHandler {
	return &PreferencesHandler{
		prefsService: prefsService,
	}
}

func (h *PreferencesHandler) GetPreferences() (*models.UserPreferencesData, error) {
	return h.prefsService.GetPreferences()
}

func (h *PreferencesHandler) UpdatePreferences(data map[string]interface{}) error {
	return h.prefsService.UpdatePreferences(data)
}