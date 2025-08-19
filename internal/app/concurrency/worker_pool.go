package concurrency

import (
	"context"
	"path/filepath"
	"runtime"
	"sync"

	"kleinpdf/internal/common"
	"kleinpdf/internal/compression"
)

// NewWorkerPool creates a new worker pool instance
func NewWorkerPool(ctx context.Context, processor ProcessorFunc) *WorkerPool {
	return &WorkerPool{
		ctx:       ctx,
		processor: processor,
	}
}

// ProcessBatch processes a batch of files concurrently
func (wp *WorkerPool) ProcessBatch(request BatchRequest) BatchResult {
	if len(request.Files) == 0 {
		return BatchResult{
			Success: false,
			Error:   "no files provided",
		}
	}

	wp.totalFiles = len(request.Files)
	wp.maxWorkers = wp.calculateOptimalWorkerCount()

	// Create file work items with unique IDs
	var workItems []WorkItem
	for _, filePath := range request.Files {
		workItems = append(workItems, WorkItem{
			ID:       common.GenerateUUID(),
			FilePath: filePath,
		})
	}

	// Initialize channels
	wp.workChan = make(chan WorkItem, wp.totalFiles)
	wp.resultChan = make(chan *FileResult, wp.totalFiles)

	// Fill the work channel
	for _, work := range workItems {
		wp.workChan <- work
	}
	close(wp.workChan)

	// Start concurrent workers
	var wg sync.WaitGroup
	for i := 0; i < wp.maxWorkers && i < wp.totalFiles; i++ {
		wg.Add(1)
		go wp.worker(i, &wg, request.CompressionLevel, request.AdvancedOptions)
	}

	// Wait for all workers and close result channel
	go func() {
		wg.Wait()
		close(wp.resultChan)
	}()

	// Collect results
	return wp.collectResults()
}

// worker processes files from the work channel
func (wp *WorkerPool) worker(workerID int, wg *sync.WaitGroup, compressionLevel string, advancedOptions *compression.CompressionOptions) {
	defer wg.Done()

	for work := range wp.workChan {
		// Check for context cancellation
		select {
		case <-wp.ctx.Done():
			return
		default:
		}

		result, err := wp.processor(work.ID, work.FilePath, compressionLevel, advancedOptions, workerID)
		if err != nil {
			// Send error result
			errorResult := &FileResult{
				FileID:           work.ID,
				OriginalFilename: filepath.Base(work.FilePath),
				Status:           "error",
				Error:            err.Error(),
			}
			wp.resultChan <- errorResult
		} else {
			result.Status = "completed"
			wp.resultChan <- result
		}
	}
}

// calculateOptimalWorkerCount determines the optimal number of workers
func (wp *WorkerPool) calculateOptimalWorkerCount() int {
	maxConcurrency := runtime.NumCPU()
	if maxConcurrency > common.MaxConcurrencyLimit {
		maxConcurrency = common.MaxConcurrencyLimit
	}
	return maxConcurrency
}

// collectResults collects results from the result channel
func (wp *WorkerPool) collectResults() BatchResult {
	var results []FileResult
	var totalOriginalSize, totalCompressedSize int64
	completed := 0

	for result := range wp.resultChan {
		results = append(results, *result)
		if result.Status == "completed" {
			totalOriginalSize += result.OriginalSize
			totalCompressedSize += result.CompressedSize
		}
		completed++
	}

	// Calculate overall compression ratio
	var overallCompressionRatio float64
	if totalOriginalSize > 0 {
		overallCompressionRatio = float64(totalOriginalSize-totalCompressedSize) / float64(totalOriginalSize) * 100
	}

	return BatchResult{
		Success:                 true,
		Results:                 results,
		TotalFiles:              len(results),
		TotalOriginalSize:       totalOriginalSize,
		TotalCompressedSize:     totalCompressedSize,
		OverallCompressionRatio: overallCompressionRatio,
	}
}