// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"go.uber.org/atomic"
	"gopkg.in/yaml.v3"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/spidernet-io/spiderpool/api/v1/agent/client"
	"github.com/spidernet-io/spiderpool/api/v1/agent/server"
	"github.com/spidernet-io/spiderpool/pkg/constant"
	"github.com/spidernet-io/spiderpool/pkg/ipam"
	"github.com/spidernet-io/spiderpool/pkg/ippoolmanager"
	"github.com/spidernet-io/spiderpool/pkg/logutils"
	"github.com/spidernet-io/spiderpool/pkg/namespacemanager"
	"github.com/spidernet-io/spiderpool/pkg/nodemanager"
	"github.com/spidernet-io/spiderpool/pkg/podmanager"
	"github.com/spidernet-io/spiderpool/pkg/reservedipmanager"
	"github.com/spidernet-io/spiderpool/pkg/statefulsetmanager"
	"github.com/spidernet-io/spiderpool/pkg/subnetmanager"
	"github.com/spidernet-io/spiderpool/pkg/workloadendpointmanager"
)

var agentContext = new(AgentContext)

type envConf struct {
	envName          string
	defaultValue     string
	required         bool
	associateStrKey  *string
	associateBoolKey *bool
	associateIntKey  *int
}

// EnvInfo collects the env and relevant agentContext properties.
var envInfo = []envConf{
	{"GIT_COMMIT_VERSION", "", false, &agentContext.Cfg.CommitVersion, nil, nil},
	{"GIT_COMMIT_TIME", "", false, &agentContext.Cfg.CommitTime, nil, nil},
	{"VERSION", "", false, &agentContext.Cfg.AppVersion, nil, nil},
	{"GOLANG_ENV_MAXPROCS", "8", false, nil, nil, &agentContext.Cfg.GoMaxProcs},

	{"SPIDERPOOL_LOG_LEVEL", logutils.LogInfoLevelStr, true, &agentContext.Cfg.LogLevel, nil, nil},
	{"SPIDERPOOL_ENABLED_METRIC", "false", false, nil, &agentContext.Cfg.EnableMetric, nil},
	{"SPIDERPOOL_ENABLED_DEBUG_METRIC", "false", false, nil, &agentContext.Cfg.EnableDebugLevelMetric, nil},
	{"SPIDERPOOL_HEALTH_PORT", "5710", true, &agentContext.Cfg.HttpPort, nil, nil},
	{"SPIDERPOOL_METRIC_HTTP_PORT", "5711", true, &agentContext.Cfg.MetricHttpPort, nil, nil},
	{"SPIDERPOOL_GOPS_LISTEN_PORT", "5712", false, &agentContext.Cfg.GopsListenPort, nil, nil},
	{"SPIDERPOOL_PYROSCOPE_PUSH_SERVER_ADDRESS", "", false, &agentContext.Cfg.PyroscopeAddress, nil, nil},

	{"SPIDERPOOL_IPPOOL_MAX_ALLOCATED_IPS", "5000", true, nil, nil, &agentContext.Cfg.IPPoolMaxAllocatedIPs},
	{"SPIDERPOOL_WAIT_SUBNET_POOL_TIME_IN_SECOND", "2", false, nil, nil, &agentContext.Cfg.WaitSubnetPoolTime},
	{"SPIDERPOOL_WAIT_SUBNET_POOL_MAX_RETRIES", "25", false, nil, nil, &agentContext.Cfg.WaitSubnetPoolMaxRetries},
}

