package main

import (
	"fmt"
	"io"
	"time"
)

// WritePrometheusMetrics 把快照渲染为 Prometheus 文本 exposition 格式。
func WritePrometheusMetrics(w io.Writer, s *Signal, t time.Duration) {
	fmt.Fprintln(w, "# HELP aidc_device_scrape_timestamp_seconds Mock virtual time in seconds")
	fmt.Fprintln(w, "# TYPE aidc_device_scrape_timestamp_seconds gauge")
	fmt.Fprintf(w, "aidc_device_scrape_timestamp_seconds %.3f\n\n", t.Seconds())

	fmt.Fprintln(w, "# HELP aidc_interface_oper_status Current operational status (1=UP, 0=DOWN)")
	fmt.Fprintln(w, "# TYPE aidc_interface_oper_status gauge")
	for _, ifc := range ifaces {
		val := 0
		if s.operStatus(t) == "UP" {
			val = 1
		}
		fmt.Fprintf(w, "aidc_interface_oper_status{device=\"mock-leaf1\",interface=\"%s\"} %d\n", ifc, val)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "# HELP aidc_interface_out_octets_total Outbound octets counter")
	fmt.Fprintln(w, "# TYPE aidc_interface_out_octets_total counter")
	for _, ifc := range ifaces {
		fmt.Fprintf(w, "aidc_interface_out_octets_total{device=\"mock-leaf1\",interface=\"%s\"} %d\n", ifc, uint64(s.baseOctets(t)))
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "# HELP aidc_interface_out_discards_total Outbound discarded packets")
	fmt.Fprintln(w, "# TYPE aidc_interface_out_discards_total counter")
	for _, ifc := range ifaces {
		discards := uint64(accumulate(t, s.StormPeriod, s.StormStart, s.StormLen, s.StormPausePPS*0.02))
		fmt.Fprintf(w, "aidc_interface_out_discards_total{device=\"mock-leaf1\",interface=\"%s\"} %d\n", ifc, discards)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "# HELP aidc_interface_out_pfc_pause_frames_total Priority Flow Control pause frames transmitted")
	fmt.Fprintln(w, "# TYPE aidc_interface_out_pfc_pause_frames_total counter")
	for _, ifc := range ifaces {
		pause := uint64(accumulate(t, s.StormPeriod, s.StormStart, s.StormLen, s.StormPausePPS))
		fmt.Fprintf(w, "aidc_interface_out_pfc_pause_frames_total{device=\"mock-leaf1\",interface=\"%s\",priority=\"%s\"} %d\n", ifc, queueName, pause)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "# HELP aidc_interface_ecn_marked_pkts_total Packets marked with Congestion Experienced (ECN)")
	fmt.Fprintln(w, "# TYPE aidc_interface_ecn_marked_pkts_total counter")
	for _, ifc := range ifaces {
		ecn := uint64(accumulate(t, s.StormPeriod, s.StormStart, s.StormLen, s.StormECNPPS))
		fmt.Fprintf(w, "aidc_interface_ecn_marked_pkts_total{device=\"mock-leaf1\",interface=\"%s\",priority=\"%s\"} %d\n", ifc, queueName, ecn)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "# HELP aidc_queue_watermark_bytes Queue peak watermark in bytes")
	fmt.Fprintln(w, "# TYPE aidc_queue_watermark_bytes gauge")
	for _, ifc := range ifaces {
		fmt.Fprintf(w, "aidc_queue_watermark_bytes{device=\"mock-leaf1\",interface=\"%s\",queue=\"%s\"} %d\n", ifc, queueName, s.watermark(t))
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "# HELP aidc_queue_transmit_pkts_total Queue transmitted packets")
	fmt.Fprintln(w, "# TYPE aidc_queue_transmit_pkts_total counter")
	for _, ifc := range ifaces {
		tx := uint64(accumulate(t, s.StormPeriod, s.StormStart, s.StormLen, s.StormPausePPS))
		fmt.Fprintf(w, "aidc_queue_transmit_pkts_total{device=\"mock-leaf1\",interface=\"%s\",queue=\"%s\"} %d\n", ifc, queueName, tx)
	}
}
