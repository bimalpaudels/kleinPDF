import { useState, useEffect } from "preact/hooks";
import { signal } from "@preact/signals";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import {
  CompressPDF,
  ProcessFileData,
  OpenFileDialog,
} from "../../wailsjs/go/app/App";
import * as wailsModels from "../../wailsjs/go/models";
import { ProgressData, CompressionProgressEvent } from "../types/app";
import { readFileData } from "../utils/fileUtils";
import { selectedCompressionLevel, advancedOptions } from "./usePreferences";

// Global state for file processing
export const files = signal<wailsModels.app.FileResult[]>([]);
export const processing = signal<boolean>(false);
export const progress = signal<ProgressData>({
  percent: 0,
  current: 0,
  total: 0,
  file: "",
});

export const useFileProcessing = () => {
  const [dragOver, setDragOver] = useState<boolean>(false);

  useEffect(() => {
    // Set up event listener for progress updates
    const unsubscribeProgress = EventsOn(
      "compression:progress",
      (data: CompressionProgressEvent) => {
        progress.value = data;
      }
    );

    return () => {
      unsubscribeProgress();
    };
  }, []);

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
      const fileDataPromises = fileList.map((file) => readFileData(file));
      const fileDataArray = await Promise.all(fileDataPromises);

      const results: wailsModels.app.CompressionResponse =
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
      setTimeout(() => {
        progress.value = { percent: 0, current: 0, total: 0, file: "" };
      }, 2000);
    }
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
      const compressionOptions = new wailsModels.compression.CompressionOptions(
        {
          image_dpi: advancedOptions.value.imageDpi,
          image_quality: advancedOptions.value.imageQuality,
          pdf_version: advancedOptions.value.pdfVersion,
          remove_metadata: advancedOptions.value.removeMetadata,
          embed_fonts: advancedOptions.value.embedFonts,
          generate_thumbnails: advancedOptions.value.generateThumbnails,
          convert_to_grayscale: advancedOptions.value.convertToGrayscale,
        }
      );

      const compressionRequest = new wailsModels.app.CompressionRequest({
        files: filePaths,
        compressionLevel: selectedCompressionLevel.value,
        advancedOptions: compressionOptions,
      });

      const results: wailsModels.app.CompressionResponse = await CompressPDF(
        compressionRequest
      );

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

  return {
    files,
    processing,
    progress,
    dragOver,
    setDragOver,
    handleDrop,
    handleDragOver,
    handleDragLeave,
    handleFileSelect,
    handleBrowseFiles,
  };
};
