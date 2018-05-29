type Stream_replicator::FileSSL = Struct[{
  identity => String,
  scheme => Enum["file", "manual"],
  ca => Stdlib::Unixpath,
  cert => Stdlib::Unixpath,
  key => Stdlib::Unixpath
}]
