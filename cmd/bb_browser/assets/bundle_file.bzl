def _bundle_file_impl(ctx):
    out = ctx.actions.declare_file(ctx.attr.name + ".go")
    ctx.actions.run(
        outputs = [out],
        inputs = [ctx.file._bundle_file, ctx.file.src],
        executable = "python",
        arguments = [
            ctx.file._bundle_file.path,
            ctx.file.src.path,
            out.path,
            ctx.attr.package,
            ctx.attr.name,
        ],
    )
    return [DefaultInfo(files = depset([out]))]

bundle_file = rule(
    _bundle_file_impl,
    attrs = {
        "_bundle_file": attr.label(
            allow_single_file = True,
            default = Label("//cmd/bb_browser/assets:bundle_file.py"),
        ),
        "package": attr.string(mandatory = True),
        "src": attr.label(
            mandatory = True,
            allow_single_file = True,
        ),
    },
)
