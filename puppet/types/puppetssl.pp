type Stream_replicator::PuppetSSL = Struct[{
  identity => String,
  scheme => Enum["puppet"],
  ssl_dir => Stdlib::Unixpath
}]
