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

package common

import (
	corev1 "k8s.io/api/core/v1"
)

// InjectPodLabel adds a label (key, value pair) in the pod's metadata
func InjectPodLabel(pod *corev1.Pod, key, value string) {
	labels := pod.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
		pod.SetLabels(labels)
	}
	pod.ObjectMeta.Labels[key] = value
}
