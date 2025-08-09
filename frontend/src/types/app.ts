// Additional app-specific types

export interface ProgressData {
  percent: number;
  current: number;
  total: number;
  file: string;
}

export interface AdvancedOptions {
  imageDpi: number;
  imageQuality: number;
  pdfVersion: string;
  removeMetadata: boolean;
  embedFonts: boolean;
  generateThumbnails: boolean;
  convertToGrayscale: boolean;
}

export interface FileUploadData {
  name: string;
  data: number[];
  size: number;
}

export type CompressionLevel = 'good_enough' | 'aggressive' | 'ultra';

export interface CompressionOption {
  level: CompressionLevel;
  icon: string;
  name: string;
  desc: string;
}

// Event types for Wails runtime
export interface CompressionProgressEvent {
  percent: number;
  current: number;
  total: number;
  file: string;
}

export interface StatsUpdateEvent {
  session_files_compressed: number;
  session_data_saved: number;
  total_files_compressed: number;
  total_data_saved: number;
}