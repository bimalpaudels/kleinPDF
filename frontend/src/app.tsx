import { useState, useEffect } from 'preact/hooks'
import { signal } from '@preact/signals'
import './styles.css'

// Wails runtime imports
import { 
  CompressPDF, 
  ProcessFileData, 
  GetPreferences, 
  UpdatePreferences, 
  OpenFileDialog, 
  OpenDirectoryDialog, 
  ShowSaveDialog, 
  OpenFile, 
  GetStats 
} from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

// Type imports
import { main, models, services } from '../wailsjs/go/models'
import { 
  ProgressData, 
  AdvancedOptions, 
  FileUploadData, 
  CompressionLevel, 
  CompressionOption,
  CompressionProgressEvent,
  StatsUpdateEvent
} from './types/app'

// Global state
const files = signal<main.FileResult[]>([])
const processing = signal<boolean>(false)
const progress = signal<ProgressData>({ percent: 0, current: 0, total: 0, file: '' })
const selectedCompressionLevel = signal<CompressionLevel>('good_enough')
const autoDownload = signal<boolean>(true)
const downloadFolder = signal<string>('')
const stats = signal<main.AppStats>({ 
  session_files_compressed: 0, 
  session_data_saved: 0, 
  total_files_compressed: 0, 
  total_data_saved: 0 
})

// Advanced options
const advancedOptions = signal<AdvancedOptions>({
  imageDpi: 150,
  imageQuality: 85,
  pdfVersion: '1.4',
  removeMetadata: false,
  embedFonts: true,
  generateThumbnails: false,
  convertToGrayscale: false
})

