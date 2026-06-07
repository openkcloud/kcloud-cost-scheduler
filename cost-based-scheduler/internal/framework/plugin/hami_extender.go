/*
Hardware Aware Metrics Integration Extender

This module integrates with the HAMI scheduler extender to account for
advanced hardware topologies.
*/
package plugin

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type ExtenderArgs struct {
	Pod       *v1.Pod    `json:"pod"`
	Nodes     []*v1.Node `json:"nodes,omitempty"`
	NodeNames []string   `json:"nodeNames,omitempty"`
}

type ExtenderFilterResult struct {
	Nodes       *v1.NodeList `json:"nodes,omitempty"`
	NodeNames   *[]string    `json:"nodeNames,omitempty"`
	FailedNodes map[string]string `json:"failedNodes,omitempty"`
	Error       string       `json:"error,omitempty"`
}

type ExtenderBindingArgs struct {
	PodName      string `json:"podName"`
	PodNamespace string `json:"podNamespace"`
	PodUID       string `json:"podUID"`
	Node         string `json:"node"`
}

type HAMiExtenderClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHAMiExtenderClient() *HAMiExtenderClient {
	return &HAMiExtenderClient{
		baseURL: "https://10.109.140.73:443",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (c *HAMiExtenderClient) Filter(pod *v1.Pod, nodes []*v1.Node) ([]*v1.Node, error) {
	if !hasGPURequest(pod) {
		return nodes, nil
	}

	nodeNames := make([]string, 0, len(nodes))
	for _, n := range nodes {
		nodeNames = append(nodeNames, n.Name)
	}

	args := ExtenderArgs{
		Pod:       pod,
		NodeNames: nodeNames,
	}

	payload, _ := json.Marshal(args)
	resp, err := c.httpClient.Post(c.baseURL+"/filter", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		klog.Warningf("HAMi Extender filter failed (falling back): %v", err)
		return nodes, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		klog.Warningf("HAMi Extender returned non-OK status: %d", resp.StatusCode)
		return nodes, nil
	}

	var result ExtenderFilterResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nodes, nil
	}

	if result.NodeNames == nil {
		return nodes, nil
	}

	filteredNodes := make([]*v1.Node, 0)
	keepSet := make(map[string]bool)
	for _, name := range *result.NodeNames {
		keepSet[name] = true
	}

	for _, n := range nodes {
		if keepSet[n.Name] {
			filteredNodes = append(filteredNodes, n)
		}
	}

	return filteredNodes, nil
}

func (c *HAMiExtenderClient) Bind(pod *v1.Pod, node string) error {
	args := ExtenderBindingArgs{
		PodName:      pod.Name,
		PodNamespace: pod.Namespace,
		PodUID:       string(pod.UID),
		Node:         node,
	}

	payload, _ := json.Marshal(args)
	resp, err := c.httpClient.Post(c.baseURL+"/bind", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HAMi bind status: %d", resp.StatusCode)
	}

	return nil
}

func hasGPURequest(pod *v1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if _, ok := container.Resources.Requests["nvidia.com/gpu"]; ok {
			return true
		}
	}
	return false
}
