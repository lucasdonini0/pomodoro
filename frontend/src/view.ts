import { historyLayout } from "./history";

const icons: Record<string, string> = {
  play: '<path d="m8 5 11 7-11 7z"/>',
  pause: '<path d="M8 5v14M16 5v14"/>',
  reset: '<path d="M3 10a9 9 0 1 1 2 8M3 4v6h6"/>',
  skip: '<path d="m5 5 10 7-10 7zM19 5v14"/>',
  settings: '<path d="M4 7h16M4 17h16M8 4v6M16 14v6"/>',
  compact:
    '<rect x="4" y="4" width="16" height="16" rx="1"/><path d="M12 14h6v4h-6z"/>',
  expand: '<path d="M8 3H3v5M16 3h5v5M3 16v5h5M21 16v5h-5"/>',
  close: '<path d="m6 6 12 12M18 6 6 18"/>',
  minus: '<path d="M5 12h14"/>',
  pin: '<path d="m8 3 8 0-1 7 4 4H5l4-4zM12 14v7"/>',
  plus: '<path d="M12 5v14M5 12h14"/>',
  trash: '<path d="M4 6h16M9 6V3h6v3M7 6l1 15h8l1-15M10 10v7M14 10v7"/>',
  up: '<path d="m6 14 6-6 6 6"/>',
  edit: '<path d="m4 16 12-12 4 4-12 12H4z"/>',
  check: '<path d="m5 12 4 4L19 6"/>',
  bell: '<path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4"/>',
};
export const icon = (name: string) =>
  `<svg viewBox="0 0 24 24" aria-hidden="true">${icons[name]}</svg>`;
const button = (action: string, title: string, name: string) =>
  `<button class="icon" data-action="${action}" title="${title}" aria-label="${title}">${icon(name)}</button>`;
