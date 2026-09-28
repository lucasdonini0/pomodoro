package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestHistoryReloadAndMigration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	a := newApp()
	a.startup(context.Background())
	now := time.Now()
	_ = a.engine.command("toggle", "", 0, now.Add(-time.Minute))
	_ = a.engine.command("toggle", "", 0, now)
	a.shutdown(context.Background())
	b := newApp()
	b.startup(context.Background())
	if len(b.engine.History) != 1 || b.engine.History[0].End.Sub(b.engine.History[0].Start) != time.Minute {
		t.Fatal("history was not restored")
	}
	if b.engine.Clocks["pomodoro"].Running {
		t.Fatal("closed session resumed")
	}
	b.shutdown(context.Background())
	data, _ := json.Marshal(struct {
		Settings Settings `json:"settings"`
	}{newEngine().Settings})
	if err := os.WriteFile(b.path, data, 0600); err != nil {
		t.Fatal(err)
	}
	c := newApp()
	c.startup(context.Background())
	defer c.shutdown(context.Background())
	if c.warning != "" || len(c.engine.History) != 0 {
		t.Fatal("old data failed migration")
	}
}

func TestSuspendResumesWithoutNewSessionStart(t *testing.T) {
	a := newApp()
	_ = a.engine.command("toggle", "", 0, time.Now())
	a.suspend()
	if a.engine.Clocks["pomodoro"].Running || !a.suspended {
		t.Fatal("suspend did not pause")
	}
	a.resume()
	if !a.engine.Clocks["pomodoro"].Running || a.suspended || a.engine.Starts != 1 {
		t.Fatal("resume changed focus start")
	}
}

func TestHistoryPauseResume(t *testing.T) {
	e := newEngine()
	now := time.Date(2026, 9, 28, 10, 0, 0, 0, time.Local)
	_ = e.command("toggle", "", 0, now)
	_ = e.command("toggle", "", 0, now.Add(10*time.Minute))
	_ = e.command("toggle", "", 0, now.Add(20*time.Minute))
	e.advance(now.Add(35 * time.Minute))
	if len(e.History) != 2 {
		t.Fatalf("segments: %d", len(e.History))
	}
	var total time.Duration
	for _, entry := range e.History {
		total += entry.End.Sub(entry.Start)
	}
	if total != 25*time.Minute {
		t.Fatalf("focus: %v", total)
	}
	if e.History[0].Completed || !e.History[1].Completed {
		t.Fatal("completion count incorrect")
	}
	_ = e.command("toggle", "", 0, now.Add(35*time.Minute))
	e.advance(now.Add(40 * time.Minute))
	if len(e.History) != 2 {
		t.Fatal("break was recorded")
	}
}

func TestHistoryDeadlineAndMidnight(t *testing.T) {
	e := newEngine()
	now := time.Date(2026, 9, 28, 23, 50, 0, 0, time.FixedZone("local", -3*3600))
	_ = e.command("toggle", "", 0, now)
	e.advance(now.Add(12 * time.Hour))
	if len(e.History) != 1 {
		t.Fatal("missing session")
	}
	entry := e.History[0]
	if entry.End.Sub(entry.Start) != 25*time.Minute || entry.End.Day() != 29 || entry.End.Hour() != 0 || entry.End.Minute() != 15 {
		t.Fatalf("wrong deadline: %+v", entry)
	}
	if !entry.Completed {
		t.Fatal("focus not completed")
	}
}

func TestHistoryResetSkipAndSwitch(t *testing.T) {
	for _, action := range []string{"reset", "skip", "mode"} {
		t.Run(action, func(t *testing.T) {
			e := newEngine()
			now := time.Now()
			_ = e.command("toggle", "", 0, now)
			_ = e.command(action, "timer", 0, now.Add(time.Minute))
			if len(e.History) != 1 || e.History[0].Completed || e.History[0].End.Sub(e.History[0].Start) != time.Minute {
				t.Fatal("partial focus not preserved")
			}
			_ = e.command("mode", "stopwatch", 0, now.Add(2*time.Minute))
			_ = e.command("toggle", "", 0, now.Add(2*time.Minute))
			e.advance(now.Add(3 * time.Minute))
			if len(e.History) != 2 || e.History[1].Focus {
				t.Fatal("stopwatch counted as focus")
			}
		})
	}
}

func TestHistoryPersistence(t *testing.T) {
	a := newApp()
	a.path = t.TempDir() + "/data.json"
	now := time.Now()
	_ = a.engine.command("toggle", "", 0, now)
	a.engine.advance(now.Add(time.Minute))
	a.save()
	data, err := os.ReadFile(a.path)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		History  []Activity
		Settings Settings
	}
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.History) != 1 || saved.History[0].End.Sub(saved.History[0].Start) != time.Minute || !saved.Settings.valid() {
		t.Fatal("history not persisted")
	}
}

func TestHistoryDurationChange(t *testing.T) {
	e := newEngine()
	now := time.Now()
	_ = e.command("mode", "timer", 0, now)
	_ = e.command("toggle", "", 0, now)
	_ = e.command("timer", "", 600, now.Add(time.Minute))
	_ = e.command("toggle", "", 0, now.Add(2*time.Minute))
	e.advance(now.Add(3 * time.Minute))
	if len(e.History) != 2 || e.History[0].End.Sub(e.History[0].Start) != time.Minute || e.History[1].End.Sub(e.History[1].Start) != time.Minute {
		t.Fatal("duration change merged sessions")
	}
}
