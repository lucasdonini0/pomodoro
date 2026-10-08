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
	if c.warning != "Histórico recuperado da cópia de segurança." || len(c.engine.History) != 1 {
		t.Fatal("old data failed migration or history recovery")
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

func TestTimerDeepWorkOnlyCountsItsOwnMinutes(t *testing.T) {
	e := newEngine()
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.Local)
	if err := e.command("mode", "timer", 0, now); err != nil {
		t.Fatal(err)
	}
	if err := e.command("toggle", "", 0, now); err != nil {
		t.Fatal(err)
	}
	if err := e.command("timer-work", "deep", 0, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := e.command("timer-work", "normal", 0, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	e.advance(now.Add(5 * time.Minute))
	if len(e.History) != 3 {
		t.Fatalf("want three segments, got %d", len(e.History))
	}
	wantDuration := []time.Duration{2 * time.Minute, time.Minute, 2 * time.Minute}
	wantFocus := []bool{false, true, false}
	for i, entry := range e.History {
		if entry.End.Sub(entry.Start) != wantDuration[i] || entry.Focus != wantFocus[i] || entry.Completed {
			t.Fatalf("segment %d: %+v", i, entry)
		}
	}
	if e.Bell != 1 || e.Clocks["timer"].Running {
		t.Fatal("switching session type changed timer completion")
	}
}

func TestStopwatchDeepWorkOnlyCountsItsOwnMinutes(t *testing.T) {
	e := newEngine()
	now := time.Date(2026, 10, 8, 9, 0, 0, 0, time.Local)
	for _, step := range []struct {
		action, value string
		minutes       int
	}{
		{"mode", "stopwatch", 0},
		{"toggle", "", 0},
		{"stopwatch-work", "deep", 2},
		{"stopwatch-work", "normal", 3},
	} {
		if err := e.command(step.action, step.value, 0, now.Add(time.Duration(step.minutes)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	e.advance(now.Add(5 * time.Minute))
	if len(e.History) != 3 {
		t.Fatalf("want three segments, got %d", len(e.History))
	}
	wantDuration := []time.Duration{2 * time.Minute, time.Minute, 2 * time.Minute}
	wantFocus := []bool{false, true, false}
	for i, entry := range e.History {
		if entry.Mode != "stopwatch" || entry.End.Sub(entry.Start) != wantDuration[i] || entry.Focus != wantFocus[i] || entry.Completed {
			t.Fatalf("segment %d: %+v", i, entry)
		}
	}
	if e.Bell != 0 || e.Starts != 1 || !e.Clocks["stopwatch"].Running || e.Clocks["stopwatch"].elapsed(now.Add(5*time.Minute)) != 300 {
		t.Fatal("switching session type interrupted or reset the stopwatch")
	}
}

func TestSessionDeepWorkSelectionsPersistIndependently(t *testing.T) {
	for _, mode := range []string{"timer", "stopwatch"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("APPDATA", dir)
			t.Setenv("XDG_CONFIG_HOME", dir)
			a := newApp()
			a.startup(context.Background())
			if err := a.Command("mode", mode, 0); err != nil {
				t.Fatal(err)
			}
			if err := a.Command(mode+"-work", "deep", 0); err != nil {
				t.Fatal(err)
			}
			a.shutdown(context.Background())
			b := newApp()
			b.startup(context.Background())
			defer b.shutdown(context.Background())
			if b.engine.TimerDeepWork != (mode == "timer") || b.engine.StopwatchDeepWork != (mode == "stopwatch") {
				t.Fatal("session types were not restored independently")
			}
		})
	}
}

func TestHistoryBackupRestoresMissingEntries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	a := newApp()
	a.startup(context.Background())
	start := time.Date(2026, 10, 5, 9, 0, 0, 0, time.Local)
	a.engine.History = append(a.engine.History, Activity{
		Start: start, End: start.Add(25 * time.Minute), Mode: "pomodoro", Focus: true, Completed: true,
	})
	a.save()
	a.save()
	a.engine.History = append(a.engine.History, Activity{
		Start: start.Add(time.Hour), End: start.Add(85 * time.Minute), Mode: "timer", Focus: true,
	})
	a.save()
	a.save()
	a.shutdown(context.Background())
	data, err := os.ReadFile(a.path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]json.RawMessage
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved["history"] = json.RawMessage("[]")
	data, err = json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.path, data, 0600); err != nil {
		t.Fatal(err)
	}
	b := newApp()
	b.startup(context.Background())
	defer b.shutdown(context.Background())
	if len(b.engine.History) != 2 || !b.engine.History[0].Completed {
		t.Fatal("missing history was not restored from backup")
	}
}

func TestHistoryOnDiskCannotBeOverwrittenByEmptyState(t *testing.T) {
	path := t.TempDir() + "/data.json"
	a := newApp()
	a.path = path
	a.engine.History = []Activity{{Start: time.Now().Add(-time.Minute), End: time.Now(), Mode: "timer", Focus: true}}
	a.save()
	b := newApp()
	b.path = path
	b.save()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct{ History []Activity }
	if err := json.Unmarshal(data, &saved); err != nil || len(saved.History) != 1 || !b.dataReadOnly {
		t.Fatal("existing history was overwritten")
	}
}

func TestInvalidSavedDataIsPreserved(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := dir + "/pomodoro/data.json"
	if err := os.MkdirAll(dir+"/pomodoro", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newApp()
	a.startup(context.Background())
	a.shutdown(context.Background())
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "invalid json" {
		t.Fatal("invalid saved data was overwritten")
	}
}
