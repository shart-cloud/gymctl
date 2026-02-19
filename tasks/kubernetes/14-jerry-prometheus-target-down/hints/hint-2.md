Prometheus can only scrape what the Service resolves to.

If `metrics-exporter` has no endpoints, the target will stay DOWN no matter what the scrape config says.
