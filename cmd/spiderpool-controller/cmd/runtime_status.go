// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/go-openapi/runtime/middleware"

	"github.com/spidernet-io/spiderpool/api/v1/controller/server/restapi/runtime"
)

// Singleton
var (
	httpGetControllerStartup   = &_httpGetControllerStartup{controllerContext}
	httpGetControllerReadiness = &_httpGetControllerReadiness{controllerContext}
	httpGetControllerLiveness  = &_httpGetControllerLiveness{controllerContext}
)

type _httpGetControllerStartup struct {
	*ControllerContext
}

// Handle handles GET requests for k8s startup probe.
func (g *_httpGetControllerStartup) Handle(params runtime.GetRuntimeStartupParams) middleware.Responder {
	if !g.IsStartupProbe.Load() {
		return runtime.NewGetRuntimeStartupInternalServerError()
	}

	return runtime.NewGetRuntimeStartupOK()
}

type _httpGetControllerReadiness struct {
	*ControllerContext
}

// Handle handles GET requests for k8s readiness probe.
func (g *_httpGetControllerReadiness) Handle(params runtime.GetRuntimeReadinessParams) middleware.Responder {
	if err := WebhookHealthyCheck(g.webhookClient, g.Cfg.WebhookPort); err != nil {
		logger.Sugar().Errorf("failed to check spiderpool controller readiness probe, error: %v", err)
		return runtime.NewGetRuntimeReadinessInternalServerError()
	}

	if len(g.Leader.GetLeader()) == 0 {
		logger.Warn("there's not leader in the current cluster, please wait for a while")
		return runtime.NewGetRuntimeReadinessInternalServerError()
	}

	if g.Leader.IsElected() {
		if !g.GCManager.Health() {
			logger.Sugar().Warnf("the IP GC is still not ready, please wait for a while")
			return runtime.NewGetRuntimeReadinessInternalServerError()
		}
	}

	return runtime.NewGetRuntimeReadinessOK()
}

type _httpGetControllerLiveness struct {
	*ControllerContext
}

// Handle handles GET requests for k8s liveness probe.
func (g *_httpGetControllerLiveness) Handle(params runtime.GetRuntimeLivenessParams) middleware.Responder {
	if err := WebhookHealthyCheck(g.webhookClient, g.Cfg.WebhookPort); err != nil {
		logger.Sugar().Errorf("failed to check spiderpool controller liveness probe, error: %v", err)
		return runtime.NewGetRuntimeLivenessInternalServerError()
	}

	return runtime.NewGetRuntimeLivenessOK()
}
