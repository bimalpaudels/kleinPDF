import { useState, useEffect } from "preact/hooks";
import { signal } from "@preact/signals";
import "./styles.css";

// Wails runtime imports
import {
  CompressPDF,
  ProcessFileData,
  GetPreferences,
  UpdatePreferences,
  OpenFileDialog,
  ShowSaveDialog,
  GetStats,
} from "../wailsjs/go/application/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

// Type imports
import * as wailsModels from "../wailsjs/go/models";
import {
  ProgressData,
  AdvancedOptions,
  CompressionLevel,
  CompressionOption,
  CompressionProgressEvent,
  StatsUpdateEvent,
} from "./types/app";

// Global state
const files = signal<wailsModels.transport.FileResult[]>([]);
const processing = signal<boolean>(false);
const progress = signal<ProgressData>({
  percent: 0,
  current: 0,
  total: 0,
  file: "",
});
const selectedCompressionLevel = signal<CompressionLevel>("good_enough");
// Note: Auto-download functionality removed from backend
const stats = signal<wailsModels.transport.AppStats>({
  session_files_compressed: 0,
  session_data_saved: 0,
  total_files_compressed: 0,
  total_data_saved: 0,
});

// Advanced options
const advancedOptions = signal<AdvancedOptions>({
  imageDpi: 150,
  imageQuality: 85,
  pdfVersion: "1.4",
  removeMetadata: false,
  embedFonts: true,
  generateThumbnails: false,
  convertToGrayscale: false,
});

