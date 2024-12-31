package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"

	"github.com/simonswine/fronius-exporter/api"
)

const timeout = 15 * time.Second

var log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().
	Timestamp().
	Logger()

type collector struct {
	api *api.Fronius

	inverterInfo        *prometheus.Desc
	inverterStatus      *prometheus.Desc
	inverterTotalEnergy *prometheus.Desc
	inverterDCVoltage   *prometheus.Desc
	inverterDCCurrent   *prometheus.Desc
	inverterACFrequency *prometheus.Desc
	inverterACVoltage   *prometheus.Desc
	inverterACCurrent   *prometheus.Desc
}

func newCollector(api *api.Fronius) *collector {
	return &collector{
		api: api,
		inverterInfo: prometheus.NewDesc(
			"fronius_inverter_info",
			"Information about the inverter",
			[]string{"device_id", "device_type", "device_name", "serial"},
			nil,
		),
		inverterStatus: prometheus.NewDesc(
			"fronius_inverter_status",
			"Status of the inverter",
			[]string{"device_id", "status"},
			nil,
		),
		inverterTotalEnergy: prometheus.NewDesc(
			"inverter_yield_total",
			"Information about the inverter",
			[]string{"device_id"},
			nil,
		),
		inverterDCVoltage: prometheus.NewDesc(
			"inverter_dc_voltage",
			"Solar panel (DC) voltage",
			[]string{"device_id"},
			nil,
		),
		inverterDCCurrent: prometheus.NewDesc(
			"inverter_dc_current",
			"Solar panel (DC) current",
			[]string{"device_id"},
			nil,
		),
		inverterACFrequency: prometheus.NewDesc(
			"inverter_grid_frequency",
			"Grid (AC) frequency",
			[]string{"device_id"},
			nil,
		),
		inverterACVoltage: prometheus.NewDesc(
			"inverter_grid_voltage",
			"Grid (AC) current",
			[]string{"device_id", "phase"},
			nil,
		),
		inverterACCurrent: prometheus.NewDesc(
			"inverter_grid_current",
			"Grid (AC) current",
			[]string{"device_id", "phase"},
			nil,
		),
	}
}

// Describe implements Collector.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.inverterInfo
	ch <- c.inverterStatus
	ch <- c.inverterTotalEnergy
	ch <- c.inverterDCVoltage
	ch <- c.inverterDCCurrent
	ch <- c.inverterACFrequency
	ch <- c.inverterACVoltage
	ch <- c.inverterACCurrent
}

// Collect implements Collector.
func (c *collector) collectInverters(ch chan<- prometheus.Metric) (deviceIDs []string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	inverters, err := c.api.GetInverterInfo(ctx)
	if err != nil {
		log.Err(err).Msg("unable to get inverter info")
		return
	}
	ids := make([]string, 0, len(inverters))
	for _, inverter := range inverters {
		ids = append(ids, inverter.Name)

		// report metadata of inverter
		m, err := prometheus.NewConstMetric(
			c.inverterInfo,
			prometheus.GaugeValue,
			1.0,
			inverter.Name,
			fmt.Sprintf("%d", inverter.Dt),
			inverter.CustomName,
			inverter.UniqueID,
		)
		if err != nil {
			log.Err(err).Msg("unable to generate metrics for inverter info")
			continue
		}
		ch <- m

		// report status of inverter
		for _, status := range api.StatusCodes() {
			value := 0.0
			if inverter.StatusCode.String() == status {
				value = 1.0
			}
			m, err := prometheus.NewConstMetric(
				c.inverterStatus,
				prometheus.GaugeValue,
				value,
				inverter.Name,
				strings.ToLower(status),
			)
			if err != nil {
				log.Err(err).Msg("unable to generate metrics for inverter status")
				continue
			}
			ch <- m
		}

	}

	return ids
}

