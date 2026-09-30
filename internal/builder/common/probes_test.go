// SPDX-FileCopyrightText: Copyright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDisableHealthProbes(t *testing.T) {
	tests := []struct {
		name string
		obj  metav1.Object
		want bool
	}{
		{"nil object", nil, false},
		{"no annotations", &metav1.ObjectMeta{}, false},
		{"annotation false", &metav1.ObjectMeta{Annotations: map[string]string{AnnotationDisableHealthProbes: "false"}}, false},
		{"annotation true", &metav1.ObjectMeta{Annotations: map[string]string{AnnotationDisableHealthProbes: "true"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisableHealthProbes(tt.obj); got != tt.want {
				t.Errorf("DisableHealthProbes() = %v, want %v", got, tt.want)
			}
		})
	}
}
