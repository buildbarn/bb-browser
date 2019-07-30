package buildevents

import (
	"sort"

	buildeventstream "github.com/bazelbuild/bazel/src/main/java/com/google/devtools/build/lib/buildeventstream/proto"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type node interface {
	addActionCompletedNode(child *ActionCompletedNode) error
	addBuildFinishedNode(child *BuildFinishedNode) error
	addBuildMetricsNode(child *BuildMetricsNode) error
	addBuildToolLogsNode(child *BuildToolLogsNode) error
	addConfigurationNode(child *ConfigurationNode) error
	addConfiguredLabelNode(child *ConfiguredLabelNode) error
	addFetchNode(child *FetchNode) error
	addNamedSetNode(child *NamedSetNode) error
	addOptionsParsedNode(child *OptionsParsedNode) error
	addPatternNode(child *PatternNode) error
	addProgressNode(child *ProgressNode) error
	addStartedNode(child *StartedNode) error
	addStructuredCommandLineNode(child *StructuredCommandLineNode) error
	addTargetCompletedNode(child *TargetCompletedNode) error
	addTargetConfiguredNode(child *TargetConfiguredNode) error
	addTestResultNode(child *TestResultNode) error
	addTestSummaryNode(child *TestSummaryNode) error
	addUnconfiguredLabelNode(child *UnconfiguredLabelNode) error
	addUnstructuredCommandLineNode(child *UnstructuredCommandLineNode) error
	addWorkspaceConfigNode(child *WorkspaceConfigNode) error
	addWorkspaceStatusNode(child *WorkspaceStatusNode) error
}

// StreamParser recombines all BuildEvent messages that are stored in a
// Build Event Stream into a tree, based on parent-child relationships
// specified in the messages.
//
// The generated tree is strongly typed, in the sense that it can only
// encode parent-child relationships that are actually emitted by Bazel
// in practice. This makes the resulting tree suitable for
// analysis/displaying purposes.
type StreamParser struct {
	root    rootNode
	parents map[string][]node

	delayedActionsCompleted []*ActionCompletedNode
	delayedPatterns         []*PatternNode
}

// NewStreamParser constructs a StreamParser that is in the initial
// state. It contains an empty tree of build events; no StartedNode is
// present yet. It expects that the first event to be published has a
// BuildEventId of kind `started`.
func NewStreamParser() *StreamParser {
	p := &StreamParser{
		parents: map[string][]node{},
	}
	p.parents[proto.MarshalTextString(&buildeventstream.BuildEventId{
		Id: &buildeventstream.BuildEventId_Started{
			Started: &buildeventstream.BuildEventId_BuildStartedId{},
		},
	})] = []node{&p.root}
	return p
}

func (p *StreamParser) getSingleParent(key string) (node, error) {
	parents, ok := p.parents[key]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Event with ID %#v was not expected", key)
	}
	if len(parents) != 1 {
		return nil, status.Errorf(codes.InvalidArgument, "Event with ID %#v cannot have multiple parents", key)
	}
	delete(p.parents, key)
	return parents[0], nil
}

func (p *StreamParser) getAllParents(key string) ([]node, error) {
	parents, ok := p.parents[key]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Event with ID %#v was not expected", key)
	}
	delete(p.parents, key)
	return parents, nil
}

