load("@bazel_gazelle//:deps.bzl", "go_repository")

def bb_browser_go_dependencies():
    go_repository(
        name = "com_github_buildkite_terminal",
        importpath = "github.com/buildkite/terminal",
        patches = ["@com_github_buildbarn_bb_browser//:patches/com_github_buildkite_terminal/assets.diff"],
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
        name = "com_github_grpc_ecosystem_go_grpc_middleware",
        importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
        sha256 = "e9178768b55709d2fc2b5a509baceccb4e51d841fa13ed409e16455435e6917b",
        strip_prefix = "go-grpc-middleware-1.0.0",
        urls = ["https://github.com/grpc-ecosystem/go-grpc-middleware/archive/v1.0.0.tar.gz"],
    )

    go_repository(
        name = "com_github_kballard_go_shellquote",
        commit = "95032a82bc518f77982ea72343cc1ade730072f0",
        importpath = "github.com/kballard/go-shellquote",
    )

    go_repository(
        name = "com_github_tiziano88_elm_protobuf",
        commit = "b76b05c2ab22f89475cc038b2a2f358ca5b7808f",
        importpath = "github.com/tiziano88/elm-protobuf",
        patches = [
            "@com_github_buildbarn_bb_browser//:patches/com_github_tiziano88_elm_protobuf/bytes.diff",
            "@com_github_buildbarn_bb_browser//:patches/com_github_tiziano88_elm_protobuf/proto2.diff",
            "@com_github_buildbarn_bb_browser//:patches/com_github_tiziano88_elm_protobuf/recursive-type.diff",
            "@com_github_buildbarn_bb_browser//:patches/com_github_tiziano88_elm_protobuf/silent.diff",
        ],
    )
