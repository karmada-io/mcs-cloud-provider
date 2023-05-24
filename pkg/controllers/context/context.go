package context

import (
	"context"

	"github.com/karmada-io/karmada/pkg/sharedcli/ratelimiterflag"
	"github.com/karmada-io/karmada/pkg/util/fedinformer/genericmanager"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"

	multiclusterprovider "github.com/karmada-io/multicluster-cloud-provider"
)

// Options defines all the parameters required by our controllers.
type Options struct {
	// Controllers contains all controller names.
	Controllers []string
	// RateLimiterOptions contains the options for rate limiter.
	RateLimiterOptions ratelimiterflag.Options
}

// Context defines the context object for controller.
type Context struct {
	Context           context.Context
	Mgr               controllerruntime.Manager
	CloudProvider     multiclusterprovider.Interface
	Opts              Options
	DynamicClientSet  dynamic.Interface
	InformerManager   genericmanager.SingleClusterInformerManager
	ProviderClassName string
}

// IsControllerEnabled check if a specified controller enabled or not.
func (c Context) IsControllerEnabled(name string, disabledByDefaultControllers sets.Set[string]) bool {
	hasStar := false
	for _, ctrl := range c.Opts.Controllers {
		if ctrl == name {
			return true
		}
		if ctrl == "-"+name {
			return false
		}
		if ctrl == "*" {
			hasStar = true
		}
	}
	// if we get here, there was no explicit choice
	if !hasStar {
		// nothing on by default
		return false
	}

	return !disabledByDefaultControllers.Has(name)
}

// InitFunc is used to launch a particular controller.
// Any error returned will cause the controller process to `Fatal`
// The bool indicates whether the controller was enabled.
type InitFunc func(ctx Context) (enabled bool, err error)

// Initializers is a public map of named controller groups
type Initializers map[string]InitFunc

// ControllerNames returns all known controller names
func (i Initializers) ControllerNames() []string {
	return sets.StringKeySet(i).List()
}

// StartControllers starts a set of controllers with a specified ControllerContext
func (i Initializers) StartControllers(ctx Context, controllersDisabledByDefault sets.Set[string]) error {
	for controllerName, initFn := range i {
		if !ctx.IsControllerEnabled(controllerName, controllersDisabledByDefault) {
			klog.Warningf("%q is disabled", controllerName)
			continue
		}
		klog.V(1).Infof("Starting %q", controllerName)
		started, err := initFn(ctx)
		if err != nil {
			klog.Errorf("Error starting %q", controllerName)
			return err
		}
		if !started {
			klog.Warningf("Skipping %q", controllerName)
			continue
		}
		klog.Infof("Started %q", controllerName)
	}
	return nil
}
