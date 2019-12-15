workspace(name = "com_github_buildbarn_bb_browser")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive", "http_file")

http_archive(
    name = "bazel_toolchains",
    sha256 = "28cb3666da80fbc62d4c46814f5468dd5d0b59f9064c0b933eee3140d706d330",
    strip_prefix = "bazel-toolchains-0.27.1",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-toolchains/archive/0.27.1.tar.gz",
        "https://github.com/bazelbuild/bazel-toolchains/archive/0.27.1.tar.gz",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "14ac30773fdb393ddec90e158c9ec7ebb3f8a4fd533ec2abbfd8789ad81a284b",
    strip_prefix = "rules_docker-0.12.1",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.12.1/rules_docker-v0.12.1.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "e88471aea3a3a4f19ec1310a55ba94772d087e9ce46e41ae38ecebe17935de7b",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/v0.20.3/rules_go-v0.20.3.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.20.3/rules_go-v0.20.3.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "86c6d481b3f7aedc1d60c1c211c6f76da282ae197c3b3160f54bd3a8f847896f",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.1/bazel-gazelle-v0.19.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.1/bazel-gazelle-v0.19.1.tar.gz",
    ],
)

http_archive(
    name = "com_github_bazelbuild_bazel",
    patches = ["//:patches/com_github_bazelbuild_bazel/build_event_stream.diff"],
    sha256 = "2cea463d611f5255d2f3d41c8de5dcc0961adccb39cf0ac036f07070ba720314",
    urls = ["https://github.com/bazelbuild/bazel/releases/download/0.28.1/bazel-0.28.1-dist.zip"],
)

http_archive(
    name = "com_github_twbs_bootstrap",
    build_file_content = """exports_files(["css/bootstrap.min.css", "js/bootstrap.min.js"])""",
    sha256 = "888ffd30b7e192381e2f6a948ca04669fdcc2ccc2ba016de00d38c8e30793323",
    strip_prefix = "bootstrap-4.3.1-dist",
    urls = ["https://github.com/twbs/bootstrap/releases/download/v4.3.1/bootstrap-4.3.1-dist.zip"],
)

http_file(
    name = "com_jquery_jquery",
    sha256 = "0497a8d2a9bde7db8c0466fae73e347a3258192811ed1108e3e096d5f34ac0e8",
    urls = ["https://code.jquery.com/jquery-3.4.0.min.js"],
)

http_archive(
    name = "com_github_buildbarn_bb_deployments",
    sha256 = "cf910624a50d3f1f4c8af98d96f4ff7cbdd51f2c107315ac835256776a41df1c",
    strip_prefix = "bb-deployments-cf505b7f363ce87798a367d6741b8f550a5e077a",
    url = "https://github.com/buildbarn/bb-deployments/archive/cf505b7f363ce87798a367d6741b8f550a5e077a.tar.gz",
)

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

# TODO: Use the copy from Git!
local_repository(
    name = "com_github_buildbarn_bb_storage",
    path = "../bb-storage-staging",
)

load("@io_bazel_rules_docker//repositories:repositories.bzl", container_repositories = "repositories")

container_repositories()

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

load("@io_bazel_rules_docker//go:image.bzl", _go_image_repos = "repositories")

_go_image_repos()

load("//:go_dependencies.bzl", "bb_browser_go_dependencies")

bb_browser_go_dependencies()

load("@com_github_buildbarn_bb_storage//:go_dependencies.bzl", "bb_storage_go_dependencies")

bb_storage_go_dependencies()

load("@com_github_bazelbuild_remote_apis//:repository_rules.bzl", "switched_rules_by_language")

switched_rules_by_language(
    name = "bazel_remote_apis_imports",
    go = True,
)

http_archive(
    name = "com_google_protobuf",
    sha256 = "761bfffc7d53cd01514fa237ca0d3aba5a3cfd8832a71808c0ccc447174fd0da",
    strip_prefix = "protobuf-3.11.1",
    urls = ["https://github.com/protocolbuffers/protobuf/releases/download/v3.11.1/protobuf-all-3.11.1.tar.gz"],
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()
