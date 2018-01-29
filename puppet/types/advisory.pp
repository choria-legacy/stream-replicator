type Stream_replicator::Advisory = Struct[{
    target => String,
    cluster => Enum["source", "target"],
    age => Pattern[/\d+(m|h)/]
}]