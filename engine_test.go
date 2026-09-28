package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestPauseResume(t *testing.T) {
	e := newEngine()
	now := time.Unix(1000, 0)
	_ = e.command("toggle", "", 0, now)
	_ = e.command("toggle", "", 0, now.Add(10*time.Second))
	if got := e.Clocks["pomodoro"].elapsed(now.Add(time.Hour)); got != 10 {
		t.Fatalf("paused: %v", got)
	}
	_ = e.command("toggle", "", 0, now.Add(time.Hour))
	if got := e.Clocks["pomodoro"].elapsed(now.Add(time.Hour + 5*time.Second)); got != 15 {
		t.Fatalf("resumed: %v", got)
	}
}

func TestCycleAndTaskCredit(t *testing.T) {
	e := newEngine()
	e.Tasks = []Task{{ID: "a", Title: "Read", Estimate: 4}}
	e.Selected = "a"
	now := time.Unix(1000, 0)
	for round := 1; round <= 4; round++ {
		_ = e.command("toggle", "", 0, now)
		now = now.Add(25 * time.Minute)
		e.advance(now)
		want := "short"
		if round == 4 {
			want = "long"
		}
		if e.Phase != want || e.Round != round || e.Clocks["pomodoro"].Running {
			t.Fatalf("round %d: %+v", round, e)
		}
		if e.Tasks[0].Completed != round {
			t.Fatalf("credit: %d", e.Tasks[0].Completed)
		}
		_ = e.command("skip", "", 0, now)
	}
	if e.Round != 1 || e.Phase != "focus" {
		t.Fatal("cycle did not restart")
	}
}

func TestSkippedFocusHasNoCredit(t *testing.T) {
	e := newEngine()
	e.Tasks = []Task{{ID: "a", Title: "Read"}}
	e.Selected = "a"
	_ = e.command("skip", "", 0, time.Now())
	if e.Tasks[0].Completed != 0 {
		t.Fatal("skipped focus credited")
	}
}

func TestSuspendAndAutoStart(t *testing.T) {
	e := newEngine()
	e.Settings.AutoBreak = true
	now := time.Unix(1000, 0)
	_ = e.command("toggle", "", 0, now)
	e.advance(now.Add(12 * time.Hour))
	if e.Phase != "short" || !e.Clocks["pomodoro"].Running || e.Bell != 1 {
		t.Fatal("resume transition failed")
	}
	if e.Clocks["pomodoro"].elapsed(now.Add(12*time.Hour)) != 0 {
		t.Fatal("break should start on resume")
	}
}

func TestModeSwitchPauses(t *testing.T) {
	e := newEngine()
	now := time.Now()
	_ = e.command("toggle", "", 0, now)
	_ = e.command("mode", "stopwatch", 0, now.Add(time.Second))
	if e.Clocks["pomodoro"].Running || e.Clocks["pomodoro"].Elapsed != 1 {
		t.Fatal("previous mode not paused")
	}
	_ = e.command("toggle", "", 0, now)
	e.advance(now.Add(48 * time.Hour))
	if e.Clocks["stopwatch"].elapsed(now.Add(48*time.Hour)) != 172800 {
		t.Fatal("stopwatch capped")
	}
}

func TestTimerCompletesOnce(t *testing.T) {
	e := newEngine()
	now := time.Now()
	_ = e.command("mode", "timer", 0, now)
	if e.command("timer", "", 0, now) == nil {
		t.Fatal("zero duration accepted")
	}
	_ = e.command("timer", "", 43200, now)
	_ = e.command("toggle", "", 0, now)
	e.advance(now.Add(12 * time.Hour))
	e.advance(now.Add(13 * time.Hour))
	if e.Bell != 1 || e.Clocks["timer"].Running {
		t.Fatal("completion repeated")
	}
	_ = e.command("toggle", "", 0, now.Add(14*time.Hour))
	if e.Clocks["timer"].Elapsed != 0 || !e.Clocks["timer"].Running {
		t.Fatal("timer did not restart")
	}
}

func TestPersistence(t *testing.T) {
	a := newApp()
	a.path = t.TempDir() + "/data.json"
	a.engine.Tasks = []Task{{ID: "a", Title: "Read", Completed: 2}}
	a.save()
	a.engine.Tasks[0].Completed = 3
	a.save()
	if a.warning != "" {
		t.Fatal(a.warning)
	}
	data, err := os.ReadFile(a.path)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct{ Tasks []Task }
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Tasks) != 1 || saved.Tasks[0].Completed != 3 {
		t.Fatal("changes not persisted")
	}
}

func TestSettingsApplyNextPhase(t *testing.T) {
	a := newApp()
	now := time.Now()
	_ = a.engine.command("toggle", "", 0, now)
	settings := a.engine.Settings
	settings.Focus, settings.Short = 40, 10
	if err := a.Configure(settings); err != nil {
		t.Fatal(err)
	}
	if a.engine.Clocks["pomodoro"].Duration != 1500 {
		t.Fatal("active focus changed")
	}
	a.engine.advance(now.Add(25 * time.Minute))
	if a.engine.Clocks["pomodoro"].Duration != 600 {
		t.Fatal("new break duration missing")
	}
}

func TestTaskCreditCannotBeOverwritten(t *testing.T) {
	a := newApp()
	a.engine.Tasks = []Task{{ID: "a", Title: "Read", Completed: 3}}
	if err := a.SaveTasks([]Task{{ID: "a", Title: "Write", Completed: 0}}); err != nil {
		t.Fatal(err)
	}
	if a.engine.Tasks[0].Completed != 3 {
		t.Fatal("stale UI overwrote credit")
	}
}
