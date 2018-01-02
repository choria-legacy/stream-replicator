class stream_replicator::defaults_service {
    file{$stream_replicator::defaults_file:
        content => epp("stream_replicator/defaults.epp")
    } ~>

    service{$stream_replicator::service_name:
        ensure => $stream_replicator::ensure,
        enable => $stream_replicator::ensure
    }
}