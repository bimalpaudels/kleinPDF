import {
  selectedCompressionLevel,
  savePreferences,
} from "../hooks/usePreferences";
import { CompressionOption } from "../types/app";

export const CompressionLevelSelector = () => {
  const compressionOptions: CompressionOption[] = [
    {
      level: "good_enough",
      icon: "✅",
      name: "Good Enough",
      desc: "Balanced compression, good quality",
    },
    {
      level: "aggressive",
      icon: "⚡",
      name: "Aggressive",
      desc: "Good compression, maintains quality",
    },
    { level: "ultra", icon: "🔥", name: "Ultra", desc: "Maximum compression" },
  ];

  return (
    <div className="mb-8">
      <h3 className="text-base font-semibold mb-4 text-text-primary">
        Compression Level
      </h3>
      <div className="flex flex-col gap-3">
        {compressionOptions.map((option) => (
          <div
            key={option.level}
            className={`border-2 rounded-lg py-2.5 px-3 cursor-pointer transition-all duration-200 text-left ${
              selectedCompressionLevel.value === option.level
                ? "text-white bg-pdf-red border-pdf-red"
                : "bg-bg-tertiary border-transparent hover:bg-gray-200"
            }`}
            onClick={() => {
              selectedCompressionLevel.value = option.level;
              savePreferences({
                default_compression_level: option.level,
              });
            }}
          >
            <div className="flex items-center gap-2.5 mb-0.5">
              <span className="text-base">{option.icon}</span>
              <span className="font-semibold text-sm">{option.name}</span>
            </div>
            <div
              className={`text-xs ml-7 ${
                selectedCompressionLevel.value === option.level
                  ? "text-white/80"
                  : "text-text-secondary"
              }`}
            >
              {option.desc}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
