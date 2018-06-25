# NATS Streaming Topic Replicator

This is a tool that consumes 1 topic in a NATS Streaming server and replicates it to another NATS Streaming server.

## Features

  * Order preserving single worker topic replication
  * Vertically and Horizontally scalable worker pools for unordered replication
  * Filtering out high frequency updates of similar data to reduce traffic in a multi site feeding a single global store scenario.
  * Metrics about the performance and throughput are optionally exposed in Prometheus format

When there is only 1 worker in a un-queued setup this will create a Durable subscription in the origin NATS Stream with either a generated name or one you configure.

When there are many workers or it's specified to belong to a queue group a Durable Queue Group is created with either a generated name or ones you specify.

First time it connects it attempts to replicate all messages, as the subscription is Durable it will from then on continue where it left off.

## Requirements

For reliable operation where connection issues get correctly detected NATS Streaming Server version 0.10.0 or newer is required

## Status

This is a pretty new project that is being used in production, however as it is new and developing use with caution. I'd love any feedback you might have - especially design ideas about multi worker order preserving replication!

Feature wise I have a few TODOs:

  * Add a control plane by embedding a Choria server

Initial packages for el6 and 7 64bit systems are now on the Choria YUM repository,
see below.

[![CircleCI](https://circleci.com/gh/choria-io/stream-replicator/tree/master.svg?style=svg)](https://circleci.com/gh/choria-io/stream-replicator/tree/master)

## Configuration

A single configuration file can be used to configure multiple instances of the replicator.

You'll run one or more processes per replicated topic but not 1 process for multiple topics - unless you use wildcards.

Configuration is done using a YAML file:

```yaml
debug: false                     # default
verbose: false                   # default
logfile: "/path/to/logfile"      # STDOUT default
state_dir: "/path/to/statedir"   # optional
topics:
    cmdb:
        topic: acme.cmdb
        source_url: nats://source1:4222,nats://source2:4222
        source_cluster_id: dc1
        target_url: nats://target1:4222,nats://target2:4222
        target_cluster_id: dc2
        workers: 10              # optional
        queued: true             # optional
        queue_group: cmdb        # optional
        inspect: host            # optional
        age: 1h                  # optional
        monitor: 10000           # optional
        name: cmdb_replicator    # optional
```

You would then run the replicator with `stream-replicator --config sr.yaml --topic cmdb`

## TLS to the NATS infrastructure

SSL is supported on the network connections, 2 modes of configuration exist - Puppet compatible or full manual config.

The examples below will show a top level `tls` key, you can also put it at a individual topic level if needed.

### Puppet Compatible

If you are a Puppet user you might want to re-use the Puppet CA, a sample SSL configuration can be seen here:

```yaml
tls:
  identity: foo.replicator
  scheme: puppet
  ssl_dir: /etc/stream-replicator/ssl

topics:
    cmdb:
        topic: acme.cmdb
        source_url: nats://source1:4222,nats://source2:4222
        source_cluster_id: dc1
        target_url: nats://target1:4222,nats://target2:4222
        target_cluster_id: dc2
```

Here it will attempt to use a Puppet standard layout for the various certificates.

You can arrange to put the certificate, keys etc there or use the enroll command, it will create the private key and CSR and send it to the Puppet CA and then wait until it gets signed (up to 30 minutes).

```
# stream-replicator enroll foo.replicator --dir /etc/stream-replicator/ssl
Attempting to download certificate for bob, try 1.
Attempting to download certificate for bob, try 2.
Attempting to download certificate for bob, try 3.
Attempting to download certificate for bob, try 4.
```

### Full Manual Config

If you have your own CA you can use this too, we can't help you enroll in that that, you would configure it like this:

```yaml
tls:
  identity: foo.replicator
  scheme: manual
  ca: /path/to/ca.pem
  cert: /path/to/cert.pem
  key: /path/to/key.pem

topics:
    cmdb:
        topic: acme.cmdb
        source_url: nats://source1:4222,nats://source2:4222
        source_cluster_id: dc1
        target_url: nats://target1:4222,nats://target2:4222
        target_cluster_id: dc2
```

## Replicating a topic, preserving order

The most obvious thing you'd want to do is replicate one topic between clusters and preserve order - not sequence IDs, that's not possible.

Due to how NATS Streaming work it's possible but with a few caveats:

  1. You can only have 1 worker subscribed to the topic
  1. You cannot have this worker belong to any queue group

This means you'll be limited in throughput and it might not work well for very busy streams.  As far as I can tell this is unavoidable, you have to code your apps to be resilient to out of order messages in distributed systems so presumably this is not a huge limiting factor, but if you have to it can be done with a topic config like this:

```yaml
topics:
    cmdb:
        topic: acme.cmdb
        source_url: nats://source1:4222,nats://source2:4222
        source_cluster_id: dc1
        target_url: nats://target1:4222,nats://target2:4222
        target_cluster_id: dc2
        workers: 1
```

You can specify `monitor` and inspect settings but no workers or queue related settings.

The `workers` here is optional but it's handy to set it to 1 specifically to enforce the intent.

NOTE: With a more complex replicator - one that builds its own buffer of messages you could make this scale better but its a significantly more complex piece of software in that case.

## Scaled topic replication

If you have a topic and order does not matter in it - like regular node registration data or metrics perhaps - you can have many workers on one or more machines all sharing the load of replicating the topic.

In that case ordering is not guaranteed but you will get much higher throughput.

```yaml
topics:
    cmdb:
        topic: acme.cmdb
        source_url: nats://source1:4222,nats://source2:4222
        source_cluster_id: dc1
        target_url: nats://target1:4222,nats://target2:4222
        target_cluster_id: dc2
        workers: 10
```

This will automatically create a Queue Group name based on the hostname and topic and so all workers will belong to the same group.  If you set up the same configuration on a 2nd node it too will join the same group and so load share the replication duty.

See the notes below about client names and queue group names though if you wish to scale this to multiple nodes.

## Inspecting and limiting replication of duplicated data

I intend to use this with Choria's NATS Stream adapter to build a registration database.  My nodes will publish their metadata regularly but outside of the local network I don't really need it that regular.

```
                   /--------\
                   | stream |
                   \--------/
                 /            \
               /   once / hour  \
             /                    \
      /--------\                /--------\
      | stream |                | stream |   ...... [ x 10s of sites ]
      \--------/                \--------/

  /////||||||||\\\\\        /////||||||||\\\\\
    5 min interval            5 min interval
      node data                  node data
```

In the above scenario I get data from my nodes very frequently and the combined stream would overwhelm my main layer data processors due to the amount of downstream sources.

I will thus have a high frequency processor - and so very fresh data - in every data center but the central one gets a 1 hourly update from all nodes only.

The replicator can facilitate this by inspecting the JSON data in the messages for a specific uniquely identifying field - like a hostname - and only publishing data upward if it has not yet seen that machine in the past hour.

This works well it means new registration data goes up immediately and a regular configurable heartbeat flows through the entire system.  With in-datacenter data being fresh where high throughput automations depend on fresh data.  See the sample graph down by Metrics where one can see the effect of the local traffic vs replicated traffic with a 30 minute setting.

At present the data store for the last-seen data is in memory only so this only works on a single node scenario (but supports many workers), in future perhaps we can support something like `etcd` or `redis` to store that data.

It does however support storing the state every 30 seconds and on shutdown to a file per topic in `state_dir` and it will read these files on startup.  On a large site with 10s of thousands of unique senders this greatly reduce the restart cost, but on a small site it's probably not worth bothering with, unless you really care and things will break if you get more updates per `age` than configured.

```yaml
state_dir: /var/cache/stream-replicator

topics:
    dc1_cmdb:
        topic: acme.cmdb
        source_url: nats://source1:4222,nats://source2:4222
        source_cluster_id: dc1
        target_url: nats://target1:4222,nats://target2:4222
        target_cluster_id: dc2
        inspect: sender
        age: 1h
```

Additionally you might want to flag that your data has changed - perhaps it's data like node metadata that changes very infrequently but when it does you'd like to replicate it unconditionally - this can be achieved by adding a boolean flag to your data and configuring the `update_flag` item:

```yaml
state_dir: /var/cache/stream-replicator

topics:
    dc1_cmdb:
        # as above
        inspect: sender
        update_flag: updated
        age: 1h
```

Here it will look at the `updated` key in your data and if it's true replicate the data regardless of time stamps.  It will mark the data as replicated though and then fall back into its standard interval behavior from that point onward.

A companion feature to this one lets you send advisories about when machines stop responding, since with this enabled internally every sender is tracked we can use this to also identify nodes that did not send data in a given interval and then send alerts.

```yaml
topics:
    dc1_cmdb:
        # as above
        inspect: sender
        age: 1h
        advisory:
          target: sr.advisories.cmdb
          cluster: target  # or source
          age: 30m
```

Now advisories will be sent to the NATS Streaming target `sr.advisories.cmdb`:

 * The first time a node is seen to not have responded in 30 minutes.  A `timeout` event is sent.
 * If the node is seen again before max age exceeds. A `recover` event is sent.
 * When the node exceeds the max age for the topic - 1 hour here. An `expire` event is sent.


Once a node is past max age we forget it ever existed - but when it comes back it's data will immediately be replicated so you can do recovery logic that way for really long outages.

A sample advisory can be seen below:

```json
{
    "$schema":"https://choria.io/schemas/sr/v1/age_advisory.json",
    "inspect":"sender",
    "value":"test",
    "age":1000,
    "seen":1516987895,
    "replicator":"testing",
    "timestamp":1516987895,
    "event":"timeout"
}
```

**NOTE**: Advisories that fail to send are retried for 10 times, but after that they are discarded

## About client and queue group names

By default if you replicate topic `acme.cmdb` and the config name is `cmdb` the client name for the replicator will be `dc1_cmdb_acme_cmdb_stream_replicator_n` where `n` is the number of the worker.

If a group is needed - like when workers is > 1 or `queued` is set the group will be called `acme_cmdb_stream_replicator_grp`.

This works fine for almost all cases out of the box if you want to scale your workers across multiple machines you might need to set custom names, something like:

```yaml
topics:
    cmdb:
        topic: acme.cmdb
        source_url: nats://source1:4222,nats://source2:4222
        source_cluster_id: dc1
        target_url: nats://target1:4222,nats://target2:4222
        target_cluster_id: dc2
        workers: 10
        name: cmdb_replicatorA
        queue_group: cmdb
```

This will start workers with names `cmdb_replicatorA_0`...`cmdb_replicatorA_9` all belonging to the same queue group.  On another node set `name: cmdb_replicatorB` and so forth.

NOTE: This is likely to change in future releases, right now I don't need multi node scaled replicators so once I do this will be made easier (or file a issue)

## Prometheus Metrics

Stats are exposed as prometheus metrics, some info about what gets exposed below:

In all cases the `name` label is the configured name or generated one as described above, the `worker` label is the unique name per worker.

|Stat|Comments|
|----|--------|
|`stream_replicator_received_msgs`|How many messages were received. The difference between this and `stream_replicator_copied_msgs` is how many limiters skipped|
|`stream_replicator_received_bytes`|The size of messages that were received|
|`stream_replicator_copied_msgs`|A Counter indicating how many messages were copied|
|`stream_replicator_copied_bytes`|A Counter indicating the size of that messages were copied|
|`stream_replicator_failed_msgs`|How many messages failed to copy|
|`stream_replicator_acks_failed`|How many times did sending the ack fail|
|`stream_replicator_processing_time`|How long it takes to do the processing per message including ack'ing it to the source|
|`stream_replicator_connection_reconnections`|How many times did the NATS connection reconnect|
|`stream_replicator_connection_closed`|How many times did the NATS connection close|
|`stream_replicator_connection_errors`|How many times did the NATS connection encounter errors|
|`stream_replicator_current_sequence`|The current sequence per worker, this is kind of not useful in pooled workers since messages are tried in any order, but in a single worker scenario this can help you discover how far behind you are|
|`stream_replicator_limiter_memory_seen`|When inspecting the messages this shows the current size of the known list in the memory limiter - the list is scrubbed every `age` + 10 minutes of within that time.|
|`stream_replicator_limiter_memory_skipped`|Number of times the memory limiter determined a message should be skipped|
|`stream_replicator_limiter_memory_passed`|Number of times the memory limiter allowed a message to be processed|
|`stream_replicator_limiter_memory_errors`|Number of times the processor function returned an error|
|`stream_replicator_advisories_timeout`|Number of advisories that were sent when nodes went down|
|`stream_replicator_advisories_recover`|Number of advisories that were sent when nodes recovered before expiry deadline|
|`stream_replicator_advisories_expire`|Number of advisories sent when nodes expired before the deadline|
|`stream_replicator_advisories_errors`|Number of times publishing advisories failed|

A sample Grafana dashboard can be found in [dashboard.json](dashboard.json), it will make a graph along these lines:

![](stream-replicator.png)

## Packages

RPMs are hosted in the Choria yum repository for el6 and 7 64bit systems:

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

On a RHEL7 system the systemd unit files are using templating, if you have a configuration section for `cmdb` you would run that using `systemctl start stream-replicator@cmdb`.

On a RHEL6 system you can edit `/etc/sysconfig/stream-replicator` and set `TOPICS="cmdb monitor"` to start a instance for the configured topics matching the names.

## Nightly Builds

Nightly RPMs are published for EL7 64bit in the following repo:

```ini
[choria_nightly]
name=choria_nightly
baseurl=https://packagecloud.io/choria/nightly/el/$releasever/$basearch
repo_gpgcheck=1
gpgcheck=0
enabled=1
gpgkey=https://packagecloud.io/choria/nightly/gpgkey
sslverify=1
sslcacert=/etc/pki/tls/certs/ca-bundle.crt
metadata_expire=300
```

Nightly packages are versioned `0.99.0` with a date portion added: `stream-replicator-0.99.0.20180126-1.el7.x86_64.rpm`

## Puppet Module

A Puppet module to install and manage the Stream Replicator can be found on the Puppet Forge as `choria/stream_replicator`

## Thanks

<a href="https://packagecloud.io/"><img src="https://packagecloud.io/images/packagecloud-badge.png" width="158"></a>