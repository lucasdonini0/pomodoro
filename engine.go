package main

import (
	"errors"
	"math"
	"time"
)

type Settings struct {
	Focus     int     `json:"focus"`
	Short     int     `json:"short"`
	Long      int     `json:"long"`
	Rounds    int     `json:"rounds"`
	AutoBreak bool    `json:"autoBreak"`
	AutoFocus bool    `json:"autoFocus"`
	Sound     bool    `json:"sound"`
	Volume    float64 `json:"volume"`
}

type Task struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Estimate  int    `json:"estimate"`
	Completed int    `json:"completed"`
	Done      bool   `json:"done"`
}

type Clock struct {
	Duration float64 `json:"duration"`
	Elapsed  float64 `json:"elapsed"`
	Running  bool    `json:"running"`
	Started  bool    `json:"started"`
	anchor   time.Time
}

type Engine struct {
	Settings          Settings          `json:"settings"`
	Tasks             []Task            `json:"tasks"`
	Selected          string            `json:"selected"`
	Mode              string            `json:"mode"`
	TimerDeepWork     bool              `json:"timerDeepWork"`
	StopwatchDeepWork bool              `json:"stopwatchDeepWork"`
	Phase             string            `json:"phase"`
	Round             int               `json:"round"`
	Clocks            map[string]*Clock `json:"clocks"`
	Bell              int               `json:"bell"`
	Starts            int               `json:"starts"`
	History           []Activity        `json:"-"`
	active            int
}

func newEngine() *Engine {
	return &Engine{
		History: []Activity{}, active: -1,
		Settings: Settings{Focus: 25, Short: 5, Long: 15, Rounds: 4, Sound: true, Volume: 0.4},
		Tasks:    []Task{}, Mode: "pomodoro", Phase: "focus", Round: 1,
		Clocks: map[string]*Clock{"pomodoro": {Duration: 1500}, "timer": {Duration: 300}, "stopwatch": {}},
	}
}

func (c *Clock) elapsed(now time.Time) float64 {
	if c.Running {
		return c.Elapsed + math.Max(0, float64(now.UnixMilli()-c.anchor.UnixMilli())/1000)
	}
	return c.Elapsed
}

func (c *Clock) pause(now time.Time) {
	c.Elapsed = c.elapsed(now)
	c.Running = false
}

func (e *Engine) advance(now time.Time) {
	e.record(now)
	c := e.Clocks[e.Mode]
	if e.Mode == "stopwatch" || !c.Running || c.elapsed(now) < c.Duration {
		return
	}
	e.finish(now, true)
	c.Elapsed, c.Running = c.Duration, false
	e.Bell++
	if e.Mode == "pomodoro" {
		e.next(now, true)
	}
}

func (e *Engine) next(now time.Time, completed bool) {
	if e.Phase == "focus" {
		if completed {
			for i := range e.Tasks {
				if e.Tasks[i].ID == e.Selected && !e.Tasks[i].Done {
					e.Tasks[i].Completed++
				}
			}
		}
		e.Phase = "short"
		if e.Round >= e.Settings.Rounds {
			e.Phase = "long"
		}
	} else {
		if e.Phase == "long" {
			e.Round = 1
		} else {
			e.Round++
		}
		e.Phase = "focus"
	}
	duration := e.Settings.Focus
	if e.Phase == "short" {
		duration = e.Settings.Short
	}
	if e.Phase == "long" {
		duration = e.Settings.Long
	}
	running := completed && ((e.Phase == "focus" && e.Settings.AutoFocus) || (e.Phase != "focus" && e.Settings.AutoBreak))
	if running {
		e.Starts++
	}
	e.Clocks["pomodoro"] = &Clock{Duration: float64(duration * 60), Running: running, Started: running, anchor: now}
}

func (e *Engine) command(action, value string, seconds int, now time.Time) error {
	e.advance(now)
	c := e.Clocks[e.Mode]
	switch action {
	case "mode":
		if _, ok := e.Clocks[value]; !ok {
			return errors.New("Modo inválido")
		}
		if value == e.Mode {
			return nil
		}
		c.pause(now)
		e.finish(now, false)
		e.Mode = value
	case "toggle":
		if c.Running {
			e.finish(now, false)
			c.pause(now)
		} else {
			if !c.Started || (e.Mode != "stopwatch" && c.Elapsed >= c.Duration) {
				e.Starts++
			}
			if e.Mode != "stopwatch" && c.Elapsed >= c.Duration {
				c.Elapsed = 0
			}
			c.Running, c.Started, c.anchor = true, true, now
		}
	case "reset":
		e.finish(now, false)
		c.Elapsed, c.Running, c.Started = 0, false, false
	case "skip":
		if e.Mode == "pomodoro" {
			e.finish(now, false)
			e.next(now, false)
		}
	case "timer":
		if seconds < 1 || seconds > 86400 {
			return errors.New("Use uma duração entre 1 segundo e 24 horas")
		}
		if e.Mode == "timer" {
			e.finish(now, false)
		}
		e.Clocks["timer"] = &Clock{Duration: float64(seconds)}
	case "timer-work", "stopwatch-work":
		mode, selected := "timer", &e.TimerDeepWork
		if action == "stopwatch-work" {
			mode, selected = "stopwatch", &e.StopwatchDeepWork
		}
		if e.Mode != mode || (value != "normal" && value != "deep") {
			return errors.New("Tipo de sessão inválido")
		}
		deep := value == "deep"
		if deep == *selected {
			return nil
		}
		if c.Running {
			e.finish(now, false)
			c.Elapsed, c.anchor = c.elapsed(now), now
		}
		*selected = deep
	case "duration":
		if e.Mode == "stopwatch" || value != e.Mode+":"+e.Phase {
			return errors.New("A etapa mudou. Abra o seletor novamente")
		}
		if seconds < 1 || seconds > 86399 {
			return errors.New("Escolha um tempo entre 1 segundo e 23:59:59")
		}
		e.finish(now, false)
		e.Clocks[e.Mode] = &Clock{Duration: float64(seconds)}
	case "select":
		if e.Phase == "focus" && e.Clocks["pomodoro"].Started {
			return errors.New("Reinicie o foco antes de trocar a meta")
		}
		if value != "" {
			found := false
			for _, task := range e.Tasks {
				if task.ID == value && !task.Done {
					found = true
				}
			}
			if !found {
				return errors.New("Selecione uma tarefa pendente")
			}
		}
		e.Selected = value
	default:
		return errors.New("Comando inválido")
	}
	e.record(now)
	return nil
}

func (s Settings) valid() bool {
	return s.Focus >= 1 && s.Focus <= 240 && s.Short >= 1 && s.Short <= 120 && s.Long >= 1 && s.Long <= 240 && s.Rounds >= 1 && s.Rounds <= 20 && s.Volume >= 0 && s.Volume <= 1
}
