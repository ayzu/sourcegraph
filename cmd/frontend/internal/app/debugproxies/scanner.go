package debugproxies

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/inconshreveable/log15"
)

// Represents an endpoint
type Endpoint struct {
	// Service to which the endpoint belongs
	Service string
	// Host:port, so hostname part of a URL (ip address ok)
	Host string
}

// ScanConsumer is the callback to consume scan results.
type ScanConsumer func([]Endpoint)

// Declares methods we use with k8s.Client. Useful to plug testing replacements or even logging middleware.
type kubernetesClient interface {
	Watch(ctx context.Context, namespace string, r k8s.Resource, options ...k8s.Option) (*k8s.Watcher, error)
	List(ctx context.Context, namespace string, resp k8s.ResourceList, options ...k8s.Option) error
	Get(ctx context.Context, namespace, name string, resp k8s.Resource, options ...k8s.Option) error
	Namespace() string
}

// "real" implementation that sends calls to the k8s.Client
type k8sClientImpl struct {
	client *k8s.Client
}

func (kci *k8sClientImpl) Watch(ctx context.Context, namespace string, r k8s.Resource, options ...k8s.Option) (*k8s.Watcher, error) {
	return kci.client.Watch(ctx, namespace, r, options...)
}

func (kci *k8sClientImpl) List(ctx context.Context, namespace string, resp k8s.ResourceList, options ...k8s.Option) error {
	return kci.client.List(ctx, namespace, resp, options...)
}

func (kci *k8sClientImpl) Get(ctx context.Context, namespace, name string, resp k8s.Resource, options ...k8s.Option) error {
	return kci.client.Get(ctx, namespace, name, resp, options...)
}

func (kci *k8sClientImpl) Namespace() string {
	return kci.client.Namespace
}

// clusterScanner scans the cluster for endpoints belonging to services that have annotation sourcegraph.prometheus/scrape=true.
// It runs an event loop that reacts to changes to the endpoints set. Everytime there is a change it calls the ScanConsumer.
type clusterScanner struct {
	client  kubernetesClient
	consume ScanConsumer
}

// Starts a cluster scanner with the specified client and consumer. Does not block.
func startClusterScannerWithClient(client kubernetesClient, consumer ScanConsumer) error {
	cs := &clusterScanner{
		client:  client,
		consume: consumer,
	}

	go cs.runEventLoop()
	return nil
}

// Starts a cluster scanner with the specified consumer. Does not block.
func StartClusterScanner(consumer ScanConsumer) error {
	client, err := k8s.NewInClusterClient()
	if err != nil {
		return err
	}

	kci := &k8sClientImpl{client: client}
	return startClusterScannerWithClient(kci, consumer)
}

// Runs the k8s.Watch endpoints event loop, and triggers a rescan of cluster when something changes with endpoints.
// Before spinning in the loop does an initial scan.
func (cs *clusterScanner) runEventLoop() {
	cs.scanCluster()
	for {
		err := cs.watchEndpointEvents()
		log15.Debug("failed to watch kubernetes endpoints", "error", err)
		time.Sleep(time.Second * 5)
	}
}

// watchEndpointEvents uses the k8s watch API operation to watch for endpoint events. Spins forever unless an error
// occurs that would necessitate creating a new watcher. The caller will then call again creating the new watcher.
func (cs *clusterScanner) watchEndpointEvents() error {
	watcher, err := cs.client.Watch(context.Background(), cs.client.Namespace(), new(corev1.Endpoints))
	if err != nil {
		return fmt.Errorf("k8s client.Watch error: %w", err)
	}
	defer watcher.Close()

	for {
		var eps corev1.Endpoints
		eventType, err := watcher.Next(&eps)
		if err != nil {
			// we need a new watcher
			return fmt.Errorf("k8s watcher.Next error: %w", err)
		}

		if eventType == k8s.EventError {
			// we need a new watcher
			return errors.New("error event")
		}

		cs.scanCluster()
	}
}

// scanCluster looks for endpoints belonging to services that have annotation sourcegraph.prometheus/scrape=true.
// It derives the appropriate port from the prometheus.io/port annotation.
func (cs *clusterScanner) scanCluster() {
	var services corev1.ServiceList

	err := cs.client.List(context.Background(), cs.client.Namespace(), &services)
	if err != nil {
		log15.Error("k8s failed to list services", "error", err)
		return
	}

	var scanResults []Endpoint

	for _, svc := range services.Items {
		svcName := *svc.Metadata.Name

		// TODO(uwedeportivo): pgsql doesn't work, figure out why
		if svcName == "pgsql" {
			continue
		}

		if svc.Metadata.Annotations["sourcegraph.prometheus/scrape"] != "true" {
			continue
		}

		var port int
		if portStr := svc.Metadata.Annotations["prometheus.io/port"]; portStr != "" {
			port, err = strconv.Atoi(portStr)
			if err != nil {
				log15.Debug("k8s prometheus.io/port annotation for service is not an integer", "service", svcName, "port", portStr)
				continue
			}
		}

		var endpoints corev1.Endpoints
		err = cs.client.Get(context.Background(), cs.client.Namespace(), svcName, &endpoints)
		if err != nil {
			log15.Error("k8s failed to get endpoints", "error", err)
			return
		}

		for _, subset := range endpoints.Subsets {
			var ports []int
			if port != 0 {
				ports = []int{port}
			} else {
				for _, port := range subset.GetPorts() {
					ports = append(ports, int(port.GetPort()))
				}
			}

			for _, addr := range subset.Addresses {
				for _, port := range ports {
					host := addrToHost(addr, port)
					if host != "" {
						scanResults = append(scanResults, Endpoint{
							Service: svcName,
							Host:    host,
						})
					}
				}
			}
		}
	}

	cs.consume(scanResults)
}

// addrToHost converts a scanned k8s endpoint address structure into a string that is the host:port part of a URL.
func addrToHost(addr *corev1.EndpointAddress, port int) string {
	if addr.Hostname != nil && *addr.Hostname != "" {
		return fmt.Sprintf("%s:%d", *addr.Hostname, port)
	} else if addr.Ip != nil {
		return fmt.Sprintf("%s:%d", *addr.Ip, port)
	}
	return ""
}
