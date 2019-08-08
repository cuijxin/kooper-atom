package controller

import (
	"log"
	"sync"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Span tag and log keys.
const (
	kubernetesObjectKeyKey         = "kubernetes.object.key"
	kubernetesObjectNSKey          = "kubernetes.object.namespace"
	kubernetesObjectNameKey        = "kubernetes.object.name"
	eventKey                       = "event"
	kooperControllerKey            = "kooper.controller"
	processedTimesKey              = "kubernetes.object.total_processed_times"
	retriesRemainingKey            = "kubernetes.object.retries_remaining"
	processingRetryKey             = "kubernetes.object.processing_retry"
	retriesExecutedKey             = "kubernetes.object.retries_consumed"
	controllerNameKey              = "controller.cfg.name"
	controllerResyncKey            = "controller.cfg.resync_interval"
	controllerMaxRetriesKey        = "controller.cfg.max_retries"
	controllerConcurrentWorkersKey = "controller.cfg.concurrent_workers"
	successKey                     = "success"
	messageKey                     = "message"
)

// generic controller is a controller that can be used to create different kind of controllers.
type generic struct {
	queue     workqueue.RateLimitingInterface // queue will have the jobs that the controller will get and send to handlers.
	informer  cache.SharedIndexInformer       // informer will notify be inform us about resource changes.
	handler   handler.Handler                 // handler is where the logic of resource processing.
	running   bool
	runningMu sync.Mutex
	cfg       Config
	tracer    opentracing.Tracer // use directly opentracing API because it's not an implementation.
	metrics   metrics.Recorder
	leRunner  leaderelection.Runner
	logger    log.Logger
}
