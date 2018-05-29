class stream_replicator::install {
    if $stream_replicator::ensure == "present" {
        $ensure = $stream_replicator::version
    } else {
        $ensure = "absent"
    }

    package{$stream_replicator::package_name:
        ensure => $ensure
    }
}
