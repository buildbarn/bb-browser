module github.com/buildbarn/bb-browser

go 1.16

replace github.com/gordonklaus/ineffassign => github.com/gordonklaus/ineffassign v0.0.0-20201223204552-cba2d2a1d5d9

require (
	github.com/bazelbuild/remote-apis v0.0.0-20210505181611-ce7036ef5417
	github.com/buildbarn/bb-remote-execution v0.0.0-20210612211840-40946a11cb84 // indirect
	github.com/buildbarn/bb-storage v0.0.0-20210612140245-0c202af9f01f
	github.com/buildkite/terminal-to-html v3.2.0+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	gonum.org/v1/plot v0.9.0 // indirect
	google.golang.org/grpc v1.36.0
	google.golang.org/protobuf v1.26.0
)
