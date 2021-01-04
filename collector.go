package rtl_433_exporter

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &Collector{}

// collector is a prometheus.Collector for rtl_433 data
type Collector struct {
	Temperature *prometheus.Desc
	Humidity    *prometheus.Desc
	Battery     *prometheus.Desc

	RoomMeasurements      *prometheus.Desc
	Measurements          *prometheus.Desc
	DiscardedMeasurements *prometheus.Desc

	areaMap               map[string]string
	latestMeasurements    map[string]sample
	totalMeasurements     map[string]float64
	discardedMeasurements float64

	mu sync.Mutex
}

// NewCollector returns a a collector
func NewCollector(areaMap map[string]string) *Collector {
	return &Collector{
		Temperature: prometheus.NewDesc(
			"rtl_temperature",
			"Temperature in F of the area (or id).",
			[]string{"area"},
			nil,
		),
		Humidity: prometheus.NewDesc(
			"rtl_humidity",
			"Humidity as a percentage for the area (or id).",
			[]string{"area"},
			nil,
		),
		Battery: prometheus.NewDesc(
			"rtl_battery_status",
			"Battery status as reported be the sensor",
			[]string{"area"},
			nil,
		),
		Measurements: prometheus.NewDesc(
			"rtl_measurements_total",
			"Number of measurements reported for the area",
			[]string{"area"},
			nil,
		),
		DiscardedMeasurements: prometheus.NewDesc(
			"rtl_discarded_measurements_total",
			"Number of measurements reported that were invalid for some reason",
			nil,
			nil,
		),

		areaMap:               areaMap,
		latestMeasurements:    map[string]sample{},
		totalMeasurements:     map[string]float64{},
		discardedMeasurements: 0.0,
	}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.Temperature,
		c.Humidity,
		c.Battery,
		c.Measurements,
		c.DiscardedMeasurements,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// meta counts
	for area, measurements := range c.totalMeasurements {
		ch <- prometheus.MustNewConstMetric(
			c.Measurements,
			prometheus.CounterValue,
			measurements,
			area,
		)
	}

	ch <- prometheus.MustNewConstMetric(
		c.DiscardedMeasurements,
		prometheus.CounterValue,
		c.discardedMeasurements,
	)

	// the good stuff
	for area, sample := range c.latestMeasurements {
		// if this is a recent measurement, show it
		if time.Now().Sub(sample.Time).Seconds() < 300 {
			ch <- prometheus.MustNewConstMetric(
				c.Temperature,
				prometheus.GaugeValue,
				sample.Temperature,
				area,
			)

			ch <- prometheus.MustNewConstMetric(
				c.Humidity,
				prometheus.GaugeValue,
				sample.Humidity,
				area,
			)

			ch <- prometheus.MustNewConstMetric(
				c.Battery,
				prometheus.GaugeValue,
				sample.Battery,
				area,
			)
		}
	}
}

// Observe some samples and reconcile them with the current state of the world
func (c *Collector) Observe(samples []*InfluxSample) {
	for _, sample := range samples {
		s := c.influxSampleToSample(sample)
		if s == nil {
			continue
		}

		currentAreaSample, exists := c.latestMeasurements[s.Area]
		if !exists {
			// if we haven't seen this room, lock it in
			c.mu.Lock()
			c.latestMeasurements[s.Area] = *s
			c.mu.Unlock()
			continue
		}

		// if this is a newer sample for the room, lock it in
		if s.Time.After(currentAreaSample.Time) {
			c.mu.Lock()
			c.latestMeasurements[s.Area] = *s
			c.mu.Unlock()
		}
	}
}

type sample struct {
	Time        time.Time
	Area        string
	Temperature float64
	Humidity    float64
	Battery     float64
}

var requiredMetadataFields = []string{
	"id",
	"temperature_F",
	"humidity",
	"battery_ok",
}

func (c *Collector) influxSampleToSample(s *InfluxSample) *sample {
	for _, rf := range requiredMetadataFields {
		if _, exists := s.Metadata[rf]; !exists {
			c.discardedMeasurements += 1
			log.Printf("discarding sample, not all required fields found: %+v", *s)
			return nil
		}
	}

	id := s.Metadata["id"]
	if x, known := c.areaMap[id]; known {
		id = x
	}
	temp, err1 := strconv.ParseFloat(s.Metadata["temperature_F"], 64)
	humidity, err2 := strconv.ParseFloat(s.Metadata["humidity"], 64)
	battery, err3 := strconv.ParseFloat(s.Metadata["battery_ok"], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		log.Printf("discarding sample, unable to parse temperature, humidity, or battery status: %+v", *s)
		c.discardedMeasurements += 1
		return nil
	}

	if n, exists := c.totalMeasurements[id]; !exists {
		c.totalMeasurements[id] = 1
	} else {
		c.totalMeasurements[id] = n + 1
	}

	return &sample{
		Time:        s.Timestamp,
		Area:        id,
		Temperature: temp,
		Humidity:    humidity,
		Battery:     battery,
	}
}
