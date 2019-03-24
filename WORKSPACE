workspace(name = "com_github_buildbarn_bb_browser")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive", "http_file")

http_archive(
    name = "bazel_toolchains",
    sha256 = "109a99384f9d08f9e75136d218ebaebc68cc810c56897aea2224c57932052d30",
    strip_prefix = "bazel-toolchains-94d31935a2c94fe7e7c7379a0f3393e181928ff7",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-toolchains/archive/94d31935a2c94fe7e7c7379a0f3393e181928ff7.tar.gz",
        "https://github.com/bazelbuild/bazel-toolchains/archive/94d31935a2c94fe7e7c7379a0f3393e181928ff7.tar.gz",
    ],
)

http_archive(
    name = "build_bazel_rules_nodejs",
    sha256 = "fb87ed5965cef93188af9a7287511639403f4b0da418961ce6defb9dcf658f51",
    urls = ["https://github.com/bazelbuild/rules_nodejs/releases/download/0.27.7/rules_nodejs-0.27.7.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
    strip_prefix = "rules_docker-0.7.0",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "6776d68ebb897625dead17ae510eac3d5f6342367327875210df44dbe2aeeb19",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.17.1/rules_go-0.17.1.tar.gz"],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
)

http_archive(
    name = "com_github_bazelbuild_bazel",
    patches = ["//:patches/com_github_bazelbuild_bazel/build_event_stream.diff"],
    sha256 = "6860a226c8123770b122189636fb0c156c6e5c9027b5b245ac3b2315b7b55641",
    urls = ["https://github.com/bazelbuild/bazel/releases/download/0.22.0/bazel-0.22.0-dist.zip"],
)

http_archive(
    name = "com_github_edschouten_rules_elm",
    patches = ["//:patches/com_github_edschouten_rules_elm/bytes.diff"],
    sha256 = "0b8a4e288ce9fe255074adb07be443cdda3a9fa9667de775b01decb93507a6d7",
    strip_prefix = "rules_elm-0.3",
    urls = ["https://github.com/EdSchouten/rules_elm/archive/v0.3.tar.gz"],
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

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "com_github_buildbarn_bb_storage",
    commit = "3b214a585bac257fe81d9e40a9f468d23de25538",
    remote = "https://github.com/buildbarn/bb-storage.git",
)

load("@io_bazel_rules_docker//repositories:repositories.bzl", container_repositories = "repositories")

container_repositories()

load("@com_github_edschouten_rules_elm//elm:deps.bzl", "elm_register_toolchains")

elm_register_toolchains()

load("@com_github_edschouten_rules_elm//repository:def.bzl", "elm_repository")

elm_repository(
    name = "elm_package_avh4_elm_color",
    sha256 = "f8dfda51b7515d42442bb6a7706a28e03c5db5a159e252c757a91f74b7e52658",
    strip_prefix = "elm-color-1.0.0",
    urls = ["https://github.com/avh4/elm-color/archive/1.0.0.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_browser",
    sha256 = "6afa0d009826abd3cd83b396ecda3dfb16e40fa9b2c03d5f73e7d1278ee995fe",
    strip_prefix = "browser-1.0.1",
    urls = ["https://github.com/elm/browser/archive/1.0.1.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_bytes",
    sha256 = "922f3526e3b430e947d1d2eac5965e4caae625013649de2f657d4f258a5bdc0b",
    strip_prefix = "bytes-1.0.8",
    urls = ["https://github.com/elm/bytes/archive/1.0.8.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_core",
    sha256 = "5a891e637f310b37b26e41dce3af5ec2989e3effa595aed1ff3324fed96a18d0",
    strip_prefix = "core-1.0.2",
    urls = ["https://github.com/elm/core/archive/1.0.2.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_explorations_test",
    sha256 = "1233c0cb3d663630b939edb058d904e1275dfae0813ddf0bb63459d0cdf8bfe9",
    strip_prefix = "test-1.2.1",
    urls = ["https://github.com/elm-explorations/test/archive/1.2.1.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_file",
    sha256 = "c85b4025e12c1bf2dee9e4d853459ead7d1fa917304adfa2af27d116c86292e6",
    strip_prefix = "file-1.0.5",
    urls = ["https://github.com/elm/file/archive/1.0.5.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_html",
    sha256 = "73b885e0a3d2f9781b1c9bbcc1ee9ac032f503f5ef46a27da3ba617cebbf6fd8",
    strip_prefix = "html-1.0.0",
    urls = ["https://github.com/elm/html/archive/1.0.0.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_http",
    sha256 = "619bc23d7753bc172016ea764233dd7dfded1d919263c41b59885c5bcdd10b01",
    strip_prefix = "http-2.0.0",
    urls = ["https://github.com/elm/http/archive/2.0.0.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_json",
    sha256 = "d0635f33137e4ad3fc323f96ba280e45dc41afa51076c53d9f04fd92c2cf5c4e",
    strip_prefix = "json-1.1.3",
    urls = ["https://github.com/elm/json/archive/1.1.3.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_parser",
    sha256 = "2294a3274ee08fdb6fec78983c00d71f9516e53e175a3f7d7abc9eba76ee6c28",
    strip_prefix = "parser-1.1.0",
    urls = ["https://github.com/elm/parser/archive/1.1.0.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_random",
    sha256 = "b4b9dc99d5a064bc607684dd158199208bce51c0521b7e8a515c365e0a11168d",
    strip_prefix = "random-1.0.0",
    urls = ["https://github.com/elm/random/archive/1.0.0.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_regex",
    sha256 = "42e98d657040339c05c4001ea0f7469ec29beca5cc3c594fb1c11e0ecad53252",
    strip_prefix = "regex-1.0.0",
    urls = ["https://github.com/elm/regex/archive/1.0.0.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_time",
    sha256 = "e18bca487adec67bfe4043a33b975d81527a7732377050d0421dd86d503c906d",
    strip_prefix = "time-1.0.0",
    urls = ["https://github.com/elm/time/archive/1.0.0.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_url",
    patches = ["//:patches/elm_package_elm_url/parse-remainder.diff"],
    sha256 = "840e9d45d8a9bd64a7f76421a1de2518e02c7cbea7ed42efd380b4e875e9682b",
    strip_prefix = "url-1.0.0",
    urls = ["https://github.com/elm/url/archive/1.0.0.tar.gz"],
)

elm_repository(
    name = "elm_package_elm_virtual_dom",
    sha256 = "cf87286ed5d1b31aaf99c6a3368ccd340d1356b1973f1afe5f668c47e22b3b60",
    strip_prefix = "virtual-dom-1.0.2",
    urls = ["https://github.com/elm/virtual-dom/archive/1.0.2.tar.gz"],
)

elm_repository(
    name = "elm_package_jweir_elm_iso8601",
    sha256 = "38e831cf7ae5cc1ece7cd8aaec275df603e2e00c75542580aa0896de12450c8d",
    strip_prefix = "elm-iso8601-5.0.2",
    urls = ["https://github.com/jweir/elm-iso8601/archive/5.0.2.tar.gz"],
)

elm_repository(
    name = "elm_package_rundis_elm_bootstrap",
    sha256 = "6b16760fd62198a5ca51cbac59c14eca88ca2c5bccd7370e3678ed510631c84a",
    strip_prefix = "elm-bootstrap-5.1.0",
    urls = ["https://github.com/rundis/elm-bootstrap/archive/5.1.0.tar.gz"],
)

elm_repository(
    name = "elm_package_tiziano88_elm_protobuf",
    patches = ["@com_github_buildbarn_bb_browser//:patches/com_github_tiziano88_elm_protobuf/bytes.diff"],
    sha256 = "c4d499949e807e3dc96eaf91335be61e579a1c63f6a400139aba9d46292dc902",
    strip_prefix = "elm-protobuf-7269bbd2da4740cf9dc85f307e1770050b29411b",
    urls = ["https://github.com/tiziano88/elm-protobuf/archive/7269bbd2da4740cf9dc85f307e1770050b29411b.tar.gz"],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

load("//:go_dependencies.bzl", "bb_browser_go_dependencies")

bb_browser_go_dependencies()

load("@com_github_buildbarn_bb_storage//:go_dependencies.bzl", "bb_storage_go_dependencies")

bb_storage_go_dependencies()

http_archive(
    name = "googleapis",
    build_file = "BUILD.googleapis",
    sha256 = "7b6ea252f0b8fb5cd722f45feb83e115b689909bbb6a393a873b6cbad4ceae1d",
    strip_prefix = "googleapis-143084a2624b6591ee1f9d23e7f5241856642f4d",
    urls = ["https://github.com/googleapis/googleapis/archive/143084a2624b6591ee1f9d23e7f5241856642f4d.zip"],
)

load("@build_bazel_rules_nodejs//:defs.bzl", "node_repositories", "yarn_install")

node_repositories()

yarn_install(
    name = "npm",
    package_json = "//:package.json",
    yarn_lock = "//:yarn.lock",
)
