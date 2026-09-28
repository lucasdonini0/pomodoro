export type Mode = "pomodoro" | "timer" | "stopwatch";
export type Settings = {
  focus: number;
  short: number;
  long: number;
  rounds: number;
  autoBreak: boolean;
  autoFocus: boolean;
  sound: boolean;
  volume: number;
  reducedMotion: boolean;
};
export type Task = {
  id: string;
  title: string;
  estimate: number;
  completed: number;
  done: boolean;
};
export type State = {
  settings: Settings;
  tasks: Task[];
  selected: string;
  mode: Mode;
  phase: string;
  round: number;
  seconds: number;
  progress: number;
  bell: number;
  compact: boolean;
  pinned: boolean;
  warning: string;
  clocks: Record<
    Mode,
    { running: boolean; started: boolean; duration: number }
  >;
};
export type API = {
  State(): Promise<State>;
  Command(action: string, value: string, seconds: number): Promise<void>;
  Configure(settings: Settings): Promise<void>;
  SaveTasks(tasks: Task[]): Promise<void>;
  Compact(): Promise<void>;
  Pin(): Promise<void>;
  Minimize(): Promise<void>;
  Quit(): Promise<void>;
};
declare global {
  interface Window {
    go?: { main: { App: API } };
  }
}
