workspace(name = "com_github_buildbarn_bb_browser")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "bazel_toolchains",
    sha256 = "ee854b5de299138c1f4a2edb5573d22b21d975acfc7aa938f36d30b49ef97498",
    strip_prefix = "bazel-toolchains-37419a124bdb9af2fec5b99a973d359b6b899b61",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-toolchains/archive/37419a124bdb9af2fec5b99a973d359b6b899b61.tar.gz",
        "https://github.com/bazelbuild/bazel-toolchains/archive/37419a124bdb9af2fec5b99a973d359b6b899b61.tar.gz",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
    strip_prefix = "rules_docker-0.7.0",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "ade51a315fa17347e5c31201fdc55aa5ffb913377aa315dceb56ee9725e620ee",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.16.6/rules_go-0.16.6.tar.gz",
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "7949fc6cc17b5b191103e97481cf8889217263acf52e00b560683413af204fcb",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.16.0/bazel-gazelle-0.16.0.tar.gz"],
)

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "com_github_buildbarn_bb_storage",
    commit = "91e95ab36b90259bea60e5b525182694e7450349",
    remote = "git@github.com:buildbarn/bb-storage.git",
)

load("@io_bazel_rules_docker//repositories:repositories.bzl", container_repositories = "repositories")

container_repositories()

load("@io_bazel_rules_go//go:def.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

load("@com_github_buildbarn_bb_storage//:go_dependencies.bzl", "bb_storage_go_dependencies")

bb_storage_go_dependencies()

go_repository(
    name = "com_github_buildkite_terminal",
    importpath = "github.com/buildkite/terminal",
    sha256 = "5d0203bb4dd007ad607df7d0eecbe50ff4bdaa0e56e1ad2ea1eb331ff2ae5be6",
    strip_prefix = "terminal-to-html-3.1.0",
    urls = ["https://github.com/buildkite/terminal-to-html/archive/v3.1.0.tar.gz"],
)

go_repository(
    name = "com_github_gorilla_context",
    importpath = "github.com/gorilla/context",
    sha256 = "2dfdd051c238695bf9ebfed0bf6a8c533507ac0893bce23be5930e973736bb03",
    strip_prefix = "context-1.1.1",
    urls = ["https://github.com/gorilla/context/archive/v1.1.1.tar.gz"],
)

go_repository(
    name = "com_github_gorilla_mux",
    importpath = "github.com/gorilla/mux",
    sha256 = "5aca5bfa16325506b23b66ce34e2b9336a3a341b8c51ba7b0faf7d0daade0116",
    strip_prefix = "mux-1.7.0",
    urls = ["https://github.com/gorilla/mux/archive/v1.7.0.tar.gz"],
)

go_repository(
    name = "com_github_kballard_go_shellquote",
    commit = "95032a82bc518f77982ea72343cc1ade730072f0",
    importpath = "github.com/kballard/go-shellquote",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    commit = "c12348ce28de40eed0136aa2b644d0ee0650e56c",
    importpath = "github.com/matttproud/golang_protobuf_extensions",
)
