package rtl_433_exporter

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// InfluxSample is a single sample from influx-formatted data
type InfluxSample struct {
	Metric    string
	Metadata  map[string]string
	Timestamp time.Time
	Value     float64
}

// parse weird rtl_433 influx data, it is supposed to look like
// cpu_load_short,host=server01,region=us-west value=0.64 1434055562000000000
// but instead is actually (sometimes multiple) lines of
// Acurite-Tower,id=15680,channel=A battery_ok=1,temperature_F=64.760002,humidity=43,mic="CHECKSUM"
// parse that into a sample
func parseBadRTLInfluxData(data string) ([]*InfluxSample, error) {
	// grab a timestamp
	ts := time.Now()

	// we might have multiple lines, some the same, some not
	samples := []*InfluxSample{}
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		data = line

		// split on the first comma
		namesplit := strings.SplitAfterN(data, ",", 2)
		if len(namesplit) != 2 {
			return nil, fmt.Errorf("unexpectedly couldn't split %s into name, rest", data)
		}
		metricname, data := namesplit[0], namesplit[1]
		// log.Printf("metric name: %s", metricname)

		// split on spaces - metadata valuePairs
		spacesplit := strings.Split(data, " ")
		if len(spacesplit) != 2 {
			return nil, fmt.Errorf("unexpectedly couldn't split %s into metadata, valuePairs", data)
		}
		metadataPairs, valuePairs := spacesplit[0], spacesplit[1]

		// it's all metadata at this point anyway
		pairs := fmt.Sprintf("%s,%s", metadataPairs, valuePairs)
		// metadata
		metadata := map[string]string{}
		for _, pair := range strings.Split(pairs, ",") {
			// log.Printf("metadata pair %s", pair)
			if pair != "" {
				kv := strings.Split(pair, "=")
				if len(kv) != 2 {
					return nil, fmt.Errorf("couldn't split %s into key, value", pair)
				}
				key, value := kv[0], kv[1]
				metadata[key] = value
			}
		}

		// done
		// log.Printf("final metricname: %s", metricname)
		// log.Printf("final metadata: %s", metadata)
		// log.Printf("final value: %f", 0.0)
		// log.Printf("final timestamp: %s", ts)

		samples = append(samples, &InfluxSample{
			Metric:    metricname,
			Metadata:  metadata,
			Timestamp: ts,
			Value:     0.0,
		})
	}

	return samples, nil
}

// parse Influx formatted data that looks like
// this is dead code since rtl_433 does some wild stuff, but maybe it'll be useful to me some other time
// cpu_load_short,host=server01,region=us-west value=0.64 1434055562000000000
// ->
// metricname,metadata=pairs,metadata=pairs value=value timestamp
// into a InfluxSample
func parseInfluxData(data string) (*InfluxSample, error) {
	// split on the first comma
	namesplit := strings.SplitAfterN(data, ",", 2)
	metricname, data := namesplit[0], namesplit[1]
	// log.Printf("metric name: %s", metricname)

	// split on spaces - metadata valuePair timestamp
	spacesplit := strings.Split(data, " ")
	if len(spacesplit) != 3 {
		return nil, fmt.Errorf("unexpectedly couldn't split %s into metadata, value, timestamp", data)

	}
	metadataPairs, valuePair, timestamp := spacesplit[0], spacesplit[1], spacesplit[2]
	// log.Printf("metadata: %s", metadataPairs)
	// log.Printf("value: %s", valuePair)
	// log.Printf("timestamp: %s", timestamp)

	// metadata
	metadata := map[string]string{}
	for _, pair := range strings.Split(metadataPairs, ",") {
		// log.Printf("metadata pair %s", pair)
		if pair != "" {
			kv := strings.Split(pair, "=")
			if len(kv) != 2 {
				return nil, fmt.Errorf("couldn't split %s into key, value", pair)
			}
			key, value := kv[0], kv[1]
			metadata[key] = value
		}
	}

	// value
	kv := strings.Split(valuePair, "=")
	if len(kv) != 2 {
		return nil, fmt.Errorf("couldn't split %s into value, value", valuePair)

	}
	if kv[0] != "value" {
		return nil, fmt.Errorf("unexpected result in value split: %s", kv)

	}
	value, err := strconv.ParseFloat(kv[1], 64)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse value as float: %s", kv[1])

	}

	// timestamp
	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse timestamp: %s", timestamp)

	}
	ts := time.Unix(0, i)

	// done
	// log.Printf("final metricname: %s", metricname)
	// log.Printf("final metadata: %s", metadata)
	// log.Printf("final value: %f", value)
	// log.Printf("final timestamp: %s", ts)

	return &InfluxSample{
		Metric:    metricname,
		Metadata:  metadata,
		Timestamp: ts,
		Value:     value,
	}, nil
}
