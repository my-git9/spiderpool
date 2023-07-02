// Copyright 2023 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"strings"
	"os"
)

const (
	ENV_INSTALL_OVERLAY          = "INSTALL_OVERLAY_CNI"
	E2E_SPIDERPOOL_ENABLE_SUBNET = "E2E_SPIDERPOOL_ENABLE_SUBNET"
)

func checkBoolEnv(name string) bool {
	t := os.Getenv(name)
	if strings.ToLower(t) != "true" {
		return false
	} else {
		return true
	}
}

func CheckRunOverlayCNI() bool {
	return checkBoolEnv(ENV_INSTALL_OVERLAY)
}

func CheckSubnetFeatureOn() bool {
	return checkBoolEnv(E2E_SPIDERPOOL_ENABLE_SUBNET)
}
