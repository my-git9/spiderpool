// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/google/gops/agent"
	"github.com/pyroscope-io/client/pyroscope"

	"github.com/spidernet-io/spiderpool/pkg/ipam"
	"github.com/spidernet-io/spiderpool/pkg/ippoolmanager"
	"github.com/spidernet-io/spiderpool/pkg/limiter"
	"github.com/spidernet-io/spiderpool/pkg/logutils"
	"github.com/spidernet-io/spiderpool/pkg/namespacemanager"
	"github.com/spidernet-io/spiderpool/pkg/nodemanager"
	"github.com/spidernet-io/spiderpool/pkg/podmanager"
	"github.com/spidernet-io/spiderpool/pkg/reservedipmanager"
	"github.com/spidernet-io/spiderpool/pkg/singletons"
	"github.com/spidernet-io/spiderpool/pkg/statefulsetmanager"
	"github.com/spidernet-io/spiderpool/pkg/subnetmanager"
	"github.com/spidernet-io/spiderpool/pkg/workloadendpointmanager"
)

// DaemonMain runs agentContext handlers.
func DaemonMain() {
	// reinitialize the logger
	v := logutils.ConvertLogLevel(agentContext.Cfg.LogLevel)
	if v == nil {
		panic(fmt.Sprintf("unknown log level %s \n", agentContext.Cfg.LogLevel))
	}
	err := logutils.InitStdoutLogger(*v)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger with level %s, reason=%v \n", agentContext.Cfg.LogLevel, err))
	}
	logger = logutils.Logger.Named(BinNameAgent)

	currentP := runtime.GOMAXPROCS(-1)
	logger.Sugar().Infof("default max golang procs %v \n", currentP)
	if currentP > int(agentContext.Cfg.GoMaxProcs) {
		runtime.GOMAXPROCS(int(agentContext.Cfg.GoMaxProcs))
		currentP = runtime.GOMAXPROCS(-1)
		logger.Sugar().Infof("change max golang procs %v \n", currentP)
	}

	if len(agentContext.Cfg.CommitVersion) > 0 {
		logger.Sugar().Infof("CommitVersion: %v \n", agentContext.Cfg.CommitVersion)
	}
	if len(agentContext.Cfg.CommitTime) > 0 {
		logger.Sugar().Infof("CommitTime: %v \n", agentContext.Cfg.CommitTime)
	}
	if len(agentContext.Cfg.AppVersion) > 0 {
		logger.Sugar().Infof("AppVersion: %v \n", agentContext.Cfg.AppVersion)
	}

	if err := agentContext.LoadConfigmap(); err != nil {
		logger.Sugar().Fatal("failed to load Configmap: %v", err)
	}
	logger.Sugar().Infof("Spiderpool-agent config: %+v", agentContext.Cfg)

	if agentContext.Cfg.GopsListenPort != "" {
		address := "127.0.0.1:" + agentContext.Cfg.GopsListenPort
		op := agent.Options{
			ShutdownCleanup: true,
			Addr:            address,
		}
		if err := agent.Listen(op); err != nil {
			logger.Sugar().Fatalf("gops failed to listen on port %s, reason=%v", address, err)
		}
		logger.Sugar().Infof("gops is listening on %s ", address)
		defer agent.Close()
	}

	if agentContext.Cfg.PyroscopeAddress != "" {
		// push mode ,  push to pyroscope server
		logger.Sugar().Infof("pyroscope works in push mode, server %s ", agentContext.Cfg.PyroscopeAddress)
		node, e := os.Hostname()
		if e != nil || len(node) == 0 {
			logger.Sugar().Fatalf("failed to get hostname, reason=%v", e)
		}
		_, e = pyroscope.Start(pyroscope.Config{
			ApplicationName: BinNameAgent,
			ServerAddress:   agentContext.Cfg.PyroscopeAddress,
			Logger:          nil,
			Tags:            map[string]string{"node": node},
			ProfileTypes: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,
			},
		})
		if e != nil {
			logger.Sugar().Fatalf("failed to setup pyroscope, reason=%v", e)
		}
	}

	logger.Info("Begin to initialize spiderpool-agent metrics HTTP server")
	initAgentMetricsServer(context.TODO())

	logger.Sugar().Infof("Begin to initialize cluster default pool configuration")
	singletons.InitClusterDefaultPool(
		agentContext.Cfg.ClusterDefaultIPv4IPPool,
		agentContext.Cfg.ClusterDefaultIPv6IPPool,
		agentContext.Cfg.ClusterDefaultIPv4Subnet,
		agentContext.Cfg.ClusterDefaultIPv6Subnet,
		agentContext.Cfg.ClusterSubnetDefaultFlexibleIPNum,
	)

	agentContext.InnerCtx, agentContext.InnerCancel = context.WithCancel(context.Background())
	logger.Info("Begin to initialize spiderpool-agent runtime manager")
	mgr, err := newCRDManager()
	if nil != err {
		logger.Fatal(err.Error())
	}
	agentContext.CRDManager = mgr

	// init managers...
	initAgentServiceManagers(agentContext.InnerCtx)

	logger.Info("Begin to initialize IPAM")
	ipam, err := ipam.NewIPAM(
		ipam.IPAMConfig{
			EnableIPv4:               agentContext.Cfg.EnableIPv4,
			EnableIPv6:               agentContext.Cfg.EnableIPv6,
			ClusterDefaultIPv4IPPool: agentContext.Cfg.ClusterDefaultIPv4IPPool,
			ClusterDefaultIPv6IPPool: agentContext.Cfg.ClusterDefaultIPv6IPPool,
			EnableSpiderSubnet:       agentContext.Cfg.EnableSpiderSubnet,
			EnableStatefulSet:        agentContext.Cfg.EnableStatefulSet,
			OperationRetries:         agentContext.Cfg.WaitSubnetPoolMaxRetries,
			OperationGapDuration:     time.Duration(agentContext.Cfg.WaitSubnetPoolTime) * time.Second,
			LimiterConfig:            limiter.LimiterConfig{MaxQueueSize: &agentContext.Cfg.LimiterMaxQueueSize},
		},
		agentContext.IPPoolManager,
		agentContext.EndpointManager,
		agentContext.NodeManager,
		agentContext.NSManager,
		agentContext.PodManager,
		agentContext.StsManager,
		agentContext.SubnetManager,
	)
	if nil != err {
		logger.Fatal(err.Error())
	}
	agentContext.IPAM = ipam

	go func() {
		logger.Info("Starting IPAM")
		if err := ipam.Start(agentContext.InnerCtx); err != nil {
			logger.Fatal(err.Error())
		}
	}()

	go func() {
		logger.Info("Starting spiderpool-agent runtime manager")
		if err := mgr.Start(agentContext.InnerCtx); err != nil {
			logger.Fatal(err.Error())
		}
	}()

	logger.Info("Begin to initialize spiderpool-agent OpenAPI HTTP server")
	srv, err := newAgentOpenAPIHttpServer()
	if nil != err {
		logger.Fatal(err.Error())
	}
	agentContext.HttpServer = srv

	go func() {
		logger.Info("Starting spiderpool-agent OpenAPI HTTP server")
		if err = srv.Serve(); nil != err {
			if err == http.ErrServerClosed {
				return
			}
			logger.Fatal(err.Error())
		}
	}()

	logger.Info("Begin to initialize spiderpool-agent OpenAPI UNIX server")
	// clean up unix socket path legacy, it won't return an error if it doesn't exist
	if err := os.RemoveAll(agentContext.Cfg.IpamUnixSocketPath); err != nil {
		logger.Sugar().Fatalf("Failed to clean up socket %s: %v", agentContext.Cfg.IpamUnixSocketPath, err)
	}
	unixServer, err := NewAgentOpenAPIUnixServer()
	if nil != err {
		logger.Fatal(err.Error())
	}
	agentContext.UnixServer = unixServer

	go func() {
		logger.Info("Starting spiderpool-agent OpenAPI UNIX server")
		if err = unixServer.Serve(); nil != err {
			if err == net.ErrClosed {
				return
			}
			logger.Fatal(err.Error())
		}
	}()

	spiderpoolAgentAPI, err := NewAgentOpenAPIUnixClient(agentContext.Cfg.IpamUnixSocketPath)
	if nil != err {
		logger.Fatal(err.Error())
	}
	agentContext.unixClient = spiderpoolAgentAPI

	// TODO (Icarus9913): improve k8s StartupProbe
	logger.Info("Set spiderpool-agent startup probe ready")
	agentContext.IsStartupProbe.Store(true)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	WatchSignal(sigCh)
}

