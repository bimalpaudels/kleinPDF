import { files } from "../hooks/useFileProcessing";
import { formatBytes } from "../utils/formatters";
import { downloadFile } from "../utils/fileUtils";
import * as wailsModels from "../../wailsjs/go/models";

export const FileList = () => {
  if (files.value.length === 0) return null;

  return (
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
  );
};