function App() {
  const [dragOver, setDragOver] = useState<boolean>(false);

  useEffect(() => {
    // Load user preferences and stats on startup
    loadPreferences();
    loadStats();

    // Set up event listeners for real-time updates
    const unsubscribeProgress = EventsOn(
      "compression:progress",
      (data: CompressionProgressEvent) => {
        progress.value = data;
      }
    );

    const unsubscribeStats = EventsOn(
      "stats:update",
      (data: StatsUpdateEvent) => {
        stats.value = data;
      }
    );

    // Cleanup function
    return () => {
      unsubscribeProgress();
      unsubscribeStats();
    };
  }, []);

  const loadPreferences = async (): Promise<void> => {
    try {
      const prefs: wailsModels.models.UserPreferencesData =
        await GetPreferences();
      if (prefs) {
        selectedCompressionLevel.value =
          (prefs.default_compression_level as CompressionLevel) ||
          "good_enough";
        // Note: Auto-download and download folder properties removed from backend

        // Load advanced options
        advancedOptions.value = {
          imageDpi: prefs.image_dpi || 150,
          imageQuality: prefs.image_quality || 85,
          pdfVersion: prefs.pdf_version || "1.4",
          removeMetadata: prefs.remove_metadata || false,
          embedFonts: prefs.embed_fonts !== false, // default true
          generateThumbnails: prefs.generate_thumbnails || false,
          convertToGrayscale: prefs.convert_to_grayscale || false,
        };
      }
    } catch (error) {
      console.log("Could not load preferences, using defaults");
    }
  };

  const loadStats = async (): Promise<void> => {
    try {
      const currentStats: wailsModels.transport.AppStats = await GetStats();
      if (currentStats) {
        stats.value = currentStats;
      }
    } catch (error) {
      console.log("Could not load stats");
    }
  };

  const savePreferences = async (
    updates: Record<string, any>
  ): Promise<void> => {
    try {
      await UpdatePreferences(updates);
      console.log("Preferences saved");
    } catch (error) {
      console.error("Error saving preferences:", error);
    }
  };

  const handleDrop = async (e: DragEvent): Promise<void> => {
    e.preventDefault();
    e.stopPropagation();
    setDragOver(false);

    if (!e.dataTransfer?.files) return;

    const droppedFiles = Array.from(e.dataTransfer.files);
    const pdfFiles = droppedFiles.filter((file) =>
      file.name.toLowerCase().endsWith(".pdf")
    );

    if (pdfFiles.length === 0) {
      alert("Please drop PDF files only");
      return;
    }

    if (pdfFiles.length !== droppedFiles.length) {
      alert(
        `Only ${pdfFiles.length} PDF files were found. Non-PDF files were ignored.`
      );
    }

    // Handle drag & drop files (read file data)
    await handleDroppedFiles(pdfFiles);
  };

  const handleDragOver = (e: DragEvent): void => {
    e.preventDefault();
    e.stopPropagation();
    setDragOver(true);
  };

  const handleDragLeave = (e: DragEvent): void => {
    e.preventDefault();
    e.stopPropagation();
    // Only set dragOver to false if we're leaving the drop zone entirely
    const target = e.currentTarget as HTMLElement;
    const related = e.relatedTarget as HTMLElement;
    if (!target.contains(related)) {
      setDragOver(false);
    }
  };

  const handleFileSelect = async (e: Event): Promise<void> => {
    const target = e.target as HTMLInputElement;
    if (!target.files) return;

    const selectedFiles = Array.from(target.files);
    await handleDroppedFiles(selectedFiles);
    // Clear the input so the same files can be selected again
    target.value = "";
  };

  const handleDroppedFiles = async (fileList: File[]): Promise<void> => {
    if (processing.value) return;

    processing.value = true;
    progress.value = {
      percent: 0,
      current: 0,
      total: fileList.length,
      file: "Reading files...",
    };
    files.value = [];

    try {
      // Read all file data concurrently
      const fileDataPromises = fileList.map((file) => readFileData(file));
      const fileDataArray = await Promise.all(fileDataPromises);

      // Process files through Wails backend using file data
      const results: wailsModels.transport.CompressionResponse =
        await ProcessFileData(fileDataArray);

      if (results.success) {
        files.value = results.files;
      } else {
        throw new Error(results.error);
      }
    } catch (error) {
      console.error("Error compressing PDFs:", error);
      alert("Error compressing PDFs: " + (error as Error).message);
    } finally {
      processing.value = false;
      // Keep progress visible for a moment to show completion
      setTimeout(() => {
        progress.value = { percent: 0, current: 0, total: 0, file: "" };
      }, 2000);
    }
  };

  const readFileData = (
    file: File
  ): Promise<wailsModels.transport.FileUpload> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();

      reader.onload = (e) => {
        if (!e.target?.result) {
          reject(new Error(`Failed to read file: ${file.name}`));
          return;
        }

        const arrayBuffer = e.target.result as ArrayBuffer;
        const uint8Array = new Uint8Array(arrayBuffer);
        const dataArray = Array.from(uint8Array); // Convert to regular array for JSON serialization

        resolve(
          new wailsModels.transport.FileUpload({
            name: file.name,
            data: dataArray,
            size: file.size,
          })
        );
      };

      reader.onerror = () => {
        reject(new Error(`Failed to read file: ${file.name}`));
      };

      reader.readAsArrayBuffer(file);
    });
  };

  const handleFiles = async (filePaths: string[]): Promise<void> => {
    if (processing.value) return;

    processing.value = true;
    progress.value = {
      percent: 0,
      current: 0,
      total: filePaths.length,
      file: "Starting...",
    };
    files.value = [];

    try {
      // Process files through Wails backend using file paths (from file dialog)
      const compressionOptions = new wailsModels.services.CompressionOptions({
        image_dpi: advancedOptions.value.imageDpi,
        image_quality: advancedOptions.value.imageQuality,
        pdf_version: advancedOptions.value.pdfVersion,
        remove_metadata: advancedOptions.value.removeMetadata,
        embed_fonts: advancedOptions.value.embedFonts,
        generate_thumbnails: advancedOptions.value.generateThumbnails,
        convert_to_grayscale: advancedOptions.value.convertToGrayscale,
      });

      const compressionRequest = new wailsModels.transport.CompressionRequest({
        files: filePaths,
        compressionLevel: selectedCompressionLevel.value,
        advancedOptions: compressionOptions,
      });

      const results: wailsModels.transport.CompressionResponse =
        await CompressPDF(compressionRequest);

      if (results.success) {
        files.value = results.files;

        // Note: Auto-download functionality removed from backend
      } else {
        throw new Error(results.error);
      }
    } catch (error) {
      console.error("Error compressing PDFs:", error);
      alert("Error compressing PDFs: " + (error as Error).message);
    } finally {
      processing.value = false;
      // Keep progress visible for a moment to show completion
      setTimeout(() => {
        progress.value = { percent: 0, current: 0, total: 0, file: "" };
      }, 2000);
    }
  };

  const handleBrowseFiles = async (): Promise<void> => {
    try {
      const selectedFiles: string[] = await OpenFileDialog();
      if (selectedFiles && selectedFiles.length > 0) {
        handleFiles(selectedFiles);
      }
    } catch (error) {
      console.error("Error opening file dialog:", error);
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 Bytes";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const formatNumber = (num: number): string => {
    return new Intl.NumberFormat().format(num);
  };

  const compressionOptions: CompressionOption[] = [
    {
      level: "good_enough",
      icon: "✅",
      name: "Good Enough",
      desc: "Balanced compression, good quality",
    },
    {
      level: "aggressive",
      icon: "⚡",
      name: "Aggressive",
      desc: "Good compression, maintains quality",
    },
    { level: "ultra", icon: "🔥", name: "Ultra", desc: "Maximum compression" },
  ];

  return (
    <div className="min-h-screen bg-bg-primary text-text-primary font-nunito">
      {/* Enhanced Header with PDF-themed design */}
      <header className="relative overflow-hidden bg-gradient-to-br from-bg-primary to-bg-secondary">
        <div className="absolute inset-0 opacity-5 bg-[url('data:image/svg+xml,%3Csvg%20width=%2720%27%20height=%2720%27%20viewBox=%270%200%2020%2020%27%20xmlns=%27http://www.w3.org/2000/svg%27%3E%3Cg%20fill=%27%23dc2626%27%20fill-opacity=%270.1%27%3E%3Cpath%20d=%27M0%200h20v20H0z%27/%3E%3C/g%3E%3C/svg%3E')] bg-[length:20px_20px]"></div>

        <div className="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8 relative">
          <div className="flex flex-col sm:flex-row justify-between items-center gap-6 sm:gap-0">
            <div className="flex items-center">
              <div className="flex items-center gap-3 sm:gap-4">
                <div className="relative">
                  <div className="w-12 h-12 rounded-2xl flex items-center justify-center text-white text-xl font-bold shadow-lg bg-gradient-to-br from-pdf-red to-pdf-red-hover">
                    📄
                  </div>
                  <div className="absolute -top-1 -right-1 w-4 h-4 rounded-full animate-pulse bg-accent-secondary"></div>
                </div>
                <div>
                  <h1 className="text-3xl sm:text-4xl font-bold tracking-tight bg-gradient-to-r from-pdf-red to-accent-secondary bg-clip-text text-transparent">
                    KleinPDF
                  </h1>
                  <p className="text-sm font-medium opacity-75 mt-1 text-text-secondary">
                    Professional PDF Compression
                  </p>
                </div>
              </div>
            </div>

            <div className="flex items-center">
              <div className="backdrop-blur-xs rounded-2xl px-4 sm:px-6 py-3 sm:py-4 border border-border-default shadow-lg bg-bg-secondary/80">
                <div className="flex items-center gap-4 sm:gap-8">
                  <div className="text-center">
                    <div className="text-xl sm:text-2xl font-bold leading-tight text-pdf-red">
                      {formatNumber(stats.value.session_files_compressed)}
                    </div>
                    <div className="text-xs font-semibold uppercase tracking-wider mt-1 text-text-secondary">
                      Files Compressed
                    </div>
                  </div>
                  <div className="w-px h-8 sm:h-10 rounded-full bg-gradient-to-b from-transparent via-border-default to-transparent"></div>
                  <div className="text-center">
                    <div className="text-xl sm:text-2xl font-bold leading-tight text-pdf-red">
                      {formatBytes(stats.value.session_data_saved)}
                    </div>
                    <div className="text-xs font-semibold uppercase tracking-wider mt-1 text-text-secondary">
                      Data Saved
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_400px] gap-6 lg:gap-10 items-start">
          <section className="rounded-xl p-6 sm:p-10 border border-border-default shadow-xs bg-bg-secondary">
            <h2 className="text-xl font-semibold mb-2 text-text-primary">
              Compress PDF Files
            </h2>
            <p className="text-sm mb-8 text-text-secondary">
              Drag and drop your PDF files to compress them
            </p>

            <div
              className={`border-2 border-dashed rounded-xl py-12 sm:py-16 px-6 sm:px-10 text-center cursor-pointer relative drag-area ${
                dragOver ? "drag-over" : "bg-bg-tertiary border-border-light"
              } ${
                processing.value
                  ? "opacity-60 cursor-not-allowed pointer-events-none"
                  : ""
              }`}
              onDragOver={handleDragOver}
              onDragEnter={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
              onClick={() => !processing.value && handleBrowseFiles()}
            >
              <div
                className={`text-5xl mb-4 transition-all duration-200 ${
                  dragOver ? "text-pdf-red" : "text-text-secondary"
                }`}
              >
                📁
              </div>
              <div className="text-base font-medium mb-2 text-text-primary">
                {processing.value
                  ? "Processing..."
                  : "Drag & Drop your PDF files here"}
              </div>
              <div className="text-sm text-text-secondary">
                {processing.value
                  ? "Please wait..."
                  : "or click to browse files (multiple files supported)"}
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
              <div className="my-8 rounded-xl p-5 border border-border-default bg-bg-tertiary">
                <div className="flex justify-between items-center mb-3">
                  <div className="text-sm font-medium text-text-primary">
                    {progress.value.file && progress.value.file !== "Complete"
                      ? `Processing: ${progress.value.file}`
                      : `Processing files... (${progress.value.current}/${progress.value.total})`}
                  </div>
                  <div className="text-sm font-semibold text-pdf-red">
                    {Math.round(progress.value.percent)}%
                  </div>
                </div>
                <div className="w-full h-2 rounded-full overflow-hidden relative bg-border-default">
                  <div
                    className={`h-full transition-all duration-300 rounded-full relative overflow-hidden bg-gradient-to-r from-pdf-red to-accent-secondary w-[${progress.value.percent}%]`}
                  >
                    <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent progress-shine"></div>
                  </div>
                </div>
              </div>
            )}

            {files.value.length > 0 && (
              <div className="rounded-xl p-4 border border-border-default shadow-custom mt-6 w-full bg-bg-secondary">
                <h3 className="text-lg font-semibold mb-4 text-text-primary">
                  Compressed Files
                </h3>
                <div className="w-full">
                  {files.value.map((file, index) => (
                    <div
                      key={index}
                      className="flex flex-col sm:flex-row sm:justify-between sm:items-center py-3 px-3 rounded-sm mb-0.5 gap-3 border-b border-border-default hover:bg-gray-50 bg-bg-tertiary"
                    >
                      <div className="flex flex-col flex-1 min-w-0">
                        <div className="font-medium text-sm truncate text-text-primary">
                          {file.original_filename}
                        </div>
                        <div className="flex flex-col sm:flex-row sm:gap-3 sm:items-center text-xs mt-1 text-text-secondary">
                          <span className="whitespace-nowrap">
                            {formatBytes(file.original_size)} →{" "}
                            {formatBytes(file.compressed_size)}
                          </span>
                          <span className="font-bold text-text-secondary">
                            ({file.compression_ratio.toFixed(1)}% smaller)
                          </span>
                        </div>
                      </div>
                      <div className="flex gap-2 items-center sm:shrink-0">
                        <button
                          className="text-white border-none py-2 px-4 rounded-md text-xs font-semibold cursor-pointer btn-pdf-red"
                          onClick={() => downloadFile(file)}
                        >
                          Download
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </section>

          <aside className="rounded-xl p-6 sm:p-8 border border-border-default shadow-xs lg:sticky lg:top-5 bg-bg-secondary">
            <div className="mb-8">
              <h3 className="text-base font-semibold mb-4 text-text-primary">
                Compression Level
              </h3>
              <div className="flex flex-col gap-3">
                {compressionOptions.map((option) => (
                  <div
                    key={option.level}
                    className={`border-2 rounded-lg py-2.5 px-3 cursor-pointer transition-all duration-200 text-left ${
                      selectedCompressionLevel.value === option.level
                        ? "text-white bg-pdf-red border-pdf-red"
                        : "bg-bg-tertiary border-transparent hover:bg-gray-200"
                    }`}
                    onClick={() => {
                      selectedCompressionLevel.value = option.level;
                      savePreferences({
                        default_compression_level: option.level,
                      });
                    }}
                  >
                    <div className="flex items-center gap-2.5 mb-0.5">
                      <span className="text-base">{option.icon}</span>
                      <span className="font-semibold text-sm">
                        {option.name}
                      </span>
                    </div>
                    <div
                      className={`text-xs ml-7 ${
                        selectedCompressionLevel.value === option.level
                          ? "text-white/80"
                          : "text-text-secondary"
                      }`}
                    >
                      {option.desc}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Download Settings section removed - auto-download functionality not available in backend */}
          </aside>
        </div>
      </main>
    </div>
  );
}

const downloadFile = async (
  file: wailsModels.transport.FileResult
): Promise<void> => {
  try {
    // Note: saved_path property is not available in the generated model
    // Show save dialog
    const savePath: string = await ShowSaveDialog(file.compressed_filename);
    if (savePath) {
      // In a real implementation, we'd copy the file from temp to save location
      console.log("Would save file to:", savePath);
    }
  } catch (error) {
    console.error("Error handling file download:", error);
  }
};

export default App;