// WatchSignal notifies the signal to shut down agentContext handlers.
func WatchSignal(sigCh chan os.Signal) {
	for sig := range sigCh {
		logger.Sugar().Warnw("Received shutdown", "signal", sig)

		// TODO (Icarus9913): filter some signals

		// Cancel the internal context of spiderpool-agent.
		// This stops things like the runtime manager, GC, etc.
		if agentContext.InnerCancel != nil {
			agentContext.InnerCancel()
		}

		// shut down agent http server
		if nil != agentContext.HttpServer {
			if err := agentContext.HttpServer.Shutdown(); nil != err {
				logger.Sugar().Errorf("Failed to shutdown spiderpool-agent HTTP server: %v", err)
			}
		}

		// shut down agent unix server
		if nil != agentContext.UnixServer {
			if err := agentContext.UnixServer.Shutdown(); nil != err {
				logger.Sugar().Errorf("Failed to shut down spiderpool-agent UNIX server: %v", err)
			}
		}

		// others...

	}
}

func initAgentServiceManagers(ctx context.Context) {
	logger.Debug("Begin to initialize Node manager")
	nodeManager, err := nodemanager.NewNodeManager(agentContext.CRDManager.GetClient())
	if err != nil {
		logger.Fatal(err.Error())
	}
	agentContext.NodeManager = nodeManager

	logger.Debug("Begin to initialize Namespace manager")
	nsManager, err := namespacemanager.NewNamespaceManager(agentContext.CRDManager.GetClient())
	if err != nil {
		logger.Fatal(err.Error())
	}
	agentContext.NSManager = nsManager

	logger.Debug("Begin to initialize Pod manager")
	podManager, err := podmanager.NewPodManager(
		podmanager.PodManagerConfig{
			MaxConflictRetries:    agentContext.Cfg.UpdateCRMaxRetries,
			ConflictRetryUnitTime: time.Duration(agentContext.Cfg.UpdateCRRetryUnitTime) * time.Millisecond,
		},
		agentContext.CRDManager.GetClient(),
	)
	if err != nil {
		logger.Fatal(err.Error())
	}
	agentContext.PodManager = podManager

	logger.Debug("Begin to initialize StatefulSet manager")
	statefulSetManager, err := statefulsetmanager.NewStatefulSetManager(agentContext.CRDManager.GetClient())
	if err != nil {
		logger.Fatal(err.Error())
	}
	agentContext.StsManager = statefulSetManager

	logger.Debug("Begin to initialize Endpoint manager")
	endpointManager, err := workloadendpointmanager.NewWorkloadEndpointManager(
		workloadendpointmanager.EndpointManagerConfig{
			MaxConflictRetries:    agentContext.Cfg.UpdateCRMaxRetries,
			ConflictRetryUnitTime: time.Duration(agentContext.Cfg.UpdateCRRetryUnitTime) * time.Millisecond,
			MaxHistoryRecords:     &agentContext.Cfg.WorkloadEndpointMaxHistoryRecords,
		},
		agentContext.CRDManager.GetClient(),
	)
	if err != nil {
		logger.Fatal(err.Error())
	}
	agentContext.EndpointManager = endpointManager

	logger.Debug("Begin to initialize ReservedIP manager")
	rIPManager, err := reservedipmanager.NewReservedIPManager(agentContext.CRDManager.GetClient())
	if err != nil {
		logger.Fatal(err.Error())
	}
	agentContext.RIPManager = rIPManager

	logger.Debug("Begin to initialize IPPool manager")
	ipPoolManager, err := ippoolmanager.NewIPPoolManager(
		ippoolmanager.IPPoolManagerConfig{
			MaxConflictRetries:    agentContext.Cfg.UpdateCRMaxRetries,
			ConflictRetryUnitTime: time.Duration(agentContext.Cfg.UpdateCRRetryUnitTime) * time.Millisecond,
			MaxAllocatedIPs:       &agentContext.Cfg.IPPoolMaxAllocatedIPs,
		},
		agentContext.CRDManager.GetClient(),
		agentContext.RIPManager,
	)
	if err != nil {
		logger.Fatal(err.Error())
	}
	agentContext.IPPoolManager = ipPoolManager

	if agentContext.Cfg.EnableSpiderSubnet {
		logger.Debug("Begin to initialize Subnet manager")
		subnetManager, err := subnetmanager.NewSubnetManager(
			subnetmanager.SubnetManagerConfig{
				MaxConflictRetries:    agentContext.Cfg.UpdateCRMaxRetries,
				ConflictRetryUnitTime: time.Duration(agentContext.Cfg.UpdateCRRetryUnitTime) * time.Millisecond,
			},
			agentContext.CRDManager.GetClient(),
			agentContext.IPPoolManager,
			agentContext.CRDManager.GetScheme(),
		)
		if err != nil {
			logger.Fatal(err.Error())
		}
		agentContext.SubnetManager = subnetManager
	} else {
		logger.Info("Feature SpiderSubnet is disabled")
	}
}