export const layout = `
 <div class="night"></div><div class="particles" aria-hidden="true">${Array.from({ length: 28 }, (_, i) => `<i style="--x:${(i * 37 + 11) % 100}%;--y:${(i * 23 + 7) % 100}%;--delay:-${i * 1.7}s;--duration:${7 + (i % 7)}s;--drift:${(i % 2 ? -1 : 1) * (12 + (i % 17))}px"></i>`).join("")}</div>
 <header><span class="brand"><span class="brand-mark"></span>pomodoro</span><div class="window-actions">${button("settings", "Configurações", "settings")}${button("compact", "Modo compacto", "compact")}${button("minimize", "Minimizar", "minus")}${button("quit", "Fechar", "close")}</div></header>
 <main><nav aria-label="Modo"><button data-mode="pomodoro">Pomodoro</button><button data-mode="timer">Timer</button><button data-mode="stopwatch">Stopwatch</button><button data-action="alarms">Despertador</button><button data-action="history">Histórico</button></nav>
 ${historyLayout}
 <section id="alarms" hidden><form id="alarm-form"><div class="alarm-display"><div class="eyebrow">ESCOLHA UM HORÁRIO</div><div class="alarm-time-input"><select name="hours" aria-label="Hora">${Array.from({ length: 24 }, (_, i) => `<option value="${String(i).padStart(2, "0")}">${String(i).padStart(2, "0")}</option>`).join("")}</select><span>:</span><select name="minutes" aria-label="Minutos">${Array.from({ length: 60 }, (_, i) => `<option value="${String(i).padStart(2, "0")}">${String(i).padStart(2, "0")}</option>`).join("")}</select></div><div class="alarm-subtitle">Um novo começo, na hora certa.</div><div class="progress"><span></span></div><div class="controls"><button class="primary" type="submit">${icon("plus")}<span>ADICIONAR</span></button></div></div><div class="alarm-options"><div class="section-label">COMO VAI TOCAR?</div><div class="alarm-controls"><label>Som<select name="sound"><option value="alerta">Alerta</option><option value="sirene">Sirene</option><option value="campainha">Campainha</option><option value="random">Random</option></select></label><button type="button" class="secondary" data-action="preview-alarm">Ouvir</button></div><label class="volume alarm-volume">Volume<input name="volume" type="range" min="0" max="1" step="0.05" value="0.85"></label><label class="switch-row">Repetir até desativar<input name="repeat" type="checkbox" checked></label><label class="switch-row">Remover depois de tocar<input name="removeAfter" type="checkbox"></label></div></form><div class="alarm-list-heading section-heading"><span class="section-label">HORÁRIOS SALVOS</span><span id="alarm-count"></span></div><div id="alarm-list"></div><p class="hint alarm-hint">O app precisa permanecer aberto para o despertador tocar.</p></section>
 <section class="clock"><div class="eyebrow" id="phase">SEU TEMPO, COM INTENÇÃO</div><button id="time" data-action="time-picker" aria-label="Configurar tempo">25:00</button><div id="subtitle">Um passo de cada vez.</div><div class="progress"><span></span></div><div class="controls">${button("reset", "Reiniciar", "reset")}<button class="primary" data-action="toggle" id="toggle">${icon("play")}<span>INICIAR</span></button>${button("skip", "Pular etapa", "skip")}</div><div id="rounds"></div></section>
 <section id="timer-options"><div class="section-label">ESCOLHA UMA DURAÇÃO</div><div class="presets">${[5, 10, 15, 30, 60, 180, 360, 720].map((m) => `<button data-seconds="${m * 60}">${m < 60 ? `${m} min` : `${m / 60} h`}</button>`).join("")}</div></section>
 <section id="tasks-section"><div class="section-heading"><span class="section-label">O QUE VAMOS FAZER?</span><span id="task-count"></span></div><div id="task-list"></div><form id="add-task"><input name="title" aria-label="Nova tarefa" placeholder="Adicionar uma tarefa…" maxlength="180" required autocomplete="off"><input name="estimate" type="number" aria-label="Meta de pomodoros" title="Meta de pomodoros" min="0" max="99" value="1"><button class="icon" aria-label="Adicionar tarefa">${icon("plus")}</button></form></section>
 <section id="stopwatch-note"><span class="small-mark">↗</span><h2>No seu ritmo.</h2><p>Cada segundo também conta.</p></section>
 </main>
 <div id="mini"><div class="mini-top"><span id="mini-phase"></span><div>${button("pin", "Sempre por cima", "pin")}${button("compact", "Restaurar janela", "expand")}</div></div><div class="mini-bottom"><strong id="mini-time"></strong><button class="icon" data-action="toggle" id="mini-toggle" aria-label="Iniciar ou pausar"></button></div></div>
 <dialog id="settings"><form id="settings-form"><div class="section-heading"><h2>Configurações</h2>${button("close-settings", "Fechar configurações", "close")}</div><div class="settings-grid">${[
   ["focus", "Foco", 240],
   ["short", "Pausa curta", 120],
   ["long", "Pausa longa", 240],
   ["rounds", "Focos por ciclo", 20],
 ]
   .map(
     ([name, title, max]) =>
       `<label>${title}<div><input name="${name}" type="number" min="1" max="${max}" required><span>${name === "rounds" ? "focos" : "min"}</span></div></label>`,
   )
   .join("")}</div>${[
   ["autoBreak", "Iniciar pausas automaticamente"],
   ["autoFocus", "Iniciar focos automaticamente"],
   ["sound", "Som ao concluir"],
 ]
   .map(
     ([name, title]) =>
       `<label class="switch-row">${title}<input type="checkbox" name="${name}"></label>`,
   )
   .join(
     "",
   )}<label class="volume">Volume<input name="volume" type="range" min="0" max="1" step="0.05"></label><button class="primary save" type="submit">Salvar ajustes</button></form></dialog>
 <dialog id="time-dialog"><form id="time-form"><h2>Configurar tempo</h2><div class="time-picker">${[
   ["hours", "Horas", 23],
   ["minutes", "Minutos", 59],
   ["seconds", "Segundos", 59],
 ]
   .map(
     ([name, label, max]) =>
       `<label>${label}<select name="${name}">${Array.from({ length: Number(max) + 1 }, (_, i) => `<option value="${i}">${String(i).padStart(2, "0")}</option>`).join("")}</select></label>`,
   )
   .join(
     "",
   )}</div><div class="edit-actions"><button type="button" class="secondary" data-action="close-time">Cancelar</button><button class="primary" type="submit">Definir</button></div></form></dialog>
 <dialog id="edit-dialog"><form id="edit-task"><h2>Editar tarefa</h2><input name="title" aria-label="Nome da tarefa" maxlength="180" required><label>Meta de pomodoros<input name="estimate" type="number" min="0" max="99" required></label><div class="edit-actions"><button type="button" class="secondary" data-action="close-edit">Cancelar</button><button class="primary" type="submit">Salvar</button></div></form></dialog><dialog id="alarm-ringing"><div class="ringing-content"><span class="small-mark">${icon("bell")}</span><div class="section-label">DESPERTADOR</div><strong id="ringing-time"></strong><p>Está na hora.</p><button type="button" class="primary" data-action="dismiss-alarm">Desativar</button></div></dialog><div id="toast" role="status"></div>`;
