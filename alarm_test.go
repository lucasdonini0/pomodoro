package main

import (
	"testing"
	"time"
)

func TestAlarmFiresOncePerDayAndRemovesAfterDismissal(t *testing.T) {
	app := newApp()
	alarm := Alarm{ID: "one", Time: "07:30", Sound: "suave", Volume: 0.6, Repeat: true, RemoveAfter: true}
	if err := app.AddAlarm(alarm); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, time.September, 30, 7, 30, 0, 0, time.Local)
	if !app.checkAlarms(day) || app.activeAlarm == nil {
		t.Fatal("alarm should start at its scheduled time")
	}
	if len(app.alarms) != 0 {
		t.Fatal("one-time alarm should be removed after ringing")
	}
	if app.checkAlarms(day.Add(30 * time.Second)) {
		t.Fatal("alarm fired twice in the same day")
	}
	app.DismissAlarm()
	if app.activeAlarm != nil {
		t.Fatal("alarm should stop when dismissed")
	}
}

func TestDailyAlarmsQueueAtSameTime(t *testing.T) {
	app := newApp()
	for _, id := range []string{"first", "second"} {
		if err := app.AddAlarm(Alarm{ID: id, Time: "08:00", Sound: "random", Volume: 0.5}); err != nil {
			t.Fatal(err)
		}
	}
	day := time.Date(2026, time.September, 30, 8, 0, 0, 0, time.Local)
	app.checkAlarms(day)
	if app.activeAlarm == nil || app.activeAlarm.ID != "first" || len(app.pendingAlarms) != 1 {
		t.Fatal("simultaneous alarms should queue")
	}
	app.DismissAlarm()
	if app.activeAlarm == nil || app.activeAlarm.ID != "second" {
		t.Fatal("next alarm should ring after dismissal")
	}
	app.DismissAlarm()
	if len(app.alarms) != 2 || app.checkAlarms(day.Add(time.Minute)) {
		t.Fatal("daily alarms should remain saved")
	}
	if !app.checkAlarms(day.Add(24 * time.Hour)) {
		t.Fatal("daily alarms should ring again tomorrow")
	}
}
