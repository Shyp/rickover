http_archive(
    name = "io_bazel_rules_go",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.6.0/rules_go-0.6.0.tar.gz",
    sha256 = "ba6feabc94a5d205013e70792accb6cce989169476668fbaf98ea9b342e13b59",
)
load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains", "go_repository")

go_repository(
    name = "com_github_google_go_cmp",
    importpath = "github.com/google/go-cmp",
    urls = ["https://codeload.github.com/google/go-cmp/zip/18107e6c56edb2d51f965f7d68e59404f0daee54"],
    strip_prefix = "go-cmp-18107e6c56edb2d51f965f7d68e59404f0daee54",
    type = "zip",
)

go_repository(
    name = "com_github_mattn_go_colorable",
    importpath = "github.com/mattn/go-colorable",
    urls = ["https://codeload.github.com/mattn/go-colorable/zip/3fa8c76f9daed4067e4a806fb7e4dc86455c6d6a"],
    strip_prefix = "go-colorable-3fa8c76f9daed4067e4a806fb7e4dc86455c6d6a",
    type = "zip",
)

go_repository(
    name = "com_github_mattn_go_isatty",
    importpath = "github.com/mattn/go-isatty",
    urls = ["https://codeload.github.com/mattn/go-isatty/zip/fc9e8d8ef48496124e79ae0df75490096eccf6fe"],
    strip_prefix = "go-isatty-fc9e8d8ef48496124e79ae0df75490096eccf6fe",
    type = "zip",
)

go_repository(
    name = "com_github_go_stack_stack",
    importpath = "github.com/go-stack/stack",
    urls = ["https://codeload.github.com/go-stack/stack/zip/54be5f394ed2c3e19dac9134a40a95ba5a017f7b"],
    strip_prefix = "stack-54be5f394ed2c3e19dac9134a40a95ba5a017f7b",
    type = "zip",
)

go_repository(
    name = "com_github_inconshreveable_log15",
    importpath = "github.com/inconshreveable/log15",
    urls = ["https://codeload.github.com/inconshreveable/log15/zip/74a0988b5f804e8ce9ff74fca4f16980776dff29"],
    strip_prefix = "log15-74a0988b5f804e8ce9ff74fca4f16980776dff29",
    type = "zip",
)

go_rules_dependencies()
go_register_toolchains()
