/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"data-space.liqo.io/common"
	"data-space.liqo.io/consts"
)

type mutator struct {
	client  client.Client
	decoder *admission.Decoder
}

func New(cl client.Client) *webhook.Admission {
	return &webhook.Admission{Handler: &mutator{client: cl}}
}

// InjectDecoder injects the decoder - this method is used by controller runtime.
func (m *mutator) InjectDecoder(decoder *admission.Decoder) error {
	m.decoder = decoder
	return nil
}

// Handle implements the mutating webhook.
//
//nolint:gocritic // The signature of this method is imposed by controller runtime.
func (m *mutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	var pod corev1.Pod
	if err := m.decoder.Decode(req, &pod); err != nil {
		klog.Errorf("Failed decoding Pod from request: %v", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	mutatePod(&pod)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		klog.Errorf("Failed encoding Pod in response: %v", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func mutatePod(pod *corev1.Pod) {
	injectVolume(pod)
	injectInit(pod)
	injectEnvVar(pod)
	common.InjectPodLabel(pod, consts.MutatedPodLabel, "true")
	injectSecCtx(pod)
	injectSidecar(pod)
}

func injectVolume(pod *corev1.Pod) {
	volume := corev1.Volume{
		Name: "envoy-config-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "envoy-config",
				},
				Items: []corev1.KeyToPath{{
					Key:  "keys",
					Path: "envoy.yaml",
				}},
			},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
}

func injectInit(pod *corev1.Pod) {
	privileged := false
	initContainer := corev1.Container{
		Name:    "init",
		Image:   "alpine:3.16",
		Command: []string{"sh", "-c"},
		Args: []string{
			fmt.Sprint(
				"apk add --no-cache iptables;",
				"iptables -t nat -N PROXY_INIT_REDIRECT;",
				"iptables -t nat -A PROXY_INIT_REDIRECT -p tcp -j REDIRECT --to-port 13041;",
				"iptables -t nat -A PREROUTING -j PROXY_INIT_REDIRECT;",
				"iptables -t nat -N PROXY_INIT_OUTPUT;",
				"iptables -t nat -A PROXY_INIT_OUTPUT -m owner --uid-owner 1303 -j RETURN;",
				"iptables -t nat -A PROXY_INIT_OUTPUT -o lo -j RETURN;",
				"iptables -t nat -A PROXY_INIT_OUTPUT -p tcp -j REDIRECT --to-port 13031;",
				"iptables -t nat -A OUTPUT -j PROXY_INIT_OUTPUT;",
			),
		},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{
					corev1.Capability("NET_ADMIN"),
				},
			},
			Privileged:               &privileged,
			AllowPrivilegeEscalation: &privileged,
		},
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initContainer)
}

func injectEnvVar(pod *corev1.Pod) {
	pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env,
		corev1.EnvVar{
			Name:  "http_proxy",
			Value: fmt.Sprintf("%s:%v", consts.LOCALHOST_ADDR, consts.EgressHttpPort),
		},
		corev1.EnvVar{
			Name:  "HTTP_PROXY",
			Value: fmt.Sprintf("%s:%v", consts.LOCALHOST_ADDR, consts.EgressHttpPort),
		},
	)
}

func injectSecCtx(pod *corev1.Pod) {
	privileged := false
	pod.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
		Privileged:               &privileged,
		AllowPrivilegeEscalation: &privileged,
	}
}

func injectSidecar(pod *corev1.Pod) {
	privileged := false
	user := int64(1303)
	sidecar := corev1.Container{
		Name:  "proxy",
		Image: "envoyproxy/envoy:v1.23.1",
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                &user,
			RunAsGroup:               &user,
			Privileged:               &privileged,
			AllowPrivilegeEscalation: &privileged,
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "envoy-config-volume",
			MountPath: "/etc/envoy/envoy.yaml",
			SubPath:   "envoy.yaml",
		}},
	}
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)
}
