require "securerandom"
require "yaml"

task :default => [:test]

desc "Run just tests no measurements"
task :test do
  sh "ginkgo -r -skipMeasurements ."
end

desc "Builds packages"
task :build do
    version = ENV["VERSION"] || "development"
    sha = `git rev-parse --short HEAD`.chomp
    buildid = SecureRandom.hex
    build = ENV["BUILD"] || "foss"

    source = "/go/src/github.com/choria-io/stream-replicator"

    ["el6_32", "el6_64", "el7_64"].each do |pkg|
        sh 'docker run --rm -v `pwd`:%s -e SOURCE_DIR=%s -e ARTIFACTS=%s -e SHA1="%s" -e BUILD="%s" -e VERSION="%s" -e PACKAGE=%s ripienaar/choria-packager:el7-go9.2-puppet' % [
            source,
            source,
            source,
            sha,
            build,
            version,
            pkg
        ]
    end
end

desc "Builds binaries"
task :build_binaries do
    version = ENV["VERSION"] || "development"
    sha = `git rev-parse --short HEAD`.chomp
    buildid = SecureRandom.hex
    build = ENV["BUILD"] || "foss"

    source = "/go/src/github.com/choria-io/stream-replicator"

    sh 'docker run --rm  -v `pwd`:%s -e SOURCE_DIR=%s -e ARTIFACTS=%s -e SHA1="%s" -e BUILD="%s" -e VERSION="%s" -e BINARY_ONLY=1 ripienaar/choria-packager:el7-go9.2-puppet' % [
        source,
        source,
        source,
        sha,
        build,
        version
    ]
end
