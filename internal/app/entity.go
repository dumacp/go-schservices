package app

import (
	"math"
	"sort"
	"time"
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

func (s Service) Valid() bool {
	if s.Itinerary <= 0 {
		return false
	}
	if time.Until(time.UnixMilli(s.Timestamp)) >= 0 {
		return false
	}
	if s.Timestamp > s.TimeService {
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

func Next(ss []Service, from, to time.Time) (Service, bool) {
	var result Service
	mem := float64(math.MaxFloat64)
	for _, v := range ss {
		center := math.Pow(float64(v.Timestamp-time.Now().UnixMilli()), 2)
		if center < mem {
			mem = center
			result = v
		}
	}
	t := time.UnixMilli(result.Timestamp)
	// fmt.Printf("timestamp: %s\n", t)
	// fmt.Printf("before: %v, %s\n", t.After(from), from)
	// fmt.Printf("after: %v, %s\n", t.Before(to), to)
	if t.After(from) && t.Before(to) {

		return result, true
	}
	return result, false
}

func Newest(ss []Service) (Service, bool) {
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Timestamp < ss[j].Timestamp
	})

	var last Service
	for _, v := range ss {
		if v.Timestamp > time.Now().Add(3*time.Minute).UnixMilli() && (last != Service{}) {
			return last, true
		}
		last = v
	}
	if len(ss) > 0 {
		last := ss[len(ss)-1]
		if last.Timestamp <= time.Now().Add(3*time.Minute).UnixMilli() && (last != Service{}) {
			return last, true
		}
	}
	return Service{}, false
}