// AddBuildEvent adds a single node to the build event tree. It is added
// as a leaf to the node that previously announced its existence.
// Insertion may fail due to the message not being announced by any
// parent, bad typing or invalid cardinality.
func (p *StreamParser) AddBuildEvent(event *buildeventstream.BuildEvent) error {
	if event.Id == nil {
		return status.Error(codes.InvalidArgument, "Received build event with nil identifier")
	}

	key := proto.MarshalTextString(event.Id)

	var newChild node
	switch id := event.Id.Id.(type) {
	case *buildeventstream.BuildEventId_ActionCompleted:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_Action)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"ActionCompleted\" build event has an incorrect payload type")
		}

		n := &ActionCompletedNode{
			ID:      id.ActionCompleted,
			Payload: payload.Action,
		}
		p.delayedActionsCompleted = append(p.delayedActionsCompleted, n)
		newChild = n

	case *buildeventstream.BuildEventId_BuildFinished:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_Finished)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"BuildFinished\" build event has an incorrect payload type")
		}

		n := &BuildFinishedNode{
			ID:      id.BuildFinished,
			Payload: payload.Finished,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addBuildFinishedNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"BuildFinished\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_BuildMetrics:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_BuildMetrics)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"BuildMetrics\" build event has an incorrect payload type")
		}

		n := &BuildMetricsNode{
			ID:      id.BuildMetrics,
			Payload: payload.BuildMetrics,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addBuildMetricsNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"BuildMetrics\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_BuildToolLogs:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_BuildToolLogs)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"BuildToolLogs\" build event has an incorrect payload type")
		}

		n := &BuildToolLogsNode{
			ID:      id.BuildToolLogs,
			Payload: payload.BuildToolLogs,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addBuildToolLogsNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"BuildToolLogs\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_Configuration:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_Configuration)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"Configuration\" build event has an incorrect payload type")
		}

		n := &ConfigurationNode{
			ID:      id.Configuration,
			Payload: payload.Configuration,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addConfigurationNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"Configuration\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_ConfiguredLabel:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_Aborted)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"ConfiguredLabel\" build event has an incorrect payload type")
		}

		n := &ConfiguredLabelNode{
			ID:      id.ConfiguredLabel,
			Payload: payload.Aborted,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addConfiguredLabelNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"ConfiguredLabel\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_Fetch:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_Fetch)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"Fetch\" build event has an incorrect payload type")
		}

		n := &FetchNode{
			ID:      id.Fetch,
			Payload: payload.Fetch,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addFetchNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"Fetch\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_NamedSet:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_NamedSetOfFiles)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"NamedSet\" build event has an incorrect payload type")
		}

		n := &NamedSetNode{
			ID:      id.NamedSet,
			Payload: payload.NamedSetOfFiles,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addNamedSetNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"NamedSet\" node with ID %#v", key)
		}
		p.root.Started.namedSets[proto.MarshalTextString(n.ID)] = n.Payload
		newChild = n

	case *buildeventstream.BuildEventId_OptionsParsed:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_OptionsParsed)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"OptionsParsed\" build event has an incorrect payload type")
		}

		n := &OptionsParsedNode{
			ID:      id.OptionsParsed,
			Payload: payload.OptionsParsed,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addOptionsParsedNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"OptionsParsed\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_Pattern:
		var n *PatternNode
		switch payload := event.Payload.(type) {
		case *buildeventstream.BuildEvent_Aborted:
			n = &PatternNode{
				ID: id.Pattern,
				Aborted: &PatternAborted{
					aborted: aborted{
						Payload: payload.Aborted,
					},
				},
			}
		case *buildeventstream.BuildEvent_Expanded:
			n = &PatternNode{
				ID: id.Pattern,
				Success: &PatternSuccess{
					Payload: payload.Expanded,
				},
			}
		default:
			return status.Error(codes.InvalidArgument, "\"Pattern\" build event has an incorrect payload type")
		}
		p.delayedPatterns = append(p.delayedPatterns, n)
		newChild = n

	case *buildeventstream.BuildEventId_Progress:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_Progress)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"Progress\" build event has an incorrect payload type")
		}

		n := &ProgressNode{
			ID:      id.Progress,
			Payload: payload.Progress,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addProgressNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"Progress\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_Started:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_Started)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"Started\" build event has an incorrect payload type")
		}

		n := &StartedNode{
			ID:        id.Started,
			Payload:   payload.Started,
			namedSets: map[string]*buildeventstream.NamedSetOfFiles{},
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addStartedNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"Started\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_StructuredCommandLine:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_StructuredCommandLine)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"StructuredCommandLine\" build event has an incorrect payload type")
		}

		n := &StructuredCommandLineNode{
			ID:      id.StructuredCommandLine,
			Payload: payload.StructuredCommandLine,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addStructuredCommandLineNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"StructuredCommandLine\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_TargetCompleted:
		var n *TargetCompletedNode
		switch payload := event.Payload.(type) {
		case *buildeventstream.BuildEvent_Aborted:
			n = &TargetCompletedNode{
				ID: id.TargetCompleted,
				Aborted: &TargetCompletedAborted{
					aborted: aborted{
						Payload: payload.Aborted,
					},
				},
			}
			newChild = n
		case *buildeventstream.BuildEvent_Completed:
			n = &TargetCompletedNode{
				ID: id.TargetCompleted,
				Success: &TargetCompletedSuccess{
					Payload: payload.Completed,
				},
			}
		default:
			return status.Error(codes.InvalidArgument, "\"TargetCompleted\" build event has an incorrect payload type")
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addTargetCompletedNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"TargetCompleted\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_TargetConfigured:
		var n *TargetConfiguredNode
		switch payload := event.Payload.(type) {
		case *buildeventstream.BuildEvent_Aborted:
			n = &TargetConfiguredNode{
				ID: id.TargetConfigured,
				Aborted: &TargetConfiguredAborted{
					aborted: aborted{
						Payload: payload.Aborted,
					},
				},
			}
		case *buildeventstream.BuildEvent_Configured:
			n = &TargetConfiguredNode{
				ID: id.TargetConfigured,
				Success: &TargetConfiguredSuccess{
					Payload: payload.Configured,
				},
			}
		default:
			return status.Error(codes.InvalidArgument, "\"TargetConfigured\" build event has an incorrect payload type")
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addTargetConfiguredNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"TargetConfigured\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_TestResult:
		var n *TestResultNode
		switch payload := event.Payload.(type) {
		case *buildeventstream.BuildEvent_Aborted:
			n = &TestResultNode{
				ID: id.TestResult,
				Aborted: &TestResultAborted{
					aborted: aborted{
						Payload: payload.Aborted,
					},
				},
			}
		case *buildeventstream.BuildEvent_TestResult:
			n = &TestResultNode{
				ID: id.TestResult,
				Success: &TestResultSuccess{
					Payload: payload.TestResult,
				},
			}
		default:
			return status.Error(codes.InvalidArgument, "\"TestResult\" build event has an incorrect payload type")
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addTestResultNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"TestResult\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_TestSummary:
		var n *TestSummaryNode
		switch payload := event.Payload.(type) {
		case *buildeventstream.BuildEvent_Aborted:
			n = &TestSummaryNode{
				ID: id.TestSummary,
				Aborted: &TestSummaryAborted{
					aborted: aborted{
						Payload: payload.Aborted,
					},
				},
			}
		case *buildeventstream.BuildEvent_TestSummary:
			n = &TestSummaryNode{
				ID: id.TestSummary,
				Success: &TestSummarySuccess{
					Payload: payload.TestSummary,
				},
			}
		default:
			return status.Error(codes.InvalidArgument, "\"TestSummary\" build event has an incorrect payload type")
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addTestSummaryNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"TestSummary\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_UnconfiguredLabel:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_Aborted)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"UnconfiguredLabel\" build event has an incorrect payload type")
		}

		n := &UnconfiguredLabelNode{
			ID:      id.UnconfiguredLabel,
			Payload: payload.Aborted,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addUnconfiguredLabelNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"UnconfiguredLabel\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_UnstructuredCommandLine:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_UnstructuredCommandLine)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"UnstructuredCommandLine\" build event has an incorrect payload type")
		}

		n := &UnstructuredCommandLineNode{
			ID:      id.UnstructuredCommandLine,
			Payload: payload.UnstructuredCommandLine,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addUnstructuredCommandLineNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"UnstructuredCommandLine\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_WorkspaceStatus:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_WorkspaceStatus)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"WorkspaceStatus\" build event has an incorrect payload type")
		}

		n := &WorkspaceStatusNode{
			ID:      id.WorkspaceStatus,
			Payload: payload.WorkspaceStatus,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addWorkspaceStatusNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"WorkspaceStatus\" node with ID %#v", key)
		}
		newChild = n

	case *buildeventstream.BuildEventId_Workspace:
		payload, ok := event.Payload.(*buildeventstream.BuildEvent_WorkspaceInfo)
		if !ok {
			return status.Error(codes.InvalidArgument, "\"WorkspaceConfig\" build event has an incorrect payload type")
		}

		n := &WorkspaceConfigNode{
			ID: id.Workspace,
			Payload: payload.WorkspaceInfo,
		}
		parent, err := p.getSingleParent(key)
		if err != nil {
			return err
		}
		if err := parent.addWorkspaceConfigNode(n); err != nil {
			return util.StatusWrapf(err, "Cannot add \"WorkspaceConfig\" node with ID %#v", key)
		}
		newChild = n

	default:
		return status.Error(codes.InvalidArgument, "Received unknown build event")
	}

	for _, child := range event.Children {
		key := proto.MarshalTextString(child)
		p.parents[key] = append(p.parents[key], newChild)
	}

	return nil
}

