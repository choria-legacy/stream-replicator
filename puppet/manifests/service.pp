class stream_replicator::service {
    if $stream_replicator::service_mode == "systemd" {
        include stream_replicator::systemd_service
        Class[$name] ~> Class["stream_replicator::systemd_service"]
    }

    if $stream_replicator::service_mode == "defaults" {
        include stream_replicator::defaults_service
        Class[$name] ~> Class["stream_replicator::defaults_service"]
    }
}