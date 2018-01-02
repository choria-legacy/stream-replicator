# choria/stream_replicator

## Overview

The Choria NATS Stream Replicator is a tool to replicate a topic from one NATS Streaming cluster to another.  It's has different modes of operation for preserving order or scaling horizontally and vertically.

## Module Description

The module installs, configures and manages the associated services.

In order to install the package you have to add the Choria YUM repository to your system, in future there will be a `choria` module to do this for you.

```ini
[choria]
name=Choria Orchestrator - $architecture
baseurl=https://dl.bintray.com/choria/el-yum/el$releasever/$basearch
gpgcheck=0
repo_gpgcheck=0
enabled=1
protect=1
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