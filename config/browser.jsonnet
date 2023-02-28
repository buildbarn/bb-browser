// This example configuration is based on the output from running
// `jsonnet bb-deployments.git/docker-compose/config/browser.jsonnet`.
// For documentation, see pkg/proto/configuration/bb_browser/bb_browser.proto.
{
  blobstore: {
    actionCache: {
      grpc: {
        address: 'bb-storage:8980',
      },
    },
    contentAddressableStorage: {
      grpc: {
        address: 'bb-storage:8980',
      },
    },
  },
  maximumMessageSizeBytes: 16777216,
  listenAddress: ':80',
  authorizer: {
    allow: {},
  },
}
