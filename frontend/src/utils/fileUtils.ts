import * as wailsModels from "../../wailsjs/go/models";

export const readFileData = (
  file: File
): Promise<wailsModels.app.FileUpload> => {
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
        new wailsModels.app.FileUpload({
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

export const downloadFile = async (
  file: wailsModels.app.FileResult
): Promise<void> => {
  try {
    const { ShowSaveDialog } = await import("../../wailsjs/go/app/App");
    const savePath: string = await ShowSaveDialog(file.compressed_filename);
    if (savePath) {
      console.log("Would save file to:", savePath);
    }
  } catch (error) {
    console.error("Error handling file download:", error);
  }
};
