# Data Spacer

Kubernetes controller and webhook to use in conjunction with [Liqo](https://github.com/liqotech/liqo) for data spaces.

## Project structure

The **build** folder contains the Dockerfiles required to build the container images used in this project:

- **build/init-container** contains the Dockerfile for building and running the init container injected in the mutated pod;

The **controllers** folder contains the controller's reconcile logic to reconcile Namespace resources.

The **deploy** folder contains the manifests, divided by sub-folders, required to deploy the project.

The **pkg** folder contains the webhook logic.

The **demo** folder contains some example manifests and applications used to run a demo:

- **demo/manifests** contains manifests to deploy example namespaces, services, deployments and pods;
- **demo/build/pharma-app** and **demo/build/hospital-app** contain each a Dockerfile for building and running an example Node.js app contained therein.

## About this project

To better show what this project aims to achieve, a real-world scenario is proposed and will guide the reader throughout the next sections.

The scenario is the one made of two different administrative entities, each owning a K8s cluster on its own, represented by a pharmaceutical company ("Pharma" cluster) and a hospital ("Hospital" cluster).
The former is willing to access the latter's data to run its own software on and extract some valuable information.
The latter is willing to host the former's workloads under the condition of always being able to control what such foreign workloads (i.e. pods executing in its cluster) do and what they cannot do.
Indeed, the Hospital has some sensitive data and wants to ensure this data is not misused by the Pharma pods that are running within the Hospital borders.
This configures a use-case under which a modern _data space_ approach fits well.

The backbone that makes all this possible, aside from Kubernetes itself, is Liqo, an open-source project that enables a dynamic and seamless multi-cluster environment.
By installing Liqo in both clusters, their collaboration can start up.

The Data Spacer project constitues the additional module that, once deployed in the cluster that hosts external workloads (i.e. Hospital), enables the features that secure the Liqo-based collaboration between the two clusters.

## Main concepts

The main concepts are visualized in the following image:
![Project schema](data-spacer.png)

The Pharma Cluster establishes a peering session with the Hospital Cluster, to which it offloads its _pharma-space_ and so its _pharma-app_ pod. The _pharma-space_ in the Hospital Cluster is reconciled by the namespace controller (_liqo.io/remote-cluster-id_) and is enriched by default with two Kubernetes resources:

- a ConfigMap that stores the proper Envoy Proxy configuration required to mutate the pod and inject a proxy sidecar (adding the _data-space/apply-webhook=false_ label will prevent this);
- a NetworkPolicy for controlling egress and ingress traffic from the mutated pod (adding the _data-space/apply-netpol=false_ label will prevent this).

Once Pharma offloads its pods to Hospital, they are mutated by a mutating admission control webhook.
This injects a sidecar container that implements a proxy that forwards requests and responses from and to the Pharma's application running in the offloaded pod. This proxy is currently configured to work as a monitoring tool, and is based on the Envoy Proxy open-source project.
To ensure the traffic always goes through the proxy, the webhook also injects an init container that creates the required iptables rules.

Once the Pharma pods are running in the Hospital cluster, they are restricted in their communication by the network policy that is applied on the Hospital's namespace that hosts these external workloads.

Other security measures are also enforced by Liqo itself, to limit pod-to-pod traffic between the two clusters.

## How it works

The controller's reconcile logic watches and handles Namespace resources that are labeled with _liqo.io/remote-cluster-id_. These namespaces are enriched by default with:

- a ConfigMap resource that will contain the Envoy Proxy configuration to be used by the webhook to inject the proxy sidecar into all pods hosted on the same namespace;
- a NetworkPolicy resource that only allows egress destinations and ingress sources labeled with _data-space/netpol-allow=true_.

However, it is possible to prevent the creation of one or both of those two resources by respectively setting the following labels on Namespace resources:

- _data-space/apply-webhook=false_. This also prevents pods executed in such namespaces from being mutated by the mutating webhook.
- _data-space/apply-netpol=false_

## Demo applications

The **demo/build/hospital-app** Node.js application exemplifies a web server exposing some _fictional_ data about some patients under two endpoints, _/patients_ and _/patients/:id_.

The **demo/build/pharma-app** Node.js application exemplifies a client that makes HTTP requests. To make the interaction easier, this application also works as a web server by exposing an endpoint under _/simulate?url_ that lets users choose the endpoint to make the HTTP request to. Alternatively, the _/processed-data_ endpoint performs an HTTP request to the other hospital-app's _/patients_ endpoint.

## License

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
