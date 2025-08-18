package app

import (
	"kleinpdf/internal/database"
)

// GetPreferences gets the current user preferences
func (a *App) GetPreferences() (*database.UserPreferencesData, error) {
	return a.db.GetPreferences()
}

// UpdatePreferences updates user preferences
func (a *App) UpdatePreferences(data map[string]interface{}) error {
	return a.db.UpdatePreferences(data)
}