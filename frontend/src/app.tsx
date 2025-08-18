import "./styles.css";

// Component imports
import { Header } from "./components/Header/Header";
import { FileUploadSection } from "./components/FileUpload/FileUploadSection";
import { SettingsSidebar } from "./components/CompressionSettings/SettingsSidebar";

// Hook imports
import { usePreferences } from "./hooks/usePreferences";
import { useStats } from "./hooks/useStats";
import { useFileProcessing } from "./hooks/useFileProcessing";

function App() {
  // Initialize hooks
  usePreferences();
  useStats();
  useFileProcessing();

  return (
    <div className="min-h-screen bg-bg-primary text-text-primary font-nunito">
      <Header />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 py-6 sm:py-8">
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_400px] gap-6 lg:gap-10 items-start">
          <FileUploadSection />
          <SettingsSidebar />
        </div>
      </main>
    </div>
  );
}

export default App;