type Config struct {
	CommitVersion string
	CommitTime    string
	AppVersion    string
	GoMaxProcs    int

	// flags
	ConfigPath string

	// env
	LogLevel               string
	EnableMetric           bool
	EnableDebugLevelMetric bool

	HttpPort         string
	MetricHttpPort   string
	GopsListenPort   string
	PyroscopeAddress string

	IPPoolMaxAllocatedIPs    int
	WaitSubnetPoolTime       int
	WaitSubnetPoolMaxRetries int

	// configmap
	IpamUnixSocketPath                string   `yaml:"ipamUnixSocketPath"`
	EnableIPv4                        bool     `yaml:"enableIPv4"`
	EnableIPv6                        bool     `yaml:"enableIPv6"`
	EnableStatefulSet                 bool     `yaml:"enableStatefulSet"`
	EnableSpiderSubnet                bool     `yaml:"enableSpiderSubnet"`
	ClusterDefaultIPv4IPPool          []string `yaml:"clusterDefaultIPv4IPPool"`
	ClusterDefaultIPv6IPPool          []string `yaml:"clusterDefaultIPv6IPPool"`
	ClusterDefaultIPv4Subnet          []string `yaml:"clusterDefaultIPv4Subnet"`
	ClusterDefaultIPv6Subnet          []string `yaml:"clusterDefaultIPv6Subnet"`
	ClusterSubnetDefaultFlexibleIPNum int      `yaml:"clusterSubnetDefaultFlexibleIPNumber"`
}

type AgentContext struct {
	Cfg Config

	// InnerCtx is the context that can be used during shutdown.
	// It will be cancelled after receiving an interrupt or termination signal.
	InnerCtx    context.Context
	InnerCancel context.CancelFunc

	// manager
	IPAM              ipam.IPAM
	CRDManager        ctrl.Manager
	IPPoolManager     ippoolmanager.IPPoolManager
	EndpointManager   workloadendpointmanager.WorkloadEndpointManager
	ReservedIPManager reservedipmanager.ReservedIPManager
	NodeManager       nodemanager.NodeManager
	NSManager         namespacemanager.NamespaceManager
	PodManager        podmanager.PodManager
	StsManager        statefulsetmanager.StatefulSetManager
	SubnetManager     subnetmanager.SubnetManager

	// handler
	HttpServer        *server.Server
	UnixServer        *server.Server
	MetricsHttpServer *http.Server

	// client
	unixClient *client.SpiderpoolAgentAPI

	// probe
	IsStartupProbe atomic.Bool
}

// BindAgentDaemonFlags bind agent cli daemon flags
func (ac *AgentContext) BindAgentDaemonFlags(flags *pflag.FlagSet) {
	flags.StringVar(&ac.Cfg.ConfigPath, "config-path", "/tmp/spiderpool/config-map/conf.yml", "spiderpool-agent configmap file")
}

// ParseConfiguration set the env to AgentConfiguration
func ParseConfiguration() error {
	var result string

	for i := range envInfo {
		env, ok := os.LookupEnv(envInfo[i].envName)
		if ok {
			result = strings.TrimSpace(env)
		} else {
			// if no env and required, set it to default value.
			result = envInfo[i].defaultValue
		}
		if len(result) == 0 {
			if envInfo[i].required {
				logger.Fatal(fmt.Sprintf("empty value of %s", envInfo[i].envName))
			} else {
				// if no env and none-required, just use the empty value.
				continue
			}
		}

		if envInfo[i].associateStrKey != nil {
			*(envInfo[i].associateStrKey) = result
		} else if envInfo[i].associateBoolKey != nil {
			b, err := strconv.ParseBool(result)
			if nil != err {
				return fmt.Errorf("error: %s require a bool value, but get %s", envInfo[i].envName, result)
			}
			*(envInfo[i].associateBoolKey) = b
		} else if envInfo[i].associateIntKey != nil {
			intVal, err := strconv.Atoi(result)
			if nil != err {
				return fmt.Errorf("error: %s require a int value, but get %s", envInfo[i].envName, result)
			}
			*(envInfo[i].associateIntKey) = intVal
		} else {
			return fmt.Errorf("error: %s doesn't match any controller context", envInfo[i].envName)
		}
	}

	return nil
}

// LoadConfigmap reads configmap data from cli flag config-path
func (ac *AgentContext) LoadConfigmap() error {
	configmapBytes, err := os.ReadFile(ac.Cfg.ConfigPath)
	if nil != err {
		return fmt.Errorf("failed to read configmap file, error: %v", err)
	}

	err = yaml.Unmarshal(configmapBytes, &ac.Cfg)
	if nil != err {
		return fmt.Errorf("failed to parse configmap, error: %v", err)
	}

	if ac.Cfg.IpamUnixSocketPath == "" {
		ac.Cfg.IpamUnixSocketPath = constant.DefaultIPAMUnixSocketPath
	}

	return nil
}
