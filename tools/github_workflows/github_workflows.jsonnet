local workflows_template = import 'external/com_github_buildbarn_bb_storage/tools/github_workflows/workflows_template.libsonnet';

workflows_template.getWorkflows(
  ['bb_browser'],
  ['bb_browser:bb_browser'],
)
