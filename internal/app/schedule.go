package app

import (
	"time"
	_ "time/tzdata"
)

func scheduleLocation(zone string) *time.Location {
	l, e := time.LoadLocation(zone)
	if zone == "" || e != nil {
		return time.UTC
	}
	return l
}

// Slots follow local wall time; nonexistent DST slots are skipped and repeated slots run once.
func clockSlots(now time.Time, c ServiceConfig, zone string) []time.Time {
	l := scheduleLocation(zone)
	local := now.In(l)
	start, e := time.Parse("15:04", c.SyncStart)
	if e != nil || c.Interval <= 0 {
		return nil
	}
	slots := []time.Time{}
	for day := -2; day <= 2; day++ {
		date := local.AddDate(0, 0, day)
		for minute := 0; minute < 1440; minute++ {
			if (minute-start.Hour()*60-start.Minute()+1440)%c.Interval != 0 {
				continue
			}
			t := time.Date(date.Year(), date.Month(), date.Day(), minute/60, minute%60, 0, 0, l)
			if t.Hour() == minute/60 && t.Minute() == minute%60 {
				slots = append(slots, t)
			}
		}
	}
	return slots
}
func syncDue(now time.Time, last *time.Time, c ServiceConfig, zone string) bool {
	if !c.Enabled || c.Interval == 0 {
		return false
	}
	if last == nil {
		return true
	}
	if c.SyncStart == "" {
		return now.Sub(*last) >= time.Duration(c.Interval)*time.Minute
	}
	for _, t := range clockSlots(now, c, zone) {
		if !t.After(now) && t.After(*last) {
			return true
		}
	}
	return false
}
func nextSync(now time.Time, c ServiceConfig, zone string) *time.Time {
	if !c.Enabled || c.Interval == 0 || c.SyncStart == "" {
		return nil
	}
	for _, t := range clockSlots(now, c, zone) {
		if t.After(now) {
			v := t.UTC()
			return &v
		}
	}
	return nil
}
