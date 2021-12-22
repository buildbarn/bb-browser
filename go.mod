module github.com/buildbarn/bb-browser

go 1.16

replace github.com/gordonklaus/ineffassign => github.com/gordonklaus/ineffassign v0.0.0-20201223204552-cba2d2a1d5d9

require (
	github.com/bazelbuild/remote-apis v0.0.0-20211004185116-636121a32fa7
	github.com/buildbarn/bb-remote-execution v0.0.0-20211222101503-592ecb371dfd
	github.com/buildbarn/bb-storage v0.0.0-20211205205823-634fb8ef62e0
	github.com/buildkite/terminal-to-html v3.2.0+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	gonum.org/v1/plot v0.10.0
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.27.1
)
