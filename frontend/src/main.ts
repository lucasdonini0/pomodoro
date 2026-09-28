import type { API, Settings, State } from "./types";
import { icon, layout } from "./view";
import "./style.css";

const $ = <T extends HTMLElement = HTMLElement>(selector: string) =>
  document.querySelector<T>(selector)!;
const escape = (value: string) =>
  value.replace(
    /[&<>"']/g,
    (char) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        char
      ]!,
  );
let state: State;
let api: API;
let taskKey = "";
let lastBell = 0;
let lastWarning = "";
let audio: AudioContext | undefined;
let busy = false;
let lastStart = 0;
let messageIndex = 0;
let messageTimeout = 0;

const messages = {
  focus: [
    "Uma coisa de cada vez.",
    "Pequenos passos também levam longe.",
    "Você só precisa começar.",
    "Este momento é seu.",
    "Seu esforço está fazendo diferença.",
  ],
  rest: [
    "Respire. O descanso faz parte.",
    "Uma pausa para recarregar.",
    "Solte os ombros. Você merece esse respiro.",
    "Descanse agora, continue com calma.",
  ],
};

function updateMessage() {
  if (state.starts === lastStart) return;
  lastStart = state.starts;
  const pool =
    state.mode === "pomodoro" && state.phase !== "focus"
      ? messages.rest
      : messages.focus;
  const message = pool[messageIndex++ % pool.length];
  const subtitle = $("#subtitle");
  window.clearTimeout(messageTimeout);
  subtitle.classList.add("fading");
  messageTimeout = window.setTimeout(() => {
    subtitle.textContent = message;
    subtitle.classList.remove("fading");
  }, 220);
}

$("#app").innerHTML = layout;

function notify(message: string) {
  $("#toast").textContent = message;
  $("#toast").classList.add("visible");
  window.setTimeout(() => $("#toast").classList.remove("visible"), 4500);
}

