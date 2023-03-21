package app

import (
	"math"
	"sort"
	"time"

	"github.com/dumacp/go-schservices/pkg/messages"
)

type Services struct {
	ID                                string    `json:"id"`
	DeviceID                          string    `json:"deviceId"`
	ScheduledServices                 []Service `json:"lss"`
	LiveExecutedServices              []Service `json:"les"`
	LastTimestampScheduledServices    int64     `json:"lsst"`
	LastTimestampLiveExecutedServices int64     `json:"lest"`
}
type Service struct {
	Timestamp   int64 `json:"t"`
	TimeService int64 `json:"et"`
	Ruta        int   `json:"r"`
	Itinerary   int   `json:"i"`
}

func (s Service) Newer(o *Service) bool {
	return s.Timestamp > o.Timestamp
}

func Valid(s *messages.ScheduleService) bool {
	if len(s.GetId()) <= 0 {
		return false
	}
	if s.GetScheduleDateTime() > 0 {
		return false
	}
	return true
}

func (s Service) Same(o Service) bool {
	if s.Itinerary == o.Itinerary && s.Ruta == o.Ruta && s.TimeService == o.TimeService &&
		s.Timestamp == o.Timestamp {
		return true
	}
	return false
}

func Next(ss []*messages.ScheduleService, from, to time.Time) (*messages.ScheduleService, bool) {
	var result *messages.ScheduleService
	mem := float64(math.MaxFloat64)
	for i := range ss {
		center := math.Pow(float64(ss[i].GetScheduleDateTime()-time.Now().UnixMilli()), 2)
		if center < mem {
			mem = center
			result = ss[i]
		}
	}
	t := time.UnixMilli(result.GetScheduleDateTime())
	// fmt.Printf("timestamp: %s\n", t)
	// fmt.Printf("before: %v, %s\n", t.After(from), from)
	// fmt.Printf("after: %v, %s\n", t.Before(to), to)
	if t.After(from) && t.Before(to) {

		return result, true
	}
	return result, false
}

func Newest(ss []messages.ScheduleService) (*messages.ScheduleService, bool) {
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].GetScheduleDateTime() < ss[j].GetScheduleDateTime()
	})

	var last *messages.ScheduleService
	for i := range ss {
		v := ss[i]
		if v.GetScheduleDateTime() > time.Now().Add(3*time.Minute).UnixMilli() && (last != nil) {
			return last, true
		}
		last = &v
	}
	if len(ss) > 0 {
		last := &ss[len(ss)-1]
		if last.GetScheduleDateTime() <= time.Now().Add(3*time.Minute).UnixMilli() && (last != nil) {
			return last, true
		}
	}
	return nil, false
}
