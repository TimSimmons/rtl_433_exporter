FROM ubuntu:20.04

COPY rtl_433_exporter /rtl_433_exporter

ENTRYPOINT ["/rtl_433_exporter"]
