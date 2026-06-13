package main

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/helper"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

const (
	Name = "NodePriorityPlugin"
	// 节点标签键
	PriorityLabel = "priority-level"
)

type NodePriority struct {
	handle framework.Handle
}

var _ framework.FilterPlugin = &NodePriority{} // 过滤不满足条件的节点
var _ framework.ScorePlugin = &NodePriority{}  // 节点评分
var _ framework.ScoreExtensions = &NodePriority{}

func (n *NodePriority) Name() string {
	return Name
}

// Filter - 排除没有priority-label的节点（可选操作）
func (n *NodePriority) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, node *framework.NodeInfo) *framework.Status {
	if _, exists := node.Node().Labels[PriorityLabel]; !exists {
		// 返回错误会排除该节点
		return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("node %s missing priority label", node.Node().Name))
	}
	return nil
}

// Score - 核心评分逻辑
func (n *NodePriority) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := n.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.AsStatus(err)
	}

	node := nodeInfo.Node()
	priority, exists := node.Labels[PriorityLabel]
	if !exists {
		return 0, nil // 返回0分（不强制排除）
	}

	// 优先级权重映射
	weights := map[string]int64{
		"high":   100,
		"medium": 50,
		"low":    10,
	}

	score, valid := weights[priority]
	if !valid {
		return 0, nil
	}
	return score, nil
}

// NormalizeScore - 分数归一化（0-100区间）
func (n *NodePriority) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	return helper.DefaultNormalizeScore(framework.MaxNodeScore, false, scores)
}

func (n *NodePriority) ScoreExtensions() framework.ScoreExtensions {
	return n
}

func New(_ context.Context, obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &NodePriority{handle: h}, nil
}

func main() {
	// 框架入口（由scheduler启动）
	c := frameworkruntime.PluginFactoryAdapter(New)
	frameworkruntime.Register(Name, c)
}
