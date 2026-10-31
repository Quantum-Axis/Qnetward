import { create } from "zustand";
import { persist } from "zustand/middleware";

type ShellState = {
  /** Mobile drawer visibility. Not persisted. */
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  toggleSidebar: () => void;
  /** Desktop icon-only rail. Persisted per browser. */
  sidebarCollapsed: boolean;
  toggleSidebarCollapsed: () => void;
  /** Sidebar sections the user folded away. Persisted. */
  foldedSections: string[];
  toggleSection: (label: string) => void;
  /** Command palette (Ctrl/Cmd+K). */
  paletteOpen: boolean;
  setPaletteOpen: (open: boolean) => void;
};

export const useShellStore = create<ShellState>()(
  persist(
    (set) => ({
      sidebarOpen: false,
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
      sidebarCollapsed: false,
      toggleSidebarCollapsed: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      foldedSections: [],
      toggleSection: (label) =>
        set((state) => ({
          foldedSections: state.foldedSections.includes(label)
            ? state.foldedSections.filter((l) => l !== label)
            : [...state.foldedSections, label],
        })),
      paletteOpen: false,
      setPaletteOpen: (open) => set({ paletteOpen: open }),
    }),
    {
      name: "qnetward-shell",
      partialize: (state) => ({ sidebarCollapsed: state.sidebarCollapsed, foldedSections: state.foldedSections }),
    },
  ),
);
