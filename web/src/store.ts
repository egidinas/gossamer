import { create } from "zustand";

type Route = "landing" | "profile" | "acceptance" | "command-center-fat" | "qualification" | "mission-map" | "supervisor" | "graph-wall" | "sources" | "requirements" | "commands" | "bus-tap" | "report" | "file-viewer";

interface AppState {
  route: Route;
  activeCampaign: string;
  setRoute: (route: Route) => void;
  setActiveCampaign: (id: string) => void;
}

export const useAppStore = create<AppState>((set) => ({
  route: "landing",
  activeCampaign: "thermal_acceptance_fat",
  setRoute: (route) => set({ route }),
  setActiveCampaign: (activeCampaign) => set({ activeCampaign }),
}));
