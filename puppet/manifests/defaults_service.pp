class stream_replicator::defaults_service {
    if $stream_replicator::ensure == "present" {
        $_sensure = "running"
        $_senable = true
    } else {
        $_sensure = "stopped"
        $_senable = false
    }

    file{$stream_replicator::defaults_file:
        content => epp("stream_replicator/defaults.epp")
    }

    ~> service{$stream_replicator::service_name:
        ensure => $_sensure,
        enable => $_senable
    }

    Class["stream_replicator::install"] ~> Class[$name]
    Class["stream_replicator::config"] ~> Class[$name]
}
