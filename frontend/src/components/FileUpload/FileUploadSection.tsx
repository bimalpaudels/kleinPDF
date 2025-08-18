import { processing } from "../../hooks/useFileProcessing";
import { DragDropZone } from "./DragDropZone";
import { ProgressBar } from "./ProgressBar";
import { FileList } from "./FileList";

export const FileUploadSection = () => {
  return (
    <section className="rounded-xl p-6 sm:p-10 border border-border-default shadow-xs bg-bg-secondary">
      <h2 className="text-xl font-semibold mb-2 text-text-primary">
        Compress PDF Files
      </h2>
      <p className="text-sm mb-8 text-text-secondary">
        Drag and drop your PDF files to compress them
      </p>

      <DragDropZone />

      {processing.value && <ProgressBar />}

      <FileList />
    </section>
  );
};
