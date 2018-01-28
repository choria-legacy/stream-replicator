type Stream_replicator::Advisory = Struct[{
    target => String,
    cluster => Enum["cluster", "source"],
    age => Pattern[/\d+(m|h)/]
}]