module github.com/buildbarn/bb-browser

go 1.15

replace github.com/gordonklaus/ineffassign => github.com/gordonklaus/ineffassign v0.0.0-20201223204552-cba2d2a1d5d9

require (
	github.com/bazelbuild/remote-apis v0.0.0-20210301152524-6345202a036a
	github.com/buildbarn/bb-remote-execution v0.0.0-20210302115032-475a5a66f8bb // indirect
	github.com/buildbarn/bb-storage v0.0.0-20210226075542-a0427981f170
	github.com/buildkite/terminal-to-html v3.2.0+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	google.golang.org/grpc v1.36.0
)
