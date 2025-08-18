import { useEffect } from "preact/hooks";
import { signal } from "@preact/signals";
import { GetPreferences, UpdatePreferences } from "../../wailsjs/go/app/App";
import * as wailsModels from "../../wailsjs/go/models";
import { CompressionLevel, AdvancedOptions } from "../types/app";

// Global state for preferences
export const selectedCompressionLevel = signal<CompressionLevel>("good_enough");
export const advancedOptions = signal<AdvancedOptions>({
  imageDpi: 150,
  imageQuality: 85,
  pdfVersion: "1.4",
  removeMetadata: false,
  embedFonts: true,
  generateThumbnails: false,
  convertToGrayscale: false,
});

export const usePreferences = () => {
  const loadPreferences = async (): Promise<void> => {
    try {
      const prefs: wailsModels.database.UserPreferencesData =
        await GetPreferences();
      if (prefs) {
        selectedCompressionLevel.value =
          (prefs.default_compression_level as CompressionLevel) ||
          "good_enough";

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

  useEffect(() => {
    loadPreferences();
  }, []);

  return {
    selectedCompressionLevel,
    advancedOptions,
    loadPreferences,
    savePreferences,
  };
};

// Export savePreferences function for direct use
export const savePreferences = async (
  updates: Record<string, any>
): Promise<void> => {
  try {
    await UpdatePreferences(updates);
    console.log("Preferences saved");
  } catch (error) {
    console.error("Error saving preferences:", error);
  }
};
