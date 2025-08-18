package statistics

// AppStats represents application usage statistics
type AppStats struct {
	TotalFilesCompressed   int64 `json:"total_files_compressed"`
	TotalDataSaved         int64 `json:"total_data_saved"`
	SessionFilesCompressed int   `json:"session_files_compressed"`
	SessionDataSaved       int64 `json:"session_data_saved"`
}

// Service defines the interface for statistics operations
type Service interface {
	UpdateStats(filesCompressed int, dataSaved int64)
	GetStats() *AppStats
	GetAppStatus(workingDir string) map[string]interface{}
}