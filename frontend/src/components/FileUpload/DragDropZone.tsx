import { useFileProcessing } from "../../hooks/useFileProcessing";

export const DragDropZone = () => {
  const {
    processing,
    dragOver,
    handleDrop,
    handleDragOver,
    handleDragLeave,
    handleFileSelect,
    handleBrowseFiles,
  } = useFileProcessing();

  return (
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
        {processing.value ? "Processing..." : "Drag & Drop your PDF files here"}
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
  );
};
