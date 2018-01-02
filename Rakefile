require "securerandom"

desc "Builds packages"
task :build do
    version = ENV["VERSION"] || "development"
    sha = `git rev-parse --short HEAD`.chomp

    buildid = SecureRandom.hex
    flags = [
        "-X github.com/choria-io/stream-replicator/cmd.version=%s" % version,
        "-X github.com/choria-io/stream-replicator/cmd.sha=%s" % sha,
        "-B 0x%s" % buildid
    ]

    [["linux", "amd64"], ["linux", "386"]].each do |os, arch|
        puts ">>> Compiling %s %s" % [os, arch]
        
        ENV["GOOS"] = os
        ENV["GOARCH"] = arch
      
        args = [
            "-o %s" % output_name(version, os, arch),
            "-ldflags '%s'" % flags.join(" ")
        ]

        sh "go build %s" % args.join(" ")
    end
end

def output_name(version, os, arch)
    return ENV["OUTPUT"] if ENV["OUTPUT"]

    "build/%s-%s-%s-%s" % [File.basename(File.dirname(__FILE__)), version, os.downcase, arch]
end
