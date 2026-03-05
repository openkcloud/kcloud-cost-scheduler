/*
LeastAllocated Score Plugin

This plugin favors nodes with fewer requested resources, distributing load evenly.
*/
package plugin

import (
	"context"
	"fmt"

	framework "github.com/KETI-Cloud-Platform/kcloud-cost-scheduler/cost-based-scheduler/internal/framework"
	utils "github.com/KETI-Cloud-Platform/kcloud-cost-scheduler/cost-based-scheduler/internal/framework/utils"

	v1 "k8s.io/api/core/v1"
)

const LeastAllocatedName = "LeastAllocated"

type LeastAllocated struct{}

var _ framework.ScorePlugin = &LeastAllocated{}

func NewLeastAllocated() *LeastAllocated {
	return &LeastAllocated{}
}

func (l *LeastAllocated) Name() string {
	return LeastAllocatedName
}

func (l *LeastAllocated) Score(ctx context.Context, pod *v1.Pod, nodeName string) (int64, *utils.Status) {
	return 0, utils.NewStatus(utils.Success, "")
}

func (l *LeastAllocated) ScoreExtensions() framework.ScoreExtensions {
	return l
}

func (l *LeastAllocated) NormalizeScore(ctx context.Context, pod *v1.Pod, scores utils.PluginResult) *utils.Status {

	return utils.NewStatus(utils.Success, "")
}

func (l *LeastAllocated) scoreNode(pod *v1.Pod, nodeInfo *utils.NodeInfo) (int64, error) {
	if nodeInfo == nil || nodeInfo.Node() == nil {
		return 0, fmt.Errorf("node not found")
	}

	allocatable := nodeInfo.Node().Status.Allocatable
	requested := nodeInfo.Requested

	cpuAllocatable := allocatable.Cpu().MilliValue()
	cpuRequested := requested.MilliCPU
	cpuFree := float64(cpuAllocatable-cpuRequested) / float64(cpuAllocatable)

	memAllocatable := allocatable.Memory().Value()
	memRequested := requested.Memory
	memFree := float64(memAllocatable-memRequested) / float64(memAllocatable)

	score := int64((cpuFree + memFree) / 2 * 100)

	return score, nil
}
