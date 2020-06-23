package myfilter

import (
	"context"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	"k8s.io/kubernetes/pkg/scheduler/nodeinfo"
)


const PluginName = "myfilter-plugin"




// Plugin is an example of a simple plugin that has no state
// and implements only one hook for prebind.
type FilterPlugin struct {
	handle framework.FrameworkHandle

}

var _ framework.FilterPlugin = (*FilterPlugin)(nil)


// https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/v1alpha1/interface.go#L208
func (p FilterPlugin) Name() string {
	return PluginName
}



//  https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/v1alpha1/interface.go#L208
// https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/v1alpha1/interface.go#L259
func (p *FilterPlugin) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, node *nodeinfo.NodeInfo) *framework.Status {
	if pod == nil {
		return framework.NewStatus(framework.Error, "no pod specified")
	}
	if node.Node() == nil {
		return framework.NewStatus(framework.Error, "no node specified")
	}
	log.WithFields(log.Fields{"pod": pod.Name, "node": node.Node().Name}).Debug("filtering a pod against node")


	if e := filterHandler( pod, node.Node(),  p.handle ) ; e!=nil {

		return framework.NewStatus(framework.Unschedulable, e.Error())

	}else{

		// framework.NewStatus :  https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/v1alpha1/interface.go#L149
		return framework.NewStatus(framework.Success, "can be scheduled on this node")
	}

}



// framework.FrameworkHandle : FrameworkHandle provides data and some tools that plugins can use
//  https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/framework/v1alpha1/interface.go#L470
func New(config *runtime.Unknown, f framework.FrameworkHandle) (framework.Plugin, error) {

	return &FilterPlugin{
		handle: f,
	}, nil
}


