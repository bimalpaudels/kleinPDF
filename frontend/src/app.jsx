import { useState, useEffect } from 'preact/hooks'
import { signal } from '@preact/signals'
import './app.css'

// Wails runtime imports
import { CompressPDF, ProcessFileData, GetPreferences, UpdatePreferences, OpenFileDialog, OpenDirectoryDialog, ShowSaveDialog, OpenFile, GetStats } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

// Global state
const files = signal([])
const processing = signal(false)
const progress = signal({ percent: 0, current: 0, total: 0, file: '' })
const selectedCompressionLevel = signal('good_enough')
const autoDownload = signal(true)
const downloadFolder = signal('')
const stats = signal({ session_files_compressed: 0, session_data_saved: 0, total_files_compressed: 0, total_data_saved: 0 })

// Advanced options
const advancedOptions = signal({
  imageDpi: 150,
  imageQuality: 85,
  pdfVersion: '1.4',
  removeMetadata: false,
  embedFonts: true,
  generateThumbnails: false,
  convertToGrayscale: false
})

function App() {
  const [dragOver, setDragOver] = useState(false)

  useEffect(() => {
    // Load user preferences and stats on startup
    loadPreferences()
    loadStats()
    
    // Set up event listeners for real-time updates
    const unsubscribeProgress = EventsOn('compression:progress', (data) => {
      progress.value = data
    })
    
    const unsubscribeStats = EventsOn('stats:update', (data) => {
      stats.value = data
    })

    // Cleanup function
    return () => {
      unsubscribeProgress()
      unsubscribeStats()
    }
  }, [])

  const loadPreferences = async () => {
    try {
      const prefs = await GetPreferences()
      if (prefs) {
        selectedCompressionLevel.value = prefs.default_compression_level || 'good_enough'
        autoDownload.value = prefs.auto_download_enabled || true
        downloadFolder.value = prefs.default_download_folder || ''
        
        // Load advanced options
        advancedOptions.value = {
          imageDpi: prefs.image_dpi || 150,
          imageQuality: prefs.image_quality || 85,
          pdfVersion: prefs.pdf_version || '1.4',
          removeMetadata: prefs.remove_metadata || false,
          embedFonts: prefs.embed_fonts !== false, // default true
          generateThumbnails: prefs.generate_thumbnails || false,
          convertToGrayscale: prefs.convert_to_grayscale || false
        }
      }
    } catch (error) {
      console.log('Could not load preferences, using defaults')
    }
  }

  const loadStats = async () => {
    try {
      const currentStats = await GetStats()
      if (currentStats) {
        stats.value = currentStats
      }
    } catch (error) {
      console.log('Could not load stats')
    }
  }

  const savePreferences = async (updates) => {
    try {
      await UpdatePreferences(updates)
      console.log('Preferences saved')
    } catch (error) {
      console.error('Error saving preferences:', error)
    }
  }

  const handleDrop = async (e) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(false)
    
    const droppedFiles = Array.from(e.dataTransfer.files)
    const pdfFiles = droppedFiles.filter(file => file.name.toLowerCase().endsWith('.pdf'))
    
    if (pdfFiles.length === 0) {
      alert('Please drop PDF files only')
      return
    }
    
    if (pdfFiles.length !== droppedFiles.length) {
      alert(`Only ${pdfFiles.length} PDF files were found. Non-PDF files were ignored.`)
    }
    
    // Handle drag & drop files (read file data)
    await handleDroppedFiles(pdfFiles)
  }

  const handleDragOver = (e) => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(true)
  }

  const handleDragLeave = (e) => {
    e.preventDefault()
    e.stopPropagation()
    // Only set dragOver to false if we're leaving the drop zone entirely
    if (!e.currentTarget.contains(e.relatedTarget)) {
      setDragOver(false)
    }
  }

  const handleFileSelect = async (e) => {
    const selectedFiles = Array.from(e.target.files)
    await handleDroppedFiles(selectedFiles)
    // Clear the input so the same files can be selected again
    e.target.value = ''
  }

  const handleDroppedFiles = async (fileList) => {
    if (processing.value) return

    processing.value = true
    progress.value = { percent: 0, current: 0, total: fileList.length, file: 'Reading files...' }
    files.value = []

    try {
      // Read all file data concurrently
      const fileDataPromises = fileList.map(file => readFileData(file))
      const fileDataArray = await Promise.all(fileDataPromises)

      // Process files through Wails backend using file data
      const results = await ProcessFileData(fileDataArray)

      if (results.success) {
        files.value = results.files
      } else {
        throw new Error(results.error)
      }
    } catch (error) {
      console.error('Error compressing PDFs:', error)
      alert('Error compressing PDFs: ' + error.message)
    } finally {
      processing.value = false
      // Keep progress visible for a moment to show completion
      setTimeout(() => {
        progress.value = { percent: 0, current: 0, total: 0, file: '' }
      }, 2000)
    }
  }

  const readFileData = (file) => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader()
      
      reader.onload = (e) => {
        const arrayBuffer = e.target.result
        const uint8Array = new Uint8Array(arrayBuffer)
        const dataArray = Array.from(uint8Array)  // Convert to regular array for JSON serialization
        
        resolve({
          name: file.name,
          data: dataArray,
          size: file.size
        })
      }
      
      reader.onerror = () => {
        reject(new Error(`Failed to read file: ${file.name}`))
      }
      
      reader.readAsArrayBuffer(file)
    })
  }

  const handleFiles = async (filePaths) => {
    if (processing.value) return

    processing.value = true
    progress.value = { percent: 0, current: 0, total: filePaths.length, file: 'Starting...' }
    files.value = []

    try {
      // Process files through Wails backend using file paths (from file dialog)
      const results = await CompressPDF({
        files: filePaths,
        compressionLevel: selectedCompressionLevel.value,
        autoDownload: autoDownload.value,
        downloadFolder: downloadFolder.value,
        advancedOptions: advancedOptions.value
      })

      if (results.success) {
        files.value = results.files
        
        if (autoDownload.value && results.auto_download) {
          // Handle auto-download completed files
          console.log('Files automatically downloaded:', results.download_paths)
        }
      } else {
        throw new Error(results.error)
      }
    } catch (error) {
      console.error('Error compressing PDFs:', error)
      alert('Error compressing PDFs: ' + error.message)
    } finally {
      processing.value = false
      // Keep progress visible for a moment to show completion
      setTimeout(() => {
        progress.value = { percent: 0, current: 0, total: 0, file: '' }
      }, 2000)
    }
  }

  const handleBrowseFiles = async () => {
    try {
      const selectedFiles = await OpenFileDialog()
      if (selectedFiles && selectedFiles.length > 0) {
        handleFiles(selectedFiles)
      }
    } catch (error) {
      console.error('Error opening file dialog:', error)
    }
  }

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatNumber = (num) => {
    return new Intl.NumberFormat().format(num)
  }

  return (
    <div className="app-container">
      <header className="header">
        <div className="header-left">
          <h1 className="app-title">
            <div className="app-icon">📄</div>
            KleinPDF
          </h1>
        </div>
        
        <div className="header-right">
          <div className="stats-container">
            <div className="stat-item">
              <div className="stat-value">{formatNumber(stats.value.session_files_compressed)}</div>
              <div className="stat-label">Files Compressed</div>
            </div>
            <div className="stat-separator"></div>
            <div className="stat-item">
              <div className="stat-value">{formatBytes(stats.value.session_data_saved)}</div>
              <div className="stat-label">Data Saved</div>
            </div>
          </div>
        </div>
      </header>

      <main className="main-content">
        <section className="upload-section">
          <h2 className="section-title">Compress PDF Files</h2>
          <p className="section-subtitle">Drag and drop your PDF files to compress them</p>
          
          <div 
            className={`drop-zone ${dragOver ? 'dragover' : ''} ${processing.value ? 'disabled' : ''}`}
            onDragOver={handleDragOver}
            onDragEnter={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            onClick={() => !processing.value && handleBrowseFiles()}
          >
            <div className="drop-zone-icon">📁</div>
            <div className="drop-zone-text">
              {processing.value ? 'Processing...' : 'Drag & Drop your PDF files here'}
            </div>
            <div className="drop-zone-subtext">
              {processing.value ? 'Please wait...' : 'or click to browse files (multiple files supported)'}
            </div>
            <input 
              type="file" 
              id="fileInput" 
              multiple 
              accept=".pdf"
              style={{ display: 'none' }}
              onChange={handleFileSelect}
              disabled={processing.value}
            />
          </div>

          {processing.value && (
            <div className="progress-container">
              <div className="progress-info">
                <div className="progress-text">
                  {progress.value.file && progress.value.file !== 'Complete' ? 
                    `Processing: ${progress.value.file}` : 
                    `Processing files... (${progress.value.current}/${progress.value.total})`
                  }
                </div>
                <div className="progress-percent">{Math.round(progress.value.percent)}%</div>
              </div>
              <div className="progress-bar">
                <div className="progress-fill" style={{ width: `${progress.value.percent}%` }}></div>
              </div>
            </div>
          )}

          {files.value.length > 0 && (
            <div className="file-list-container">
              <h3 className="file-list-title">Compressed Files</h3>
              <div className="file-list">
                {files.value.map((file, index) => (
                  <div key={index} className="file-item">
                    <div className="file-info">
                      <div className="file-name">{file.original_filename}</div>
                      <div className="file-stats">
                        <span className="file-size">
                          {formatBytes(file.original_size)} → {formatBytes(file.compressed_size)}
                        </span>
                        <span className="file-ratio">({file.compression_ratio.toFixed(1)}% smaller)</span>
                      </div>
                    </div>
                    <div className="file-actions">
                      <button className="download-btn" onClick={() => downloadFile(file)}>
                        {file.saved_path ? 'Open' : 'Download'}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </section>

        <aside className="sidebar">
          <div className="compression-section">
            <h3 className="compression-title">Compression Level</h3>
            <div className="compression-options">
              {[
                { level: 'good_enough', icon: '✅', name: 'Good Enough', desc: 'Balanced compression, good quality' },
                { level: 'aggressive', icon: '⚡', name: 'Aggressive', desc: 'Good compression, maintains quality' },
                { level: 'ultra', icon: '🔥', name: 'Ultra', desc: 'Maximum compression' }
              ].map(option => (
                <div 
                  key={option.level}
                  className={`compression-option ${selectedCompressionLevel.value === option.level ? 'active' : ''}`}
                  onClick={() => {
                    selectedCompressionLevel.value = option.level
                    savePreferences({ default_compression_level: option.level })
                  }}
                >
                  <div className="option-header">
                    <span className="option-icon">{option.icon}</span>
                    <span className="option-name">{option.name}</span>
                  </div>
                  <div className="option-description">{option.desc}</div>
                </div>
              ))}
            </div>
          </div>

          <div className="auto-download-section">
            <h3 className="auto-download-title">Download Settings</h3>
            <label className="checkbox-container">
              <div 
                className={`checkbox ${autoDownload.value ? 'checked' : ''}`}
                onClick={() => {
                  autoDownload.value = !autoDownload.value
                  savePreferences({ auto_download_enabled: autoDownload.value })
                }}
              ></div>
              <div className="checkbox-text">Automatically download converted files</div>
            </label>
            
            {autoDownload.value && (
              <div className="download-folder-section">
                <label htmlFor="downloadFolder" className="folder-label">Download Folder</label>
                <div style={{ display: 'flex', gap: '8px' }}>
                  <input 
                    type="text" 
                    id="downloadFolder" 
                    value={downloadFolder.value}
                    onChange={(e) => {
                      downloadFolder.value = e.target.value
                      savePreferences({ default_download_folder: e.target.value })
                    }}
                    placeholder="Enter download folder path" 
                    className="folder-input"
                    style={{ flex: 1 }}
                  />
                  <button 
                    className="folder-browse-btn"
                    onClick={async () => {
                      try {
                        const selectedFolder = await OpenDirectoryDialog()
                        if (selectedFolder) {
                          downloadFolder.value = selectedFolder
                          savePreferences({ default_download_folder: selectedFolder })
                        }
                      } catch (error) {
                        console.error('Error opening directory dialog:', error)
                      }
                    }}
                  >
                    Browse
                  </button>
                </div>
              </div>
            )}
          </div>
        </aside>
      </main>
    </div>
  )
}

const downloadFile = async (file) => {
  try {
    if (file.saved_path) {
      // File is already saved, open it
      await OpenFile(file.saved_path)
    } else {
      // Show save dialog
      const savePath = await ShowSaveDialog(file.compressed_filename)
      if (savePath) {
        // In a real implementation, we'd copy the file from temp to save location
        console.log('Would save file to:', savePath)
      }
    }
  } catch (error) {
    console.error('Error handling file download:', error)
  }
}

export default App