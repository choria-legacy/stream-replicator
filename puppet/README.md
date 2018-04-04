# choria/stream_replicator

## Overview

The Choria NATS Stream Replicator is a tool to replicate a topic from one NATS Streaming cluster to another.  It's has different modes of operation for preserving order or scaling horizontally and vertically.

## Module Description

The module installs, configures and manages the associated services.

In order to install the package you have to add the Choria YUM repository to your system, the `choria/choria` module can do this, you can also arrange for it to be installed on your own as below:

```ini
[choria_release]
name=choria_release
baseurl=https://packagecloud.io/choria/release/el/$releasever/$basearch
repo_gpgcheck=1
gpgcheck=0
enabled=1
gpgkey=https://packagecloud.io/choria/release/gpgkey
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
metadata_expire=300
```

## Usage

```puppet
class{"stream_replicator":
  managed_topics          => ["cmdb", "jobs"],
  topics                  => {
    "cmdb"                => {
      "topic"             => "acme.cmdb",
      "source_url"        => "nats://nats1.dc1.acme.net:4222",
      "source_cluster_id" => "dc1",
      "target_url"        => "nats://nats1.dc2.acme.net:4222",
      "target_cluster_id" => "dc2",
    },
    "cmdb"                => {
      "topic"             => "acme.jobs",
      "source_url"        => "nats://nats1.dc1.acme.net:4222",
      "source_cluster_id" => "dc1",
      "target_url"        => "nats://nats1.dc2.acme.net:4222",
      "target_cluster_id" => "dc2",
    },
  }
}
```

Full reference about the available options for configuring topics can be found in the project documentation.