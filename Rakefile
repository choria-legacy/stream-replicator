require "securerandom"

desc "Builds packages"
task :build do
    version = ENV["VERSION"] || "development"
    sha = `git rev-parse --short HEAD`.chomp
    buildid = SecureRandom.hex
    package = ENV["PKG"] || "el7_64"
    build = ENV["BUILD"] || "foss"

    sh 'docker run --rm -e SOURCE_DIR=/go/src/github.com/choria-io/stream-replicator -e CIRCLE_SHA1="%s" -e BUILD="%s" -e VERSION="%s" -e ARTIFACTS=/build/artifacts -e PACKAGE=%s -v `pwd`:/go/src/github.com/choria-io/stream-replicator -e BINARY_ONLY=1 ripienaar/choria-packager:el7-go9.2-puppet' % [
        sha,
        build,
        version,
        package
    ]
end