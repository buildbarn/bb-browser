package buildevents

import (
	"sort"

	buildeventstream "github.com/bazelbuild/bazel/src/main/java/com/google/devtools/build/lib/buildeventstream/proto"
	"github.com/bazelbuild/bazel/src/main/protobuf"
	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type defaultNode struct {
}

func (n *defaultNode) addBuildFinishedNode(child *BuildFinishedNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addBuildMetricsNode(child *BuildMetricsNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addBuildToolLogsNode(child *BuildToolLogsNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addConfigurationNode(child *ConfigurationNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addExpandedNode(child *ExpandedNode, skipped bool) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addFetchNode(child *FetchNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addNamedSetNode(child *NamedSetNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addOptionsParsedNode(child *OptionsParsedNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addProgressNode(child *ProgressNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addStartedNode(child *StartedNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addStructuredCommandLineNode(child *StructuredCommandLineNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addTargetCompletedNode(child *TargetCompletedNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addTargetConfiguredNode(child *TargetConfiguredNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addTestResultNode(child *TestResultNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addTestSummaryNode(child *TestSummaryNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addUnconfiguredLabelNode(child *UnconfiguredLabelNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addUnstructuredCommandLineNode(child *UnstructuredCommandLineNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

func (n *defaultNode) addWorkspaceStatusNode(child *WorkspaceStatusNode) error {
	return status.Error(codes.InvalidArgument, "Value cannot be placed at this location")
}

type rootNode struct {
	defaultNode

	Started *StartedNode
}

func (n *rootNode) addStartedNode(child *StartedNode) error {
	if n.Started != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.Started = child
	return nil
}

// BuildFinishedNode corresponds to a Build Event Protocol message with
// BuildEventID kind `build_finished` and BuildEvent payload kind
// `finished`.
type BuildFinishedNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_BuildFinishedId
	Payload *buildeventstream.BuildFinished

	BuildMetrics  *BuildMetricsNode
	BuildToolLogs *BuildToolLogsNode
}

func (n *BuildFinishedNode) addBuildMetricsNode(child *BuildMetricsNode) error {
	if n.BuildMetrics != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.BuildMetrics = child
	return nil
}

func (n *BuildFinishedNode) addBuildToolLogsNode(buildToolLogs *BuildToolLogsNode) error {
	if n.BuildToolLogs != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.BuildToolLogs = buildToolLogs
	return nil
}

// BuildMetricsNode corresponds to a Build Event Protocol message with
// BuildEventID kind `build_metrics` and BuildEvent payload kind
// `build_metrics`.
type BuildMetricsNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_BuildMetricsId
	Payload *buildeventstream.BuildMetrics
}

// BuildToolLogsNode corresponds to a Build Event Protocol message with
// BuildEventID kind `build_tool_logs` and BuildEvent payload kind
// `build_tool_logs`.
type BuildToolLogsNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_BuildToolLogsId
	Payload *buildeventstream.BuildToolLogs
}

// ConfigurationNode corresponds to a Build Event Protocol message with
// BuildEventID kind `configuration` and BuildEvent payload kind
// `configuration`.
type ConfigurationNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_ConfigurationId
	Payload *buildeventstream.Configuration
}

// ExpandedNode corresponds to a Build Event Protocol message with
// BuildEventID kind `pattern` or `pattern_skipped` and BuildEvent
// payload kind `expanded`.
type ExpandedNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_PatternExpandedId
	Payload *buildeventstream.PatternExpanded

	Configuration     *ConfigurationNode
	TargetsConfigured []*TargetConfiguredNode
}

func (n *ExpandedNode) addConfigurationNode(configuration *ConfigurationNode) error {
	if n.Configuration != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.Configuration = configuration
	return nil
}

func (n *ExpandedNode) addTargetConfiguredNode(child *TargetConfiguredNode) error {
	n.TargetsConfigured = append(n.TargetsConfigured, child)
	return nil
}

// FetchNode corresponds to a Build Event Protocol message with
// BuildEventID kind `fetch` and BuildEvent payload kind `fetch`.
type FetchNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_FetchId
	Payload *buildeventstream.Fetch
}

// NamedSetNode corresponds to a Build Event Protocol message with
// BuildEventID kind `named_set` and BuildEvent payload kind
// `named_set_of_files`.
type NamedSetNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_NamedSetOfFilesId
	Payload *buildeventstream.NamedSetOfFiles
}

// OptionsParsedNode corresponds to a Build Event Protocol message with
// BuildEventID kind `options_parsed` and BuildEvent payload kind
// `options_parsed`.
type OptionsParsedNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_OptionsParsedId
	Payload *buildeventstream.OptionsParsed
}

// ProgressNode corresponds to a Build Event Protocol message with
// BuildEventID kind `progress` and BuildEvent payload kind `progress`.
type ProgressNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_ProgressId
	Payload *buildeventstream.Progress

	BuildMetrics  *BuildMetricsNode
	BuildToolLogs *BuildToolLogsNode
	Configuration *ConfigurationNode
	Fetches       []*FetchNode
	NamedSets     []*NamedSetNode
	Progress      *ProgressNode
}

func (n *ProgressNode) addBuildMetricsNode(child *BuildMetricsNode) error {
	if n.BuildMetrics != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.BuildMetrics = child
	return nil
}

func (n *ProgressNode) addBuildToolLogsNode(child *BuildToolLogsNode) error {
	if n.BuildToolLogs != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.BuildToolLogs = child
	return nil
}

func (n *ProgressNode) addConfigurationNode(configuration *ConfigurationNode) error {
	if n.Configuration != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.Configuration = configuration
	return nil
}

func (n *ProgressNode) addFetchNode(child *FetchNode) error {
	n.Fetches = append(n.Fetches, child)
	return nil
}

func (n *ProgressNode) addNamedSetNode(child *NamedSetNode) error {
	n.NamedSets = append(n.NamedSets, child)
	return nil
}

func (n *ProgressNode) addProgressNode(child *ProgressNode) error {
	if n.Progress != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.Progress = child
	return nil
}

// StartedNode corresponds to a Build Event Protocol message with
// BuildEventID kind `started` and BuildEvent payload kind `started`. It
// is always the root of the tree.
type StartedNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_BuildStartedId
	Payload *buildeventstream.BuildStarted

	BuildFinished            *BuildFinishedNode
	OptionsParsed            *OptionsParsedNode
	Patterns                 []*ExpandedNode
	PatternsSkipped          []*ExpandedNode
	Progress                 *ProgressNode
	StructuredCommandLines   []*StructuredCommandLineNode
	UnstructuredCommandLines []*UnstructuredCommandLineNode
	WorkspaceStatuses        []*WorkspaceStatusNode

	namedSets map[string]*buildeventstream.NamedSetOfFiles
}

func (n *StartedNode) addBuildFinishedNode(child *BuildFinishedNode) error {
	if n.BuildFinished != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.BuildFinished = child
	return nil
}

func (n *StartedNode) addOptionsParsedNode(child *OptionsParsedNode) error {
	if n.OptionsParsed != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.OptionsParsed = child
	return nil
}

func (n *StartedNode) addExpandedNode(child *ExpandedNode, skipped bool) error {
	if skipped {
		n.PatternsSkipped = append(n.PatternsSkipped, child)
	} else {
		n.Patterns = append(n.Patterns, child)
	}
	return nil
}

func (n *StartedNode) addProgressNode(child *ProgressNode) error {
	if n.Progress != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.Progress = child
	return nil
}

func (n *StartedNode) addStructuredCommandLineNode(child *StructuredCommandLineNode) error {
	n.StructuredCommandLines = append(n.StructuredCommandLines, child)
	return nil
}

func (n *StartedNode) addUnstructuredCommandLineNode(child *UnstructuredCommandLineNode) error {
	n.UnstructuredCommandLines = append(n.UnstructuredCommandLines, child)
	return nil
}

func (n *StartedNode) addWorkspaceStatusNode(child *WorkspaceStatusNode) error {
	n.WorkspaceStatuses = append(n.WorkspaceStatuses, child)
	return nil
}

// GetFilesForNamedSets obtains the transitive closure of the set of
// files stored in named sets. This function can be used to get a full
// list of files stored in an OutputGroup message (i.e., the list of
// output files for a build action).
//
// Expansion is performed in such a way that termination is guaranteed,
// even if named sets are cyclic.
func (n *StartedNode) GetFilesForNamedSets(namedSets []*buildeventstream.BuildEventId_NamedSetOfFilesId) []*buildeventstream.File {
	todo := [][]*buildeventstream.BuildEventId_NamedSetOfFilesId{namedSets}
	done := map[string]bool{}
	var files fileList
	for len(todo) > 0 {
		namedSetsList := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		for _, childID := range namedSetsList {
			key := proto.MarshalTextString(childID)
			if namedSet, ok := n.namedSets[key]; ok && !done[key] {
				files = append(files, namedSet.Files...)
				todo = append(todo, namedSet.FileSets)
				done[key] = true
			}
		}
	}
	sort.Sort(files)
	return files
}

// StructuredCommandLineNode corresponds to a Build Event Protocol
// message with BuildEventID kind `structured_command_line` and
// BuildEvent payload kind `structured_command_line`.
type StructuredCommandLineNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_StructuredCommandLineId
	Payload *protobuf.CommandLine
}

// TargetCompletedNode corresponds to a Build Event Protocol message
// with BuildEventID kind `target_completed` and BuildEvent payload kind
// `aborted` or `completed`.
type TargetCompletedNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_TargetCompletedId
	Success *TargetCompletedSuccess
	Aborted *TargetCompletedAborted
}

func (n *TargetCompletedNode) addTestResultNode(child *TestResultNode) error {
	if n.Success == nil {
		return status.Error(codes.InvalidArgument, "Cannot set value on target that did not complete successfully")
	}
	n.Success.TestResults = append(n.Success.TestResults, child)
	return nil
}

func (n *TargetCompletedNode) addTestSummaryNode(child *TestSummaryNode) error {
	if n.Success == nil {
		return status.Error(codes.InvalidArgument, "Cannot set value on target that did not complete successfully")
	}
	if n.Success.TestSummary != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.Success.TestSummary = child
	return nil
}

func (n *TargetCompletedNode) addUnconfiguredLabelNode(child *UnconfiguredLabelNode) error {
	if n.Aborted == nil {
		return status.Error(codes.InvalidArgument, "Cannot set value on target that completed successfully")
	}
	n.Aborted.UnconfiguredLabels = append(n.Aborted.UnconfiguredLabels, child)
	return nil
}

type TargetCompletedSuccess struct {
	Payload *buildeventstream.TargetComplete

	TestResults []*TestResultNode
	TestSummary *TestSummaryNode
}

type TargetCompletedAborted struct {
	Payload *buildeventstream.Aborted

	UnconfiguredLabels []*UnconfiguredLabelNode
}

// IsSuccess returns whether the target completed successfully. In more
// practical terms, whether it should be displayed as green or red.
func (n *TargetCompletedNode) IsSuccess() bool {
	return n.Success != nil && n.Success.Payload.Success && (n.Success.TestSummary == nil || n.Success.TestSummary.Payload.OverallStatus == buildeventstream.TestStatus_PASSED)
}

// TargetConfiguredNode corresponds to a Build Event Protocol message
// with BuildEventID kind `target_configured` and BuildEvent payload
// kind `aborted` or `configured`.
type TargetConfiguredNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_TargetConfiguredId
	Success *TargetConfiguredSuccess
	Aborted *TargetConfiguredAborted
}

func (n *TargetConfiguredNode) addTargetCompletedNode(child *TargetCompletedNode) error {
	if n.Success == nil {
		return status.Error(codes.InvalidArgument, "Cannot set value on target that did not configure successfully")
	}
	if n.Success.TargetCompleted != nil {
		return status.Error(codes.InvalidArgument, "Value already set")
	}
	n.Success.TargetCompleted = child
	return nil
}

func (n *TargetConfiguredNode) getDisplayOrder() int {
	if n.Success == nil || n.Success.TargetCompleted == nil {
		return 2
	}
	if n.Success.TargetCompleted.IsSuccess() {
		return 1
	}
	return 0
}

type TargetConfiguredSuccess struct {
	Payload *buildeventstream.TargetConfigured

	TargetCompleted *TargetCompletedNode
}

type TargetConfiguredAborted struct {
	Payload *buildeventstream.Aborted
}

// TestResultNode corresponds to a Build Event Protocol message with
// BuildEventID kind `test_result` and BuildEvent payload kind
// `aborted` or `test_result`.
type TestResultNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_TestResultId
	Success *TestResultSuccess
	Aborted *TestResultAborted
}

type TestResultSuccess struct {
	Payload *buildeventstream.TestResult
}

type TestResultAborted struct {
	Payload *buildeventstream.Aborted
}

// TestSummaryNode corresponds to a Build Event Protocol message with
// BuildEventID kind `test_summary` and BuildEvent payload kind
// `test_summary`.
type TestSummaryNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_TestSummaryId
	Payload *buildeventstream.TestSummary
}

// UnconfiguredLabelNode corresponds to a Build Event Protocol message
// with BuildEventID kind `unconfigured_label` and BuildEvent payload
// kind `aborted`.
type UnconfiguredLabelNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_UnconfiguredLabelId
	Payload *buildeventstream.Aborted
}

// UnstructuredCommandLineNode corresponds to a Build Event Protocol
// message with BuildEventID kind `unstructured_command_line` and
// BuildEvent payload kind `unstructured_command_line`.
type UnstructuredCommandLineNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_UnstructuredCommandLineId
	Payload *buildeventstream.UnstructuredCommandLine
}

// WorkspaceStatusNode corresponds to a Build Event Protocol message
// with BuildEventID kind `workspace_status` and BuildEvent payload kind
// `workspace_status`.
type WorkspaceStatusNode struct {
	defaultNode

	ID      *buildeventstream.BuildEventId_WorkspaceStatusId
	Payload *buildeventstream.WorkspaceStatus
}
