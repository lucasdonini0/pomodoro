package main

import "time"

func (a *App) suspend() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.engine.advance(time.Now())
	a.suspended = a.engine.Clocks[a.engine.Mode].Running
	if a.suspended {
		_ = a.engine.command("toggle", "", 0, time.Now())
	}
	a.save()
}

func (a *App) resume() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.suspended && !a.engine.Clocks[a.engine.Mode].Running {
		_ = a.engine.command("toggle", "", 0, time.Now())
	}
	a.suspended = false
	a.save()
}

type Activity struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Mode      string    `json:"mode"`
	Focus     bool      `json:"focus"`
	Completed bool      `json:"completed"`
}

func (e *Engine) record(now time.Time) {
	c := e.Clocks[e.Mode]
	if !c.Running || (e.Mode == "pomodoro" && e.Phase != "focus") {
		return
	}
	end := now
	if e.Mode != "stopwatch" {
		deadline := c.anchor.Add(time.Duration((c.Duration - c.Elapsed) * float64(time.Second)))
		if end.After(deadline) {
			end = deadline
		}
	}
	if e.active < 0 {
		focus := (e.Mode == "pomodoro" && e.Phase == "focus") ||
			(e.Mode == "timer" && e.TimerDeepWork) ||
			(e.Mode == "stopwatch" && e.StopwatchDeepWork)
		e.History = append(e.History, Activity{Start: c.anchor, End: c.anchor, Mode: e.Mode, Focus: focus})
		e.active = len(e.History) - 1
	}
	entry := &e.History[e.active]
	if end.After(entry.End) {
		entry.End = end
	}
}

func (e *Engine) finish(now time.Time, completed bool) {
	e.record(now)
	if e.active >= 0 {
		entry := &e.History[e.active]
		entry.Completed = completed && entry.Mode == "pomodoro" && entry.Focus
	}
	e.active = -1
}