// Finalize returns the root of the resulting build event tree. It may
// return nil in case not a single build event was inserted. It also
// returns how many build events are at least expected to be inserted.
// When zero, the tree is complete.
func (p *StreamParser) Finalize() (*StartedNode, int, error) {
	// TODO(edsch): Even though build events normally form a tree,
	// there are some violations:
	//
	// - ActionCompletedNodes has both a ProgressNode and a
	//   TargetCompletedNode as its parents.
	// - PatternNodes have both a ProgressNode and a PatternNode as
	//   its parents.
	//
	// The events corresponding with these nodes are emitted in an
	// order that violates topological sorting. Work around this by
	// saving up these nodes and inserting them into the tree at the
	// very end. We should see if we could at least alter Bazel to
	// respect topological sorting to remove the need for this
	// special treatment.
	//
	// Cycles in the graph are prevented in the latter case by only
	// allowing aborted PatternNodes as children of successful
	// PatternNodes.
	//
	// More details: https://github.com/bazelbuild/bazel/issues/7608
	for _, n := range p.delayedActionsCompleted {
		key := proto.MarshalTextString(&buildeventstream.BuildEventId{
			Id: &buildeventstream.BuildEventId_ActionCompleted{
				ActionCompleted: n.ID,
			},
		})
		parents, err := p.getAllParents(key)
		if err != nil {
			return nil, 0, err
		}
		for _, parent := range parents {
			if err := parent.addActionCompletedNode(n); err != nil {
				return nil, 0, util.StatusWrapf(err, "Cannot add \"ActionCompleted\" node with ID %#v", key)
			}
		}
	}
	for _, n := range p.delayedPatterns {
		key := proto.MarshalTextString(&buildeventstream.BuildEventId{
			Id: &buildeventstream.BuildEventId_Pattern{
				Pattern: n.ID,
			},
		})
		parents, err := p.getAllParents(key)
		if err != nil {
			return nil, 0, err
		}
		for _, parent := range parents {
			if err := parent.addPatternNode(n); err != nil {
				return nil, 0, util.StatusWrapf(err, "Cannot add \"Pattern\" node with ID %#v", key)
			}
		}
	}

	if started := p.root.Started; started != nil {
		for _, patternNode := range started.Patterns {
			if success := patternNode.Success; success != nil {
				sort.Sort(targetConfiguredNodeList(success.TargetsConfigured))
			}
		}
	}

	return p.root.Started, len(p.parents), nil
}
