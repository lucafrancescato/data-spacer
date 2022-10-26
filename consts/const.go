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

package consts

// Network constants
const (
	ANY_ADDR       = "0.0.0.0"
	LOCALHOST_ADDR = "127.0.0.1"
)

// App constants
const (
	// Setting DataSpaceApplyNetpolLabel to "true" in a namespace ensures
	// the NetworkPolicy resource is created within that namespace
	DataSpaceApplyNetpolLabel = "data-space/apply-netpol"
	// Setting DataSpaceApplyWebhookLabel to "true" in a namespace ensures
	// all scheduled pod in that namespace gets mutated by the webhook
	DataSpaceApplyWebhookLabel = "data-space/apply-webhook"
	// Setting DataSpaceNetpolAllowLabel to "true" in a namespace or pod ensures
	// they are selected as allowed ingress sources or egress destinations by the
	// NetworkPolicy created in the namespace that has DataSpaceApplyNetpolLabel set to "true"
	DataSpaceNetpolAllowLabel = "data-space/netpol-allow"

	// RemoteClusterIdLabel is the cluster ID of a remote cluster that
	// has established a Liqo peering with the local cluster
	RemoteClusterIdLabel = "liqo.io/remote-cluster-id"

	// Network policy name
	NetworkPolicyName = "data-space-network-policy"
	// Config map name
	ConfigMapName = "data-space-envoy-config"

	// Proxy user id
	ProxyUid = 1303

	// Iptables inbound chain name
	InboundChainName = "INBOUND_REDIRECT"
	// Iptables outbound chain name
	OutboundChainName = "OUTBOUND_REDIRECT"
)

// DNS constants
const (
	K8sAppLabel = "k8s-app"
	KUBE_SYSTEM = "kube-system"
	KUBE_DNS    = "kube-dns"
)

// Envoy config constants
const (
	AdminPort                 = 9901
	EgressTcpPort             = 13031
	EgressHttpPort            = 13032
	IngressTcpPort            = 13041
	IngressHttpPort           = 13042
	EgressClusterName         = "egress_cluster"
	IngressClusterName        = "ingress_cluster"
	EgressForwardClusterName  = "egress_forward_cluster"
	IngressForwardClusterName = "ingress_forward_cluster"
	DnsCacheConfigName        = "forward_dns_cache_config"
	EgressTcpStatPrefixName   = "egress_tcp"
	IngressTcpStatPrefixName  = "ingress_tcp"
	EgressHttpStatPrefixName  = "egress_http"
	IngressHttpStatPrefixName = "ingress_http"
	EgressHttpTapConfigId     = "tap_egress_http_id"
	IngressHttpTapConfigId    = "tap_ingress_http_id"
)

// Envoy API constants
const (
	OriginalDstListenerFilterTypeName  = "envoy.filters.listener.original_dst"
	OriginalDstListenerFilterTypeUrl   = "type.googleapis.com/envoy.extensions.filters.listener.original_dst.v3.OriginalDst"
	TcpProxyTypeName                   = "envoy.filters.network.tcp_proxy"
	TcpProxyTypeUrl                    = "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy"
	HttpConnectionManagerTypeName      = "envoy.filters.network.http_connection_manager"
	HttpConnectionManagerTypeUrl       = "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"
	DynamicForwardProxyFilterTypeName  = "envoy.filters.http.dynamic_forward_proxy"
	DynamicForwardProxyFilterTypeUrl   = "type.googleapis.com/envoy.extensions.filters.http.dynamic_forward_proxy.v3.FilterConfig"
	DynamicForwardProxyRouteTypeUrl    = "type.googleapis.com/envoy.extensions.filters.http.dynamic_forward_proxy.v3.PerRouteConfig"
	DynamicForwardProxyClusterTypeName = "envoy.clusters.dynamic_forward_proxy"
	DynamicForwardProxyClusterTypeUrl  = "type.googleapis.com/envoy.extensions.clusters.dynamic_forward_proxy.v3.ClusterConfig"
	HttpTapTypeName                    = "envoy.filters.http.tap"
	HttpTapTypeUrl                     = "type.googleapis.com/envoy.extensions.filters.http.tap.v3.Tap"
	RouterTypeName                     = "envoy.filters.http.router"
	RouterTypeUrl                      = "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
	AccessLogTypeName                  = "envoy.access_loggers.stdout"
	AccessLogTypeUrl                   = "type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog"
	OriginalDstType                    = "ORIGINAL_DST"
	OriginalDstLbPolicy                = "CLUSTER_PROVIDED"
	Ipv4Only                           = "V4_ONLY"
	JsonBodyAsString                   = "JSON_BODY_AS_STRING"
)
