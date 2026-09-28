package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	mu            sync.Mutex
	engine        *Engine
	path          string
	warning       string
	compact       bool
	pinned        bool
	width, height int
	done          chan struct{}
	lastSave      time.Time
	suspended     bool
}

type Snapshot struct {
	*Engine
	Seconds  float64 `json:"seconds"`
	Progress float64 `json:"progress"`
	Compact  bool    `json:"compact"`
	Pinned   bool    `json:"pinned"`
	Warning  string  `json:"warning"`
}

func newApp() *App {
	return &App{engine: newEngine(), width: 440, height: 640, done: make(chan struct{})}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	defer func() { go a.tick() }()
	dir, err := os.UserConfigDir()
	if err != nil {
		a.warning = err.Error()
		return
	}
	a.path = filepath.Join(dir, "pomodoro", "data.json")
	data, err := os.ReadFile(a.path)
	if err != nil {
		if !os.IsNotExist(err) {
			a.warning = "Não foi possível carregar os dados salvos."
		}
		return
	}
	var saved struct {
		Settings Settings
		Tasks    []Task
		Selected string
		History  []Activity
	}
	if json.Unmarshal(data, &saved) != nil || !saved.Settings.valid() {
		a.warning = "Os dados salvos estão inválidos."
		return
	}
	a.engine.Settings, a.engine.Selected = saved.Settings, saved.Selected
	if saved.History != nil {
		a.engine.History = saved.History
	}
	if saved.Tasks != nil {
		a.engine.Tasks = saved.Tasks
	}
	a.engine.Clocks["pomodoro"].Duration = float64(saved.Settings.Focus * 60)
}

func (a *App) save() {
	if a.path == "" {
		return
	}
	data, err := json.MarshalIndent(struct {
		Settings Settings   `json:"settings"`
		Tasks    []Task     `json:"tasks"`
		Selected string     `json:"selected"`
		History  []Activity `json:"history"`
	}{a.engine.Settings, a.engine.Tasks, a.engine.Selected, a.engine.History}, "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(a.path), 0700)
	}
	if err == nil {
		err = os.WriteFile(a.path+".tmp", data, 0600)
	}
	if err == nil {
		err = os.Rename(a.path+".tmp", a.path)
	}
	if err != nil {
		a.warning = "Não foi possível salvar as alterações."
	} else {
		a.warning = ""
		a.lastSave = time.Now()
	}
}

func (a *App) tick() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			a.mu.Lock()
			bell := a.engine.Bell
			a.engine.advance(now)
			if bell != a.engine.Bell || (a.engine.Clocks[a.engine.Mode].Running && now.Sub(a.lastSave) >= 10*time.Second) {
				a.save()
			}
			a.mu.Unlock()
		case <-a.done:
			return
		}
	}
}

func (a *App) shutdown(context.Context) {
	close(a.done)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.engine.advance(time.Now())
	a.engine.finish(time.Now(), false)
	a.save()
}

func (a *App) History() []Activity {
	a.mu.Lock()
	defer a.mu.Unlock()
	bell := a.engine.Bell
	a.engine.advance(time.Now())
	if bell != a.engine.Bell {
		a.save()
	}
	return slices.Clone(a.engine.History)
}

func (a *App) State() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now()
	bell := a.engine.Bell
	a.engine.advance(now)
	if bell != a.engine.Bell {
		a.save()
	}
	c := a.engine.Clocks[a.engine.Mode]
	elapsed := c.elapsed(now)
	seconds, progress := elapsed, 0.0
	if a.engine.Mode != "stopwatch" {
		seconds = max(0, c.Duration-elapsed)
		progress = min(1, elapsed/c.Duration)
	}
	copy := *a.engine
	copy.Tasks = slices.Clone(a.engine.Tasks)
	copy.Clocks = make(map[string]*Clock, len(a.engine.Clocks))
	for mode, clock := range a.engine.Clocks {
		value := *clock
		copy.Clocks[mode] = &value
	}
	return Snapshot{&copy, seconds, progress, a.compact, a.pinned, a.warning}
}

func (a *App) Command(action, value string, seconds int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.engine.command(action, value, seconds, time.Now()); err != nil {
		return err
	}
	a.save()
	return nil
}

func (a *App) Configure(settings Settings) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !settings.valid() {
		return errors.New("Confira os limites das configurações")
	}
	a.engine.Settings = settings
	c := a.engine.Clocks["pomodoro"]
	if !c.Started {
		minutes := settings.Focus
		if a.engine.Phase == "short" {
			minutes = settings.Short
		}
		if a.engine.Phase == "long" {
			minutes = settings.Long
		}
		c.Duration = float64(minutes * 60)
	}
	a.save()
	return nil
}

func (a *App) SaveTasks(tasks []Task) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(tasks) > 200 {
		return errors.New("Limite de 200 tarefas")
	}
	seen := map[string]bool{}
	for i := range tasks {
		tasks[i].Title = strings.TrimSpace(tasks[i].Title)
		if tasks[i].ID == "" || seen[tasks[i].ID] || len(tasks[i].Title) == 0 || len(tasks[i].Title) > 500 || tasks[i].Estimate < 0 || tasks[i].Estimate > 99 {
			return errors.New("Tarefa inválida")
		}
		seen[tasks[i].ID] = true
		tasks[i].Completed = 0
		for _, old := range a.engine.Tasks {
			if old.ID == tasks[i].ID {
				tasks[i].Completed = old.Completed
			}
		}
	}
	a.engine.Tasks = tasks
	if !seen[a.engine.Selected] {
		a.engine.Selected = ""
	}
	a.save()
	return nil
}

func (a *App) Compact() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.compact = !a.compact
	if a.compact {
		a.width, a.height = runtime.WindowGetSize(a.ctx)
		runtime.WindowSetMinSize(a.ctx, 320, 110)
		runtime.WindowSetSize(a.ctx, 320, 110)
		a.pinned = true
	} else {
		runtime.WindowSetMinSize(a.ctx, 380, 560)
		runtime.WindowSetSize(a.ctx, a.width, a.height)
		a.pinned = false
	}
	runtime.WindowSetAlwaysOnTop(a.ctx, a.pinned)
}

func (a *App) Pin() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pinned = !a.pinned
	runtime.WindowSetAlwaysOnTop(a.ctx, a.pinned)
}
func (a *App) Minimize() { runtime.WindowMinimise(a.ctx) }
func (a *App) Quit()     { runtime.Quit(a.ctx) }
