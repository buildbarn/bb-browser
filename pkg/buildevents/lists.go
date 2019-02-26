package buildevents

import (
	buildeventstream "github.com/bazelbuild/bazel/src/main/java/com/google/devtools/build/lib/buildeventstream/proto"
)

type fileList []*buildeventstream.File

func (l fileList) Len() int {
	return len(l)
}

func (l fileList) Less(i int, j int) bool {
	return l[i].Name < l[j].Name
}

func (l fileList) Swap(i int, j int) {
	tmp := l[i]
	l[i] = l[j]
	l[j] = tmp
}

type targetConfiguredNodeList []*TargetConfiguredNode

func (l targetConfiguredNodeList) Len() int {
	return len(l)
}

func (l targetConfiguredNodeList) Less(i int, j int) bool {
	iDisplayOrder := l[i].getDisplayOrder()
	jDisplayOrder := l[j].getDisplayOrder()
	if iDisplayOrder < jDisplayOrder {
		return true
	}
	if iDisplayOrder > jDisplayOrder {
		return false
	}
	return l[i].ID.Label < l[j].ID.Label
}

func (l targetConfiguredNodeList) Swap(i int, j int) {
	tmp := l[i]
	l[i] = l[j]
	l[j] = tmp
}
