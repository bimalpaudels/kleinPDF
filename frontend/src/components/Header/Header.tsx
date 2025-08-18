import { stats } from "../../hooks/useStats";
import { formatBytes, formatNumber } from "../../utils/formatters";

export const Header = () => {
  return (
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
  );
};
