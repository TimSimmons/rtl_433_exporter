package rtl_433_exporter

import (
	"io/ioutil"
	"log"
	"net/http"
)

var _ http.Handler = &InfluxHandler{}

// InfluxHandler is an HTTP handler for the influx-formatted writes from rtl_433 that forwards them on to our Prometheus collector
type InfluxHandler struct {
	c *Collector
}

// NewInfluxHandler returns a new InfluxHandler
func NewInfluxHandler(c *Collector) InfluxHandler {
	return InfluxHandler{c: c}
}

// InfluxHandler takes the influx-formatted writes from rtl_433, sends them to the parser,
// and feeds them to the Collector.
func (h InfluxHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		log.Println("no request body")
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("could not read request body")
		return
	}

	body := string(b)
	//	log.Printf("request body: %s", body)

	samples, err := parseBadRTLInfluxData(body)
	if err != nil {
		log.Printf("error parsing influx data: %s", err)
	}

	h.c.Observe(samples)

	w.WriteHeader(http.StatusOK)
}
