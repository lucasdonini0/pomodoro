package main

import (
	"testing"
	"time"
)

func TestTimerUntilStartsAndRingsAtChosenTime(t *testing.T) {
	zone := time.FixedZone("America/Sao_Paulo", -3*60*60)
	for _, test := range []struct {
		name   string
		now    time.Time
		chosen string
		end    time.Time
	}{
		{"today", time.Date(2026, 10, 8, 14, 20, 30, 0, zone), "16:05", time.Date(2026, 10, 8, 16, 5, 0, 0, zone)},
		{"tomorrow", time.Date(2026, 10, 8, 14, 20, 30, 0, zone), "13:15", time.Date(2026, 10, 9, 13, 15, 0, 0, zone)},
		{"midnight", time.Date(2026, 10, 8, 23, 50, 45, 0, zone), "00:15", time.Date(2026, 10, 9, 0, 15, 0, 0, zone)},
		{"same time", time.Date(2026, 10, 8, 14, 20, 0, 0, zone), "14:20", time.Date(2026, 10, 9, 14, 20, 0, 0, zone)},
		{"less than a minute", time.Date(2026, 10, 8, 14, 20, 45, 750000000, zone), "14:21", time.Date(2026, 10, 8, 14, 21, 0, 0, zone)},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := newEngine()
			e.Mode = "timer"
			if err := e.command("timer-until", test.chosen, 0, test.now); err != nil {
				t.Fatal(err)
			}
			clock := e.Clocks["timer"]
			if !clock.Running || !clock.Started || clock.Duration != test.end.Sub(test.now).Seconds() || e.Starts != 1 {
				t.Fatalf("timer was not started with the remaining duration: %+v", clock)
			}
			if end := clock.endsAt(test.now); end == nil || !end.Equal(test.end) {
				t.Fatalf("wrong finish time: %v", end)
			}
			e.advance(test.end.Add(-time.Second))
			if e.Bell != 0 || !clock.Running {
				t.Fatal("timer rang early")
			}
			e.advance(test.end)
			e.advance(test.end.Add(time.Minute))
			if e.Bell != 1 || clock.Running || clock.endsAt(test.end) != nil {
				t.Fatal("timer did not complete exactly once")
			}
		})
	}
}

func TestTimerUntilRejectsInvalidOrStaleSelection(t *testing.T) {
	now := time.Now()
	e := newEngine()
	if e.command("timer-until", "16:00", 0, now) == nil {
		t.Fatal("accepted target time outside Timer mode")
	}
	e.Mode = "timer"
	for _, value := range []string{"", "24:00", "12:60", "9:15", "bad"} {
		if e.command("timer-until", value, 0, now) == nil {
			t.Fatalf("accepted invalid clock time %q", value)
		}
	}
	if e.Clocks["timer"].Running || e.Starts != 0 || e.Clocks["timer"].Duration != 300 {
		t.Fatal("invalid selection changed the timer")
	}
}

func TestTimerUntilPreservesPreviousDeepWork(t *testing.T) {
	e := newEngine()
	e.Mode, e.TimerDeepWork = "timer", true
	now := time.Date(2026, 10, 8, 14, 20, 0, 0, time.Local)
	if err := e.command("toggle", "", 0, now); err != nil {
		t.Fatal(err)
	}
	if err := e.command("timer-until", "14:30", 0, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	e.advance(now.Add(5 * time.Minute))
	if e.Bell != 0 || !e.Clocks["timer"].Running {
		t.Fatal("previous deadline interrupted the new timer")
	}
	e.advance(now.Add(10 * time.Minute))
	if len(e.History) != 2 || e.History[0].End.Sub(e.History[0].Start) != 2*time.Minute || e.History[1].End.Sub(e.History[1].Start) != 8*time.Minute {
		t.Fatal("changing the finish time lost or duplicated history")
	}
	for _, entry := range e.History {
		if !entry.Focus || entry.Completed {
			t.Fatal("changing the finish time changed the session type")
		}
	}
}

func TestTimerFinishPredictionFollowsPauseAndResume(t *testing.T) {
	now := time.Date(2026, 10, 8, 14, 0, 0, 0, time.Local)
	clock := &Clock{Duration: 300}
	if end := clock.endsAt(now); end == nil || !end.Equal(now.Add(5*time.Minute)) {
		t.Fatal("incorrect prediction before starting")
	}
	clock.Running, clock.Started, clock.anchor = true, true, now
	if end := clock.endsAt(now.Add(time.Minute)); end == nil || !end.Equal(now.Add(5*time.Minute)) {
		t.Fatal("finish time moved while running")
	}
	clock.pause(now.Add(time.Minute))
	if end := clock.endsAt(now.Add(time.Hour)); end == nil || !end.Equal(now.Add(time.Hour+4*time.Minute)) {
		t.Fatal("prediction ignored the remaining time after pausing")
	}
}
