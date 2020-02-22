# Buildbarn Browser [![Build status](https://github.com/buildbarn/bb-browser/workflows/CI/badge.svg)](https://github.com/buildbarn/bb-browser/actions)

Buildbarn Browser is a simple web service written in Go that can display
objects stored in a Content Addressable Storage (CAS) and an Action
Cache (AC) that are used by the [Remote Execution API](https://github.com/bazelbuild/remote-apis).
The main purpose of this page is to get more detailed insight in how
remote execution works under the hood, especially when builds fail.

## Requirements for integrating Buildbarn Browser

Even though Buildbarn Browser was primarily developed to integrate with
[other Buildbarn components](https://github.com/buildbarn/bb-remote-execution),
it can in principle be used in combination with any service that
implements the Remote Execution API. There are, however, some features
that your build infrastructure may provide to be able to make optimal
use of Buildbarn Browser:

- **Providing automatically generated links pointing to Buildbarn Browser.**

  The Remote Execution API provides no features for ad hoc exploration
  of the CAS and AC. Buildbarn Browser can therefore only show
  information for objects for which the digest is known. It is therefore
  preferable that build services and clients generate links pointing to
  Buildbarn Browser. Buildbarn's workers attach such links to RPC
  responses and log file entries.

- **Storing results for build actions that should not be cached.**

  Results for certain build actions, such as ones that fail, may not be
  stored in the AC. This is unfortunate, as these are typically the most
  interesting ones to inspect. To still provide access to these results,
  Buildbarn's workers write such results into the CAS using
  [a custom message](https://github.com/buildbarn/bb-storage/blob/master/pkg/proto/cas/cas.proto).
  Buildbarn Browser is capable of displaying these action results in
  additition to cached ones stored in the AC.

- **Storing Build Event Streams.**

  Bazel has the ability to log build execution progress and output
  through [the Build Event Protocol](https://docs.bazel.build/versions/master/build-event-protocol.html).
  Logs can either be written to disk or be transmitted to a gRPC
  service. [The Buildbarn Event Service](https://github.com/buildbarn/bb-event-service)
  is a gRPC service that writes these logs directly into the CAS. Upon
  completion, an AC entry is created to permit lookups of streams by
  Bazel invocation ID.

We invite other implementations of the Remote Execution API to implement
such features as well. At the time of writing, the developers of
[BuildGrid](https://gitlab.com/BuildGrid) are also working on adding
some of these features (~~[#157](https://gitlab.com/BuildGrid/buildgrid/issues/157)~~,
[#158](https://gitlab.com/BuildGrid/buildgrid/issues/158)).

## Setting up Buildbarn Browser

Run the following command to build Buildbarn Browser from source, create
container image and push it into the Docker daemon running on the
current system:

```
$ bazel run //cmd/bb_browser:bb_browser_container
...
Tagging ... as bazel/cmd/bb_browser:bb_browser_container
```

This container image can then be launched using Docker as follows:

```
$ cat config/blobstore.conf
content_addressable_storage {
  grpc {
    endpoint: "bb-storage:8980"
  }
}
action_cache {
  grpc {
    endpoint: "bb-storage:8980"
  }
}
$ docker run \
      -p 80:80 \
      -v $(pwd)/config:/config \
      bazel/cmd/bb_browser:bb_browser_container
```

Buildbarn Browser uses the same storage layer as
[Buildbarn Storage](https://github.com/buildbarn/bb-storage) and can thus
access various types of storage backends (S3, Redis, etc.). In the example
above, it's been configured to simply forward storage access requests to
gRPC service `bb-storage:8980`.  Please refer to
[the configuration file's schema](https://github.com/buildbarn/bb-storage/blob/master/pkg/proto/configuration/blobstore/blobstore.proto)
for more information on how storage access may be configured.

Prebuilt container images of Buildbarn Browser may be found on
[Docker Hub](https://hub.docker.com/r/buildbarn/bb-browser). More
examples of how Buildbarn Browser may be deployed can be found in
[the Buildbarn deployments repository](https://github.com/buildbarn/bb-deployments).
