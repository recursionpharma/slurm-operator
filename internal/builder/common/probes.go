// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AnnotationDisableHealthProbes opts a workload's Slurm daemon container out of
// the /livez and /readyz HTTP probes.
//
// Slurm only serves those HTTP endpoints from 25.11 onward (slurmctld/slurmd
// "http.c"). On Slurm 25.05.x images the probes can never succeed, so the
// kubelet kills the daemon shortly after start (startup probe failure) in a
// restart loop. Workloads pinned to Slurm < 25.11 should set this annotation
// to "true" on their Controller / NodeSet resource.
const AnnotationDisableHealthProbes = "slinky.slurm.net/disable-health-probes"

// DisableHealthProbes reports whether obj opts out of the Slurm daemon health
// probes.
func DisableHealthProbes(obj metav1.Object) bool {
	return obj != nil && obj.GetAnnotations()[AnnotationDisableHealthProbes] == "true"
}
