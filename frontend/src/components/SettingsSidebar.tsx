import { CompressionLevelSelector } from "./CompressionLevelSelector";

export const SettingsSidebar = () => {
  return (
    <aside className="rounded-xl p-6 sm:p-8 border border-border-default shadow-xs lg:sticky lg:top-5 bg-bg-secondary">
      <CompressionLevelSelector />
      {/* Download Settings section removed - auto-download functionality not available in backend */}
    </aside>
  );
};
