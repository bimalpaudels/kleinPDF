import { useEffect } from "preact/hooks";
import { signal } from "@preact/signals";
import { GetStats } from "../../wailsjs/go/app/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import * as wailsModels from "../../wailsjs/go/models";
import { StatsUpdateEvent } from "../types/app";

// Global state for stats
export const stats = signal<wailsModels.app.AppStats>({
  session_files_compressed: 0,
  session_data_saved: 0,
  total_files_compressed: 0,
  total_data_saved: 0,
});

export const useStats = () => {
  const loadStats = async (): Promise<void> => {
    try {
      const currentStats: wailsModels.app.AppStats = await GetStats();
      if (currentStats) {
        stats.value = currentStats;
      }
    } catch (error) {
      console.log("Could not load stats");
    }
  };

  useEffect(() => {
    loadStats();

    // Set up event listener for real-time updates
    const unsubscribeStats = EventsOn(
      "stats:update",
      (data: StatsUpdateEvent) => {
        stats.value = data;
      }
    );

    return () => {
      unsubscribeStats();
    };
  }, []);

  return {
    stats,
    loadStats,
  };
};
