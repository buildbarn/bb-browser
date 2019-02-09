# Buildbarn Browser

Buildbarn Browser is a simple web service written in Go that can display
objects stored in a Content Addressable Storage (CAS) and an Action
Cache (AC) that are used by the [Remote Execution API](https://github.com/bazelbuild/remote-apis).
The main purpose of this page is to get more detailed insight in how
remote execution works under the hood, especially when builds fail.

## Requirements for integrating Buildbarn Browser

Even though Buildbarn Browser was primarily developed to integrate with
other Buildbarn components, it can in principle be used in combination
with any service that implements the Remote Execution API. There are,
however, two features that your build infrastructure should provide to
be able to make optimal use of Buildbarn Browser:

- **Providing automatically generated links pointing to Buildbarn Browser.**

  The Remote Execution API provides no features for ad hoc exploration
  of the CAS and AC. Buildbarn Browser can therefore only show
  information for objects for which the digest is known. It is therefore
  preferable that build services and clients generate links pointing to
  Bazel Buildbarn. Buildbarn's workers attach such links to RPC
  responses and log file entries.

- **Storing results for build actions that should not be cached.**

  Results for certain build actions, such as ones that fail, may not be
  stored in the AC. This is unfortunate, as these are typically the most
  interesting ones to inspect. To still provide access to these results,
  Buildbarn's workers write such results into the CAS using
  [a custom message](https://github.com/buildbarn/bb-storage/blob/master/pkg/proto/cas/cas.proto).
  Buildbarn Browser is capable of displaying these action results in
  additition to cached ones stored in the AC.

We invite other implementations of the Remote Execution API to implement
such features as well. At the time of writing, the developers of
[BuildGrid](https://gitlab.com/BuildGrid) are also considering adding
these features.

## Setting up Buildbarn Browser

TODO
