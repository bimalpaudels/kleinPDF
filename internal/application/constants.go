package application

const (
	// Compression constants
	DefaultCompressionLevel = "good_enough"
	MaxConcurrencyLimit     = 8
	DefaultProgressPercent  = 20.0
	CompletedProgressPercent = 100.0
	
	// File operation constants
	DefaultFilePermissions = 0755
	
	// Event names
	EventFileProgress       = "file:progress"
	EventFileCompleted      = "file:completed"
	EventCompressionProgress = "compression:progress"
	EventStatsUpdate        = "stats:update"
)