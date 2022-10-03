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

package controllers

type Route struct {
	Cluster string `yaml:"cluster"`
}
type Match struct {
	Prefix string `yaml:"prefix"`
}
type HostRoute struct {
	Name  string `yaml:"name"`
	Match Match  `yaml:"match"`
	Route Route  `yaml:"route"`
}
type VirtualHost struct {
	Name    string      `yaml:"name"`
	Domains []string    `yaml:"domains,flow"`
	Routes  []HostRoute `yaml:"routes"`
}
type RouteConfig struct {
	Name         string        `yaml:"name"`
	VirtualHosts []VirtualHost `yaml:"virtual_hosts"`
}
type TypedConfig struct {
	Type        string          `yaml:"@type"`
	StatPrefix  string          `yaml:"stat_prefix,omitempty"`
	Cluster     string          `yaml:"cluster,omitempty"`
	AccessLog   []NameAndConfig `yaml:"access_log,omitempty"`
	HttpFilters []NameAndConfig `yaml:"http_filters,omitempty"`
	RouteConfig RouteConfig     `yaml:"route_config,omitempty"`
}
type NameAndConfig struct {
	Name        string      `yaml:"name"`
	TypedConfig TypedConfig `yaml:"typed_config"`
}
type SocketAddress struct {
	Address   string `yaml:"address"`
	PortValue uint16 `yaml:"port_value"`
}
type Address struct {
	SocketAddress SocketAddress `yaml:"socket_address"`
}
type FilterChain struct {
	Filters []NameAndConfig `yaml:"filters"`
}
type Listener struct {
	Name            string          `yaml:"name"`
	Address         Address         `yaml:"address"`
	ListenerFilters []NameAndConfig `yaml:"listener_filters,omitempty"`
	FilterChains    []FilterChain   `yaml:"filter_chains"`
}

type OriginalDstLbConfig struct {
	UseHttpHeader bool `yaml:"use_http_header"`
}
type Cluster struct {
	Name                string              `yaml:"name"`
	DnsLookupFamily     string              `yaml:"dns_lookup_family"`
	Type                string              `yaml:"type"`
	LbPolicy            string              `yaml:"lb_policy"`
	ConnectTimeout      string              `yaml:"connect_timeout"`
	OriginalDstLbConfig OriginalDstLbConfig `yaml:"original_dst_lb_config"`
}

type Admin struct {
	Address Address `yaml:"address"`
}
type StaticResources struct {
	Listeners []Listener `yaml:"listeners"`
	Clusters  []Cluster  `yaml:"clusters"`
}
type EnvoyConfig struct {
	// Admin interface for stats and configs
	Admin Admin `yaml:"admin"`
	// Configuration at startup time
	StaticResources StaticResources `yaml:"static_resources"`
}