func (c *collector) collectCommonInverterData(ch chan<- prometheus.Metric, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	data, err := c.api.GetInverterRealtimeCommonData(ctx, id)
	if err != nil {
		log.Err(err).Msg("unable to get common data for inverter")
		return
	}
	{
		m, err := prometheus.NewConstMetric(
			c.inverterTotalEnergy,
			prometheus.CounterValue,
			data.TotalEnergy.Value/1000.0,
			id,
		)
		if err != nil {
			log.Err(err).Msg("unable to generate metrics for inverter total energy")
			return
		}
		ch <- m
	}
	{
		m, err := prometheus.NewConstMetric(
			c.inverterDCVoltage,
			prometheus.GaugeValue,
			data.Udc.Value,
			id,
		)
		if err != nil {
			log.Err(err).Msg("unable to generate metrics for inverter dc voltage")
			return
		}
		ch <- m
	}
	{
		m, err := prometheus.NewConstMetric(
			c.inverterDCCurrent,
			prometheus.GaugeValue,
			data.Idc.Value,
			id,
		)
		if err != nil {
			log.Err(err).Msg("unable to generate metrics for inverter dc current")
			return
		}
		ch <- m
	}
	{
		m, err := prometheus.NewConstMetric(
			c.inverterACFrequency,
			prometheus.GaugeValue,
			data.Fac.Value,
			id,
		)
		if err != nil {
			log.Err(err).Msg("unable to generate metrics for inverter grid frequency")
			return
		}
		ch <- m
	}
}

func (c *collector) collectThreePhaseInverterData(ch chan<- prometheus.Metric, id string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	data, err := c.api.GetInverterRealtimeThreePhaseData(ctx, id)
	if err != nil {
		log.Err(err).Msg("unable to get three phase data for inverter")
		return
	}

	for _, x := range []struct {
		desc  *prometheus.Desc
		data  api.DataValue
		value string
	}{
		{c.inverterACVoltage, data.UacL1, "L1"},
		{c.inverterACVoltage, data.UacL2, "L2"},
		{c.inverterACVoltage, data.UacL3, "L3"},
		{c.inverterACCurrent, data.IacL1, "L1"},
		{c.inverterACCurrent, data.IacL2, "L2"},
		{c.inverterACCurrent, data.IacL3, "L3"},
	} {

		m, err := prometheus.NewConstMetric(
			x.desc,
			prometheus.GaugeValue,
			x.data.Value,
			id,
			x.value,
		)
		if err != nil {
			log.Err(err).Msg("unable to generate metrics for grid voltage/phaseinverter total energy")
			return
		}
		ch <- m
	}

}

func (c *collector) Collect(ch chan<- prometheus.Metric) {

	ids := c.collectInverters(ch)

	for _, id := range ids {
		c.collectCommonInverterData(ch, id)
		c.collectThreePhaseInverterData(ch, id)
	}
}

func run() error {

	var (
		addr string
		url  string
	)
	flag.StringVar(&addr, "listen-address", ":9109", "The address to listen on for HTTP requests.")
	flag.StringVar(&url, "fronius-url", "", "URL for the fronius inverter.")
	flag.Parse()

	if url == "" {
		return fmt.Errorf("no fronius-url set")
	}

	// create fronius collector
	a, err := api.NewFronius(url)
	if err != nil {
		return err
	}

	coll := newCollector(a)

	reg := prometheus.NewRegistry()
	if err := reg.Register(coll); err != nil {
		return err
	}

	// go module build info.
	if err := reg.Register(collectors.NewBuildInfoCollector()); err != nil {
		return err
	}
	if err := reg.Register(collectors.NewGoCollector()); err != nil {
		return err
	}

	// Install the logger handler with default output on the console
	c := alice.New()
	c = c.Append(hlog.NewHandler(log))

	// Expose the registered metrics via HTTP.
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true,
		},
	))

	c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Stringer("url", r.URL).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("")
	}))

	return http.ListenAndServe(addr, c.Then(mux))
}

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("failed")
	}
}