function App() {
  const [dragOver, setDragOver] = useState<boolean>(false)

  useEffect(() => {
    // Load user preferences and stats on startup
    loadPreferences()
    loadStats()
    
    // Set up event listeners for real-time updates
    const unsubscribeProgress = EventsOn('compression:progress', (data: CompressionProgressEvent) => {
      progress.value = data
    })
    
    const unsubscribeStats = EventsOn('stats:update', (data: StatsUpdateEvent) => {
      stats.value = data
    })

    // Cleanup function
    return () => {
      unsubscribeProgress()
      unsubscribeStats()
    }
  }, [])

  const loadPreferences = async (): Promise<void> => {
    try {
      const prefs: models.UserPreferencesData = await GetPreferences()
      if (prefs) {
        selectedCompressionLevel.value = (prefs.default_compression_level as CompressionLevel) || 'good_enough'
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

  const loadStats = async (): Promise<void> => {
    try {
      const currentStats: main.AppStats = await GetStats()
      if (currentStats) {
        stats.value = currentStats
      }
    } catch (error) {
      console.log('Could not load stats')
    }
  }

  const savePreferences = async (updates: Record<string, any>): Promise<void> => {
    try {
      await UpdatePreferences(updates)
      console.log('Preferences saved')
    } catch (error) {
      console.error('Error saving preferences:', error)
    }
  }

  const handleDrop = async (e: DragEvent): Promise<void> => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(false)
    
    if (!e.dataTransfer?.files) return
    
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

  const handleDragOver = (e: DragEvent): void => {
    e.preventDefault()
    e.stopPropagation()
    setDragOver(true)
  }

  const handleDragLeave = (e: DragEvent): void => {
    e.preventDefault()
    e.stopPropagation()
    // Only set dragOver to false if we're leaving the drop zone entirely
    const target = e.currentTarget as HTMLElement
    const related = e.relatedTarget as HTMLElement
    if (!target.contains(related)) {
      setDragOver(false)
    }
  }

  const handleFileSelect = async (e: Event): Promise<void> => {
    const target = e.target as HTMLInputElement
    if (!target.files) return
    
    const selectedFiles = Array.from(target.files)
    await handleDroppedFiles(selectedFiles)
    // Clear the input so the same files can be selected again
    target.value = ''
  }

  const handleDroppedFiles = async (fileList: File[]): Promise<void> => {
    if (processing.value) return

    processing.value = true
    progress.value = { percent: 0, current: 0, total: fileList.length, file: 'Reading files...' }
    files.value = []

    try {
      // Read all file data concurrently
      const fileDataPromises = fileList.map(file => readFileData(file))
      const fileDataArray = await Promise.all(fileDataPromises)

      // Process files through Wails backend using file data
      const results: main.CompressionResponse = await ProcessFileData(fileDataArray)

      if (results.success) {
        files.value = results.files
      } else {
        throw new Error(results.error)
      }
    } catch (error) {
      console.error('Error compressing PDFs:', error)
      alert('Error compressing PDFs: ' + (error as Error).message)
    } finally {
      processing.value = false
      // Keep progress visible for a moment to show completion
      setTimeout(() => {
        progress.value = { percent: 0, current: 0, total: 0, file: '' }
      }, 2000)
    }
  }

  const readFileData = (file: File): Promise<FileUploadData> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader()
      
      reader.onload = (e) => {
        if (!e.target?.result) {
          reject(new Error(`Failed to read file: ${file.name}`))
          return
        }
        
        const arrayBuffer = e.target.result as ArrayBuffer
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

  const handleFiles = async (filePaths: string[]): Promise<void> => {
    if (processing.value) return

    processing.value = true
    progress.value = { percent: 0, current: 0, total: filePaths.length, file: 'Starting...' }
    files.value = []

    try {
      // Process files through Wails backend using file paths (from file dialog)
      const compressionOptions = new services.CompressionOptions({
        image_dpi: advancedOptions.value.imageDpi,
        image_quality: advancedOptions.value.imageQuality,
        pdf_version: advancedOptions.value.pdfVersion,
        remove_metadata: advancedOptions.value.removeMetadata,
        embed_fonts: advancedOptions.value.embedFonts,
        generate_thumbnails: advancedOptions.value.generateThumbnails,
        convert_to_grayscale: advancedOptions.value.convertToGrayscale
      })

      const compressionRequest = new main.CompressionRequest({
        files: filePaths,
        compressionLevel: selectedCompressionLevel.value,
        autoDownload: autoDownload.value,
        downloadFolder: downloadFolder.value,
        advancedOptions: compressionOptions
      })

      const results: main.CompressionResponse = await CompressPDF(compressionRequest)

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
      alert('Error compressing PDFs: ' + (error as Error).message)
    } finally {
      processing.value = false
      // Keep progress visible for a moment to show completion
      setTimeout(() => {
        progress.value = { percent: 0, current: 0, total: 0, file: '' }
      }, 2000)
    }
  }

  const handleBrowseFiles = async (): Promise<void> => {
    try {
      const selectedFiles: string[] = await OpenFileDialog()
      if (selectedFiles && selectedFiles.length > 0) {
        handleFiles(selectedFiles)
      }
    } catch (error) {
      console.error('Error opening file dialog:', error)
    }
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatNumber = (num: number): string => {
    return new Intl.NumberFormat().format(num)
  }

  const compressionOptions: CompressionOption[] = [
    { level: 'good_enough', icon: '✅', name: 'Good Enough', desc: 'Balanced compression, good quality' },
    { level: 'aggressive', icon: '⚡', name: 'Aggressive', desc: 'Good compression, maintains quality' },
    { level: 'ultra', icon: '🔥', name: 'Ultra', desc: 'Maximum compression' }
  ]

  return (
    <div className="max-w-7xl mx-auto p-5 min-h-screen" style={{ background: 'var(--bg-primary)', color: 'var(--text-primary)' }}>
      <header className="flex justify-between items-center py-5 border-b border-solid mb-10" style={{ borderColor: 'var(--border-color)' }}>
        <div className="flex items-center">
          <h1 className="text-3xl font-semibold flex items-center gap-3" style={{ color: 'var(--text-primary)' }}>
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white text-base font-semibold">
              📄
            </div>
            KleinPDF
          </h1>
        </div>
        
        <div className="flex items-center">
          <div className="flex items-center gap-6 rounded-xl px-5 py-3 border shadow-sm" style={{ background: 'var(--bg-secondary)', borderColor: 'var(--border-color)', boxShadow: '0 1px 3px var(--shadow)' }}>
            <div className="text-center">
              <div className="text-xl font-bold leading-tight" style={{ color: 'var(--accent-blue)' }}>{formatNumber(stats.value.session_files_compressed)}</div>
              <div className="text-xs font-medium uppercase tracking-wide mt-0.5" style={{ color: 'var(--text-secondary)' }}>Files Compressed</div>
            </div>
            <div className="w-px h-8" style={{ background: 'var(--border-color)' }}></div>
            <div className="text-center">
              <div className="text-xl font-bold leading-tight" style={{ color: 'var(--accent-blue)' }}>{formatBytes(stats.value.session_data_saved)}</div>
              <div className="text-xs font-medium uppercase tracking-wide mt-0.5" style={{ color: 'var(--text-secondary)' }}>Data Saved</div>
            </div>
          </div>
        </div>
      </header>

      <main className="grid grid-cols-1 lg:grid-cols-[1fr_400px] gap-10 items-start">
        <section className="rounded-xl p-10 border shadow-sm" style={{ background: 'var(--bg-secondary)', borderColor: 'var(--border-color)', boxShadow: '0 1px 3px var(--shadow)' }}>
          <h2 className="text-xl font-semibold mb-2" style={{ color: 'var(--text-primary)' }}>Compress PDF Files</h2>
          <p className="text-sm mb-8" style={{ color: 'var(--text-secondary)' }}>Drag and drop your PDF files to compress them</p>
          
          <div 
            className={`border-2 border-dashed rounded-xl py-16 px-10 text-center transition-all duration-200 cursor-pointer relative ${
              dragOver ? 'scale-105 shadow-lg border-blue-500' : 'border-gray-300 hover:border-blue-500 hover:bg-white'
            } ${processing.value ? 'opacity-60 cursor-not-allowed pointer-events-none' : ''}`}
            style={{ 
              background: dragOver ? 'var(--bg-secondary)' : 'var(--bg-tertiary)',
              borderColor: dragOver ? 'var(--accent-blue)' : 'var(--border-light)',
              boxShadow: dragOver ? '0 0 0 2px var(--accent-blue)' : 'none'
            }}
            onDragOver={handleDragOver}
            onDragEnter={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            onClick={() => !processing.value && handleBrowseFiles()}
          >
            <div className={`text-5xl mb-4 transition-all duration-200 ${dragOver ? 'text-blue-500' : 'text-gray-400'}`} style={{ color: dragOver ? 'var(--accent-blue)' : 'var(--text-secondary)' }}>📁</div>
            <div className="text-base font-medium mb-2" style={{ color: 'var(--text-primary)' }}>
              {processing.value ? 'Processing...' : 'Drag & Drop your PDF files here'}
            </div>
            <div className="text-sm" style={{ color: 'var(--text-secondary)' }}>
              {processing.value ? 'Please wait...' : 'or click to browse files (multiple files supported)'}
            </div>
            <input 
              type="file" 
              id="fileInput" 
              multiple 
              accept=".pdf"
              className="hidden"
              onChange={handleFileSelect}
              disabled={processing.value}
            />
          </div>

          {processing.value && (
            <div className="my-8 rounded-xl p-5 border" style={{ background: 'var(--bg-tertiary)', borderColor: 'var(--border-color)' }}>
              <div className="flex justify-between items-center mb-3">
                <div className="text-sm font-medium" style={{ color: 'var(--text-primary)' }}>
                  {progress.value.file && progress.value.file !== 'Complete' ? 
                    `Processing: ${progress.value.file}` : 
                    `Processing files... (${progress.value.current}/${progress.value.total})`
                  }
                </div>
                <div className="text-sm font-semibold" style={{ color: 'var(--accent-blue)' }}>{Math.round(progress.value.percent)}%</div>
              </div>
              <div className="w-full h-2 rounded-full overflow-hidden relative" style={{ background: 'var(--border-color)' }}>
                <div 
                  className="h-full transition-all duration-300 rounded-full relative overflow-hidden bg-gradient-to-r from-blue-500 to-purple-600" 
                  style={{ width: `${progress.value.percent}%` }}
                >
                  <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent progress-shine"></div>
                </div>
              </div>
            </div>
          )}

          {files.value.length > 0 && (
            <div className="rounded-xl p-4 border shadow-sm mt-6 w-full" style={{ background: 'var(--bg-secondary)', borderColor: 'var(--border-color)', boxShadow: '0 1px 3px var(--shadow)' }}>
              <h3 className="text-lg font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>Compressed Files</h3>
              <div className="w-full">
                {files.value.map((file, index) => (
                  <div key={index} className="flex justify-between items-center py-2 px-3 rounded mb-0.5 gap-3 border-b hover:bg-gray-50" style={{ background: 'var(--bg-tertiary)', borderColor: 'var(--border-color)' }}>
                    <div className="flex flex-col flex-1 min-w-32 max-w-52">
                      <div className="font-medium text-sm truncate" style={{ color: 'var(--text-primary)' }}>{file.original_filename}</div>
                      <div className="flex gap-3 items-center text-xs whitespace-nowrap" style={{ color: 'var(--text-secondary)' }}>
                        <span>{formatBytes(file.original_size)} → {formatBytes(file.compressed_size)}</span>
                        <span className="font-bold" style={{ color: 'var(--text-secondary)' }}>({file.compression_ratio.toFixed(1)}% smaller)</span>
                      </div>
                    </div>
                    <div className="flex gap-2 items-center">
                      <button 
                        className="bg-blue-600 text-white border-none py-2 px-4 rounded-md text-xs font-semibold cursor-pointer transition-all duration-200 hover:bg-blue-700 hover:-translate-y-px"
                        onClick={() => downloadFile(file)}
                      >
                        {file.saved_path ? 'Open' : 'Download'}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </section>

        <aside className="rounded-xl p-8 border shadow-sm sticky top-5" style={{ background: 'var(--bg-secondary)', borderColor: 'var(--border-color)', boxShadow: '0 1px 3px var(--shadow)' }}>
          <div className="mb-8">
            <h3 className="text-base font-semibold mb-4" style={{ color: 'var(--text-primary)' }}>Compression Level</h3>
            <div className="flex flex-col gap-3">
              {compressionOptions.map(option => (
                <div 
                  key={option.level}
                  className={`border-2 border-transparent rounded-lg py-2.5 px-3 cursor-pointer transition-all duration-200 text-left ${
                    selectedCompressionLevel.value === option.level ? 'text-white' : 'hover:bg-gray-200'
                  }`}
                  style={{ 
                    background: selectedCompressionLevel.value === option.level ? 'var(--accent-blue)' : 'var(--bg-tertiary)',
                    borderColor: selectedCompressionLevel.value === option.level ? 'var(--accent-blue)' : 'transparent'
                  }}
                  onClick={() => {
                    selectedCompressionLevel.value = option.level
                    savePreferences({ default_compression_level: option.level })
                  }}
                >
                  <div className="flex items-center gap-2.5 mb-0.5">
                    <span className="text-base">{option.icon}</span>
                    <span className="font-semibold text-sm">{option.name}</span>
                  </div>
                  <div 
                    className={`text-xs ml-7 ${
                      selectedCompressionLevel.value === option.level ? 'text-white/80' : ''
                    }`}
                    style={{ color: selectedCompressionLevel.value === option.level ? 'rgba(255, 255, 255, 0.8)' : 'var(--text-secondary)' }}
                  >
                    {option.desc}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="pt-6 border-t" style={{ borderColor: 'var(--border-color)' }}>
            <h3 className="text-base font-semibold mb-3" style={{ color: 'var(--text-primary)' }}>Download Settings</h3>
            <label className="flex items-start gap-3 cursor-pointer">
              <div 
                className={`w-5 h-5 border-2 rounded flex items-center justify-center transition-all duration-200 flex-shrink-0 mt-0.5 ${
                  autoDownload.value ? 'bg-blue-600 border-blue-600' : 'bg-white border-gray-300'
                }`}
                onClick={() => {
                  autoDownload.value = !autoDownload.value
                  savePreferences({ auto_download_enabled: autoDownload.value })
                }}
              >
                {autoDownload.value && <span className="text-white text-xs font-bold">✓</span>}
              </div>
              <div className="text-sm leading-tight" style={{ color: 'var(--text-primary)' }}>Automatically download converted files</div>
            </label>
            
            {autoDownload.value && (
              <div className="mt-4 pt-4 border-t" style={{ borderColor: 'var(--border-color)' }}>
                <label className="block mb-2 font-semibold text-sm" style={{ color: 'var(--text-primary)' }}>Download Folder</label>
                <div className="flex gap-2">
                  <input 
                    type="text" 
                    value={downloadFolder.value}
                    onChange={(e) => {
                      const target = e.target as HTMLInputElement
                      downloadFolder.value = target.value
                      savePreferences({ default_download_folder: target.value })
                    }}
                    placeholder="Enter download folder path" 
                    className="flex-1 py-2 px-3 border-2 rounded-md text-xs transition-all duration-200 focus:outline-none focus:border-blue-500 font-mono"
                    style={{ 
                      background: 'var(--bg-secondary)', 
                      color: 'var(--text-primary)', 
                      borderColor: 'var(--border-color)'
                    }}
                  />
                  <button 
                    className="bg-blue-600 text-white border-none py-2 px-3 rounded-md text-xs font-semibold cursor-pointer transition-all duration-200 whitespace-nowrap hover:bg-blue-700 hover:-translate-y-px"
                    onClick={async () => {
                      try {
                        const selectedFolder: string = await OpenDirectoryDialog()
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

const downloadFile = async (file: main.FileResult): Promise<void> => {
  try {
    if (file.saved_path) {
      // File is already saved, open it
      await OpenFile(file.saved_path)
    } else {
      // Show save dialog
      const savePath: string = await ShowSaveDialog(file.compressed_filename)
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