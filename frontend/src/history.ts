import "./history.css";
import type { Activity, API } from "./types";

export const historyLayout = `
<section id="history" hidden>
 <div class="history-heading"><h2>Histórico</h2><button data-history="today">Hoje</button></div>
 <div class="week-navigation"><button data-history="previous" aria-label="Semana anterior">‹</button><span id="week-label"></span><button data-history="next" aria-label="Próxima semana">›</button></div>
 <div id="week-days" aria-label="Dias da semana"></div>
 <div id="history-summary"></div>
 <div class="section-label history-label">ATIVIDADE POR HORÁRIO</div>
 <div id="hour-chart" role="img" aria-label="Tempo de uso por hora"></div>
 <div class="hour-labels"><span>00h</span><span>06h</span><span>12h</span><span>18h</span><span>24h</span></div>
 <div class="history-legend"><span>Foco</span><span>Timer / Stopwatch</span></div>
 <div id="history-sessions"></div>
</section>`;

const dayNames = ["seg", "ter", "qua", "qui", "sex", "sáb", "dom"];
const dateLabel = (date: Date) =>
  date.toLocaleDateString("pt-BR", { day: "2-digit", month: "short" });
const timeLabel = (date: Date) =>
  date.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" });
const midnight = (date: Date) =>
  new Date(date.getFullYear(), date.getMonth(), date.getDate());
const addDays = (date: Date, count: number) =>
  new Date(date.getFullYear(), date.getMonth(), date.getDate() + count);
const duration = (seconds: number) => {
  if (seconds < 60) return `${Math.floor(seconds)}s`;
  const minutes = Math.floor(seconds / 60);
  return minutes < 60
    ? `${minutes} min`
    : `${Math.floor(minutes / 60)}h ${String(minutes % 60).padStart(2, "0")}min`;
};

export function setupHistory(api: API) {
  const root = document.querySelector<HTMLElement>("#history")!;
  const element = (id: string) => root.querySelector<HTMLElement>(id)!;
  let selected = midnight(new Date());
  let lastToday = selected;
  let records: Activity[] = [];
  let loading = false;
  let lastKey = "";

  function day(date: Date) {
    const begin = +date,
      end = +addDays(date, 1);
    const sessions = records.flatMap((record) => {
      const start = Math.max(+new Date(record.start), begin);
      const stop = Math.min(+new Date(record.end), end);
      if (stop <= start) return [];
      return [
        {
          ...record,
          start,
          stop,
          seconds: (stop - start) / 1000,
          completed:
            record.completed &&
            +new Date(record.end) > begin &&
            +new Date(record.end) <= end,
        },
      ];
    });
    return {
      sessions,
      focus: sessions
        .filter((s) => s.focus)
        .reduce((sum, s) => sum + s.seconds, 0),
      active: sessions.reduce((sum, s) => sum + s.seconds, 0),
      completed: sessions.filter((s) => s.completed).length,
    };
  }

  function render() {
    const today = midnight(new Date());
    if (+selected === +lastToday || +selected > +today) selected = today;
    lastToday = today;
    const monday = addDays(selected, -((selected.getDay() + 6) % 7));
    const days = Array.from({ length: 7 }, (_, i) => addDays(monday, i)).filter(
      (date) => +date <= +today,
    );
    const current = day(selected);
    const key = JSON.stringify([
      +selected,
      +today,
      records.map((r) => [r.start, r.end.slice(0, 19), r.completed]),
    ]);
    if (key === lastKey) return;
    lastKey = key;
    element("#week-label").textContent =
      `${dateLabel(monday)} — ${dateLabel(days[days.length - 1])} · ${days[days.length - 1].getFullYear()}`;
    const next = element('[data-history="next"]') as HTMLButtonElement;
    next.disabled = +addDays(monday, 7) > +today;
    const maxFocus = Math.max(1, ...days.map((d) => day(d).focus));
    element("#week-days").innerHTML = days
      .map(
        (d, i) =>
          `<button data-day="${i}" class="${+d === +selected ? "active" : ""}" aria-pressed="${+d === +selected}" aria-label="${dayNames[i]}, ${dateLabel(d)}"><span>${dayNames[i]}</span><strong>${d.getDate()}</strong><i style="--fill:${(day(d).focus / maxFocus) * 100}%"></i></button>`,
      )
      .join("");
    element("#history-summary").innerHTML =
      `<div><strong>${duration(current.focus)}</strong><span>deep focus</span></div><div><strong>${current.completed}</strong><span>pomodoros concluídos</span></div><small>${duration(current.active)} de uso · ${dateLabel(selected)}</small>`;
    element("#hour-chart").innerHTML = Array.from({ length: 24 }, (_, hour) => {
      const start = +new Date(
        selected.getFullYear(),
        selected.getMonth(),
        selected.getDate(),
        hour,
      );
      const end = +new Date(
        selected.getFullYear(),
        selected.getMonth(),
        selected.getDate(),
        hour + 1,
      );
      let focus = 0,
        other = 0;
      for (const session of current.sessions) {
        const seconds =
          Math.max(
            0,
            Math.min(session.stop, end) - Math.max(session.start, start),
          ) / 1000;
        if (session.focus) focus += seconds;
        else other += seconds;
      }
      return `<div class="hour-column" title="${hour}h: ${duration(focus)} foco, ${duration(other)} outros"><i style="height:${(other / 3600) * 100}%"></i><b style="height:${(focus / 3600) * 100}%"></b></div>`;
    }).join("");
    element("#history-sessions").innerHTML = current.sessions.length
      ? [...current.sessions]
          .reverse()
          .map(
            (s) =>
              `<div class="history-session"><span>${timeLabel(new Date(s.start))}–${timeLabel(new Date(s.stop))}<small>${s.mode === "timer" ? s.focus ? "Timer · Deep Work" : "Timer" : s.focus ? "Foco" : "Stopwatch"}${s.completed ? " · concluído" : ""}</small></span><strong>${duration(s.seconds)}</strong></div>`,
          )
          .join("")
      : '<p class="history-empty">Nenhuma sessão neste dia.</p>';
  }

  root.addEventListener("click", (event) => {
    const button = (event.target as HTMLElement).closest<HTMLButtonElement>(
      "button",
    );
    if (!button) return;
    if (button.dataset.day !== undefined)
      selected = addDays(
        selected,
        -((selected.getDay() + 6) % 7) + Number(button.dataset.day),
      );
    if (button.dataset.history === "previous") selected = addDays(selected, -7);
    if (button.dataset.history === "next") selected = addDays(selected, 7);
    if (button.dataset.history === "today") selected = midnight(new Date());
    if (+selected > +midnight(new Date())) selected = midnight(new Date());
    render();
  });

  async function refresh() {
    if (root.hidden || loading) return;
    render();
    loading = true;
    try {
      records = await api.History();
      render();
    } catch {
      element("#history-sessions").textContent =
        "Não foi possível carregar o histórico.";
    } finally {
      loading = false;
    }
  }
  window.setInterval(() => void refresh(), 1000);
  return refresh;
}
