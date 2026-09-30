package main

import (
	"errors"
	"slices"
	"time"
)

type Alarm struct {
	ID          string  `json:"id"`
	Time        string  `json:"time"`
	Sound       string  `json:"sound"`
	Volume      float64 `json:"volume"`
	Repeat      bool    `json:"repeat"`
	RemoveAfter bool    `json:"removeAfter"`
	LastFired   string  `json:"lastFired"`
}

func (alarm Alarm) valid() bool {
	if len(alarm.ID) == 0 || len(alarm.ID) > 100 || len(alarm.Time) != 5 || alarm.Volume < 0 || alarm.Volume > 1 {
		return false
	}
	if _, err := time.Parse("15:04", alarm.Time); err != nil {
		return false
	}
	return slices.Contains([]string{"suave", "sinos", "aurora", "digital", "random"}, alarm.Sound)
}

// checkAlarms runs on the Go ticker, including while the window is minimised.
func (a *App) checkAlarms(now time.Time) bool {
	day := now.Format("2006-01-02")
	clock := now.Format("15:04")
	changed := false
	remove := map[string]bool{}
	for i := range a.alarms {
		alarm := &a.alarms[i]
		if alarm.Time != clock || alarm.LastFired == day {
			continue
		}
		alarm.LastFired = day
		if a.activeAlarm == nil {
			copy := *alarm
			a.activeAlarm = &copy
		} else {
			a.pendingAlarms = append(a.pendingAlarms, *alarm)
		}
		if alarm.RemoveAfter {
			remove[alarm.ID] = true
		}
		changed = true
	}
	if len(remove) > 0 {
		a.alarms = slices.DeleteFunc(a.alarms, func(alarm Alarm) bool { return remove[alarm.ID] })
	}
	return changed
}

func (a *App) AddAlarm(alarm Alarm) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !alarm.valid() || len(a.alarms) >= 50 {
		return errors.New("Confira o horário e as opções do despertador")
	}
	for _, existing := range a.alarms {
		if existing.ID == alarm.ID {
			return errors.New("Despertador duplicado")
		}
	}
	alarm.LastFired = ""
	a.alarms = append(a.alarms, alarm)
	a.save()
	return nil
}

func (a *App) RemoveAlarm(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.alarms = slices.DeleteFunc(a.alarms, func(alarm Alarm) bool { return alarm.ID == id })
	a.pendingAlarms = slices.DeleteFunc(a.pendingAlarms, func(alarm Alarm) bool { return alarm.ID == id })
	if a.activeAlarm != nil && a.activeAlarm.ID == id {
		a.nextAlarm()
	}
	a.save()
}

func (a *App) DismissAlarm() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeAlarm == nil {
		return
	}
	a.nextAlarm()
	a.save()
}

func (a *App) nextAlarm() {
	a.activeAlarm = nil
	if len(a.pendingAlarms) > 0 {
		copy := a.pendingAlarms[0]
		a.activeAlarm = &copy
		a.pendingAlarms = a.pendingAlarms[1:]
	}
}
