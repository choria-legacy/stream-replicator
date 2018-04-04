class stream_replicator::systemd_service {
    if $stream_replicator::ensure == "present" {
        $_sensure = "running"
        $_senable = true
    } else {
        $_sensure = "stopped"
        $_senable = false
    }

    $stream_replicator::managed_topics.each |$topic| {
        service{"${stream_replicator::service_name}@${topic}":
            ensure => $_sensure,
            enable => $_senable
        }
    }

    $stream_replicator::disabled_topics.each |$topic| {
        service{"${stream_replicator::service_name}@${topic}":
            ensure => "stopped",
            enable => false
        }
    }

    Class["stream_replicator::install"] ~> Class[$name]
    Class["stream_replicator::config"] ~> Class[$name]
}
