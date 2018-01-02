type Stream_replicator::Topic = Struct[{
  topic => String,
  source_url => Pattern[/^nats:\/\//],
  source_cluster_id => String,
  target_url => Pattern[/^nats:\/\//],
  target_cluster_id => String,
  workers => Optional[Integer],
  queued => Optional[Boolean],
  queue_group => Optional[String],
  inspect => Optional[String],
  age => Optional[Pattern[/\d+(m|h)/]],
  monitor => Optional[Integer],
  name => Optional[String]
}]