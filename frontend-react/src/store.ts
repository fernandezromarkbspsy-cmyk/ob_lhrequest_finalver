import { create } from "zustand";
import { Queue, SessionUser } from "./api";

type AppState = {
  user: SessionUser | null;
  queue: Queue;
  setUser: (user: SessionUser | null) => void;
  setQueue: (queue: AppState["queue"]) => void;
};

export const useAppStore = create<AppState>((set) => ({
  user: null,
  queue: "all",
  setUser: (user) => set({ user }),
  setQueue: (queue) => set({ queue })
}));
