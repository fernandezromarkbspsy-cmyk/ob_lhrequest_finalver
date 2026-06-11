import { create } from "zustand";

type SessionUser = {
  name: string;
  role: string;
};

type AppState = {
  user: SessionUser | null;
  queue: "all" | "ops" | "mm" | "dock";
  setUser: (user: SessionUser | null) => void;
  setQueue: (queue: AppState["queue"]) => void;
};

export const useAppStore = create<AppState>((set) => ({
  user: null,
  queue: "all",
  setUser: (user) => set({ user }),
  setQueue: (queue) => set({ queue })
}));