function format(seconds: number, stopwatch = false) {
  const total = stopwatch ? Math.floor(seconds) : Math.ceil(seconds);
  const h = Math.floor(total / 3600);
  const m = Math.floor(total / 60) % 60;
  const s = total % 60;
  return `${h ? `${String(h).padStart(2, "0")}:` : ""}${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

function chime() {
  if (!state.settings.sound || state.settings.volume === 0 || !audio) return;
  for (let i = 0; i < 2; i++) {
    const oscillator = audio.createOscillator(),
      gain = audio.createGain();
    const start = audio.currentTime + i * 0.3;
    oscillator.frequency.value = i ? 880 : 660;
    gain.gain.setValueAtTime(0, start);
    gain.gain.linearRampToValueAtTime(
      state.settings.volume * 0.2,
      start + 0.015,
    );
    gain.gain.exponentialRampToValueAtTime(0.001, start + 0.6);
    oscillator.connect(gain).connect(audio.destination);
    oscillator.start(start);
    oscillator.stop(start + 0.6);
  }
}

function render() {
  const c = state.clocks[state.mode];
  const dark =
    state.mode !== "stopwatch" &&
    c.started &&
    (state.mode === "timer" ? state.seconds > 0 : state.phase === "focus");
  document.body.classList.toggle("dark", dark);
  document.body.classList.toggle("compact", state.compact);
  const phase =
    state.mode === "pomodoro"
      ? { focus: "Foco", short: "Pausa curta", long: "Pausa longa" }[
          state.phase
        ] || "Foco"
      : state.mode === "timer"
        ? "Timer"
        : "Stopwatch";
  $("#phase").textContent =
    state.mode === "pomodoro"
      ? `${state.round}/${state.settings.rounds}`
      : phase;
  const time = format(state.seconds, state.mode === "stopwatch");
  $("#time").textContent = time;
  $<HTMLButtonElement>("#time").disabled = state.mode === "stopwatch";
  $("#mini-time").textContent = time;
  $("#time").classList.toggle("hours", time.length > 5);
  $("#mini-phase").textContent =
    state.mode === "pomodoro"
      ? `${state.round}/${state.settings.rounds}`
      : phase;
  $("#mini-toggle").innerHTML = icon(c.running ? "pause" : "play");
  $('[data-action="pin"]').classList.toggle("active", state.pinned);
  $('[data-action="pin"]').setAttribute("aria-pressed", String(state.pinned));
  updateMessage();
  $(".progress span").style.width = `${state.progress * 100}%`;
  const label = c.running
    ? "Pausar"
    : c.started && state.seconds > 0
      ? "Continuar"
      : "INICIAR";
  $("#toggle").innerHTML =
    `${icon(c.running ? "pause" : "play")}<span>${label}</span>`;
  $('[data-action="skip"]').style.visibility =
    state.mode === "pomodoro" ? "visible" : "hidden";
  $("#rounds").innerHTML =
    state.mode === "pomodoro"
      ? Array.from(
          { length: state.settings.rounds },
          (_, i) =>
            `<span class="${i < state.round - 1 || (i === state.round - 1 && state.phase !== "focus") ? "complete" : i === state.round - 1 ? "current" : ""}"></span>`,
        ).join("")
      : "";
  document.querySelectorAll<HTMLElement>("[data-mode]").forEach((el) => {
    el.classList.toggle("active", el.dataset.mode === state.mode);
    el.setAttribute("aria-pressed", String(el.dataset.mode === state.mode));
  });
  $("#tasks-section").hidden = state.mode !== "pomodoro";
  $("#timer-options").hidden = state.mode !== "timer";
  $("#stopwatch-note").hidden = state.mode !== "stopwatch";
  document
    .querySelectorAll<HTMLElement>("[data-seconds]")
    .forEach((el) =>
      el.classList.toggle(
        "active",
        Number(el.dataset.seconds) === state.clocks.timer.duration,
      ),
    );
  const key = JSON.stringify([state.tasks, state.selected]);
  if (key !== taskKey) {
    taskKey = key;
    renderTasks();
  }
  if (state.bell !== lastBell) {
    lastBell = state.bell;
    chime();
    notify(
      state.mode === "pomodoro"
        ? `Hora de ${phase.toLowerCase()}.`
        : "Timer concluído.",
    );
  }
}

function renderTasks() {
  $("#task-count").textContent =
    `${state.tasks.filter((t) => t.done).length}/${state.tasks.length}`;
  $("#task-list").innerHTML = state.tasks.length
    ? state.tasks
        .map(
          (t) =>
            `<div class="task ${t.done ? "done" : ""} ${t.id === state.selected ? "selected" : ""}" data-id="${escape(t.id)}"><button class="checkbox" data-task="done" aria-label="${t.done ? "Reabrir" : "Concluir"} tarefa" aria-pressed="${t.done}">${t.done ? icon("check") : ""}</button><button class="task-title" data-task="select" title="Usar como meta" ${t.done ? "disabled" : ""}>${escape(t.title)}<small>${t.completed}${t.estimate ? ` / ${t.estimate}` : ""} focos${t.id === state.selected ? " · meta atual" : ""}</small></button><div class="task-actions"><button class="icon" data-task="up" aria-label="Mover para cima">${icon("up")}</button><button class="icon" data-task="edit" aria-label="Editar tarefa">${icon("edit")}</button><button class="icon" data-task="delete" aria-label="Excluir tarefa">${icon("trash")}</button></div></div>`,
        )
        .join("")
    : "";
}

async function refresh() {
  state = await api.State();
  render();
  if (state.warning && state.warning !== lastWarning) notify(state.warning);
  lastWarning = state.warning;
}
async function run(operation: () => Promise<void>) {
  if (busy) return;
  busy = true;
  try {
    await operation();
    await refresh();
  } catch (error) {
    notify(String(error));
  } finally {
    busy = false;
  }
}
const command = (action: string, value = "", seconds = 0) =>
  run(() => api.Command(action, value, seconds));

document.addEventListener("click", async (event) => {
  const target = (event.target as HTMLElement).closest<HTMLButtonElement>(
    "button",
  );
  if (!target || !state) return;
  if (!audio) audio = new AudioContext();
  void audio.resume();
  if (target.dataset.mode) {
    await command("mode", target.dataset.mode);
    return;
  }
  if (target.dataset.seconds) {
    await command("timer", "", Number(target.dataset.seconds));
    return;
  }
  const action =
    target.dataset.action === "time-picker" && state.mode === "pomodoro"
      ? "settings"
      : target.dataset.action;
  if (action === "time-picker") {
    if (state.mode === "stopwatch") return;
    const seconds = Math.round(state.clocks[state.mode].duration);
    const form = $<HTMLFormElement>("#time-form");
    form.dataset.mode = state.mode;
    form.dataset.phase = state.phase;
    for (const [name, value] of Object.entries({
      hours: Math.floor(seconds / 3600),
      minutes: Math.floor(seconds / 60) % 60,
      seconds: seconds % 60,
    })) {
      (form.elements.namedItem(name) as HTMLSelectElement).value =
        String(value);
    }
    $<HTMLDialogElement>("#time-dialog").showModal();
  } else if (action === "close-time") {
    $<HTMLDialogElement>("#time-dialog").close();
  } else if (action === "settings") {
    const form = $<HTMLFormElement>("#settings-form");
    for (const [name, value] of Object.entries(state.settings)) {
      const input = form.elements.namedItem(name) as HTMLInputElement;
      if (typeof value === "boolean") input.checked = value;
      else input.value = String(value);
    }
    $<HTMLDialogElement>("#settings").showModal();
  } else if (action === "close-settings") {
    event.preventDefault();
    $<HTMLDialogElement>("#settings").close();
  } else if (action === "close-edit")
    $<HTMLDialogElement>("#edit-dialog").close();
  else if (action === "compact") await run(() => api.Compact());
  else if (action === "pin") await run(() => api.Pin());
  else if (action === "minimize") await api.Minimize();
  else if (action === "quit") await api.Quit();
  else if (action) await command(action);
  const taskAction = target.dataset.task;
  if (!taskAction) return;
  const id = target.closest<HTMLElement>("[data-id]")!.dataset.id!;
  const tasks = structuredClone(state.tasks),
    index = tasks.findIndex((t) => t.id === id),
    task = tasks[index];
  if (taskAction === "select") {
    await command("select", state.selected === id ? "" : id);
    return;
  }
  if (taskAction === "edit") {
    const form = $<HTMLFormElement>("#edit-task");
    form.dataset.id = id;
    (form.elements.namedItem("title") as HTMLInputElement).value = task.title;
    (form.elements.namedItem("estimate") as HTMLInputElement).value = String(
      task.estimate,
    );
    $<HTMLDialogElement>("#edit-dialog").showModal();
    return;
  }
  if (taskAction === "done") task.done = !task.done;
  if (taskAction === "delete") tasks.splice(index, 1);
  if (taskAction === "up" && index > 0)
    [tasks[index - 1], tasks[index]] = [tasks[index], tasks[index - 1]];
  await run(() => api.SaveTasks(tasks));
});

$("#add-task").addEventListener("submit", (event) => {
  event.preventDefault();
  const form = event.target as HTMLFormElement,
    data = new FormData(form);
  const title = String(data.get("title")).trim();
  if (!title) return;
  void run(async () => {
    await api.SaveTasks([
      ...state.tasks,
      {
        id: crypto.randomUUID(),
        title,
        estimate: Number(data.get("estimate")),
        completed: 0,
        done: false,
      },
    ]);
    form.reset();
  });
});
$("#edit-task").addEventListener("submit", (event) => {
  event.preventDefault();
  const form = event.target as HTMLFormElement,
    data = new FormData(form);
  void run(async () => {
    await api.SaveTasks(
      state.tasks.map((t) =>
        t.id === form.dataset.id
          ? {
              ...t,
              title: String(data.get("title")),
              estimate: Number(data.get("estimate")),
            }
          : t,
      ),
    );
    $<HTMLDialogElement>("#edit-dialog").close();
  });
});
$("#settings-form").addEventListener("submit", (event) => {
  event.preventDefault();
  const form = event.target as HTMLFormElement;
  const settings = { ...state.settings };
  for (const key of Object.keys(settings) as (keyof Settings)[]) {
    const input = form.elements.namedItem(key) as HTMLInputElement;
    Object.assign(settings, {
      [key]: input.type === "checkbox" ? input.checked : Number(input.value),
    });
  }
  void run(async () => {
    await api.Configure(settings);
    $<HTMLDialogElement>("#settings").close();
  });
});
$("#time-form").addEventListener("submit", (event) => {
  event.preventDefault();
  const form = event.target as HTMLFormElement;
  const data = new FormData(form);
  const seconds =
    Number(data.get("hours")) * 3600 +
    Number(data.get("minutes")) * 60 +
    Number(data.get("seconds"));
  void run(async () => {
    await api.Command(
      "duration",
      `${form.dataset.mode}:${form.dataset.phase}`,
      seconds,
    );
    $<HTMLDialogElement>("#time-dialog").close();
  });
});
document.addEventListener("keydown", (event) => {
  if (
    event.code !== "Space" ||
    (event.target as HTMLElement).closest("input,button,dialog") ||
    !state
  )
    return;
  event.preventDefault();
  void command("toggle");
});

async function start() {
  if (!window.go?.main.App) {
    $("#subtitle").textContent = "Abra pelo aplicativo Windows para iniciar.";
    return;
  }
  api = window.go.main.App;
  await refresh();
  window.setInterval(() => {
    if (!busy) void refresh().catch(() => {});
  }, 250);
}
void start().catch((error) => notify(String(error)));
