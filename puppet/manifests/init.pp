# Choria NATS Stream Replicator
#
# Install and configure a tool to replicate NATS Streaming topics between clusters.
#
# This module does not install the Choria YUM repository, it's configured as:
#
# ```
# [choria_release]
# name=choria_release
# baseurl=https://packagecloud.io/choria/release/el/$releasever/$basearch
# repo_gpgcheck=1
# gpgcheck=0
# enabled=1
# gpgkey=https://packagecloud.io/choria/release/gpgkey
# sslverify=1
# sslcacert=/etc/pki/tls/certs/ca-bundle.crt
# metadata_expire=300
# ```
#
# Each topic you wish to manage gets one service or process.  On `systemd` type
# systems if you wish to stop running a certain topic replicator you have to add
# it to the `disabled_topics` list so the service gets disabled on next run.  On
# `defaults` type this is handled by the init script for you.
#
# @param topics Known topic configuration
# @param managed_topics List of known topics to start replicators for
# @param disabled_topics List of topics to ensure the services are stopped for
# @param config_file Where the configuration file is written
# @param state_dir When tracking unique senders the state gets stored in this directory
# @param defaults_file On `defaults` type systems this is the file to manage the list of services
# @param log_file The logfile to create
# @param service_name The service name to manage
# @param service_mode The type of service this is
# @param package_name The package to install
# @param debug Enables debug logging
# @param verbose Enables verbose logging
# @param ensure Install or remove the software
# @param version Version to install
class stream_replicator(
    Stream_replicator::Topics $topics = {},
    Array[String] $managed_topics = [],
    Array[String] $disabled_topics = [],
    Optional[Variant[Stream_replicator::FileSSL, Stream_replicator::PuppetSSL]] $tls = undef,
    Stdlib::Absolutepath $config_file = "/etc/stream-replicator/sr.yaml",
    Stdlib::Absolutepath $state_dir = "/var/lib/stream-replicator",
    Stdlib::Absolutepath $defaults_file = "/etc/sysconfig/stream-replicator",
    Stdlib::Absolutepath $log_file = "/var/log/stream-replicator.log",
    String $service_name = "stream-replicator",
    Enum[defaults, systemd] $service_mode = "systemd",
    String $package_name = "stream-replicator",
    Boolean $debug = false,
    Boolean $verbose = false,
    Enum[present, absent] $ensure = "present",
    String $version = "present"
) {
    if $ensure == "present" {
        class{"stream_replicator::install": }
        -> class{"stream_replicator::config": }
        ~> class{"stream_replicator::service": }
    } else {
        class{"stream_replicator::service": }
        -> class{"stream_replicator::install": }
    }
}
