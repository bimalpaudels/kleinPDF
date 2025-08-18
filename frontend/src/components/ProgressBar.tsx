import { progress } from "../hooks/useFileProcessing";

export const ProgressBar = () => {
  return (
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
  );
};
