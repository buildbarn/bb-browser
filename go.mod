module github.com/buildbarn/bb-browser

go 1.16

replace github.com/gordonklaus/ineffassign => github.com/gordonklaus/ineffassign v0.0.0-20201223204552-cba2d2a1d5d9

require (
	github.com/bazelbuild/remote-apis v0.0.0-20210812183132-3e816456ee28
	github.com/buildbarn/bb-remote-execution v0.0.0-20210909134944-9c4e93e5b5ee
	github.com/buildbarn/bb-storage v0.0.0-20210909130924-6c0280c9d0f2
	github.com/buildkite/terminal-to-html v3.2.0+incompatible
	github.com/dustin/go-humanize v1.0.0
	github.com/gorilla/mux v1.8.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	gonum.org/v1/plot v0.9.0
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
)
