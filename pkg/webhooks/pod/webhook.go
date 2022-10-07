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
	client    client.Client
	decoder   *admission.Decoder
	initImage string
}

func New(cl client.Client, initImage string) *webhook.Admission {
	return &webhook.Admission{Handler: &mutator{client: cl, initImage: initImage}}
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

	m.mutatePod(&pod)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		klog.Errorf("Failed encoding Pod in response: %v", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// mutatePod mutates the pod
func (m *mutator) mutatePod(pod *corev1.Pod) {
	injectVolume(pod)
	m.injectInit(pod)
	common.InjectPodLabel(pod, consts.DataSpaceNetpolAllowLabel, "true")
	injectEnvVars(pod)
	injectSecCtxs(pod)
	injectSidecar(pod)
}

// injectVolume adds a volume associated to a config map
// that configures Envoy
func injectVolume(pod *corev1.Pod) {
	volume := corev1.Volume{
		Name: "envoy-config-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: consts.ConfigMapName,
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

// injectInit adds an init container as the first one to be executed
// to set iptables rules for pod's egress and ingress traffic
func (m *mutator) injectInit(pod *corev1.Pod) {
	privileged := false
	initContainer := corev1.Container{
		Name:            "init",
		Image:           m.initImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"sh", "-c"},
		Args: []string{
			fmt.Sprint(
				// INBOUND RULES
				// Create INBOUND_REDIRECT chain and attach it to the 'nat' table
				fmt.Sprintf("iptables -t nat -N %s;", consts.InboundChainName),
				// Redirect all incoming TCP packets to port 13041 (to which Envoy proxy is configured to listen for incoming traffic)
				fmt.Sprintf("iptables -t nat -A %s -p tcp -j REDIRECT --to-port %d;", consts.InboundChainName, consts.IngressTcpPort),
				// Make PREROUTING chain (first chain a packet traverses inbound) send packets to INBOUND_REDIRECT chain
				fmt.Sprintf("iptables -t nat -A PREROUTING -j %s;", consts.InboundChainName),

				// OUTBOUND RULES
				// Create OUTBOUND_REDIRECT chain and attach it to the 'nat' table
				fmt.Sprintf("iptables -t nat -N %s;", consts.OutboundChainName),
				// Skip next rules in OUTBOUND_REDIRECT chain for packets owned by the proxy (uid 1303) and return to the previous chain
				fmt.Sprintf("iptables -t nat -A %s -m owner --uid-owner %d -j RETURN;", consts.OutboundChainName, consts.ProxyUid),
				// Skip next rules in OUTBOUND_REDIRECT chain for packets sent to the loopback interface and return to the previous chain
				fmt.Sprintf("iptables -t nat -A %s -o lo -j RETURN;", consts.OutboundChainName),
				// Redirect all outgoing TCP packets to port 13031 (to which Envoy proxy is configured to listen for outgoing traffic)
				fmt.Sprintf("iptables -t nat -A %s -p tcp -j REDIRECT --to-port %d;", consts.OutboundChainName, consts.EgressTcpPort),
				// Make OUTPUT chain (second-last chain a packet traverses outbound) send packets to OUTBOUND_REDIRECT chain
				fmt.Sprintf("iptables -t nat -A OUTPUT -j %s;", consts.OutboundChainName),
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
	pod.Spec.InitContainers = append([]corev1.Container{initContainer}, pod.Spec.InitContainers...)
}

// injectEnvVars adds two environment variables to be used by various tools
func injectEnvVars(pod *corev1.Pod) {
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env,
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
}

// injectSecCtxs adds a security context to each pod's container
func injectSecCtxs(pod *corev1.Pod) {
	privileged := false
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].SecurityContext = &corev1.SecurityContext{
			Privileged:               &privileged,
			AllowPrivilegeEscalation: &privileged,
		}
	}
}

// injectSidecar adds a sidecar container running the Envoy proxy image
// that will mount the injected volume associated with the config map
// to configure Envoy.
func injectSidecar(pod *corev1.Pod) {
	privileged := false
	user := int64(consts.ProxyUid)
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
