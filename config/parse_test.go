package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setup() func() {
	os.Setenv("SHIPYARD_CONFIG", "/User/yamcha/.shipyard")

	return func() {
		os.Unsetenv("SHIPYARD_CONFIG")
	}
}

func TestSingleKubernetesCluster(t *testing.T) {
	tearDown := setup()
	defer tearDown()

	c, err := ParseFolder("./examples/single-cluster-k8s")

	assert.NoError(t, err)
	assert.NotNil(t, c)

	// validate clusters
	assert.Len(t, c.Clusters, 1)

	c1 := c.Clusters[0]
	assert.Equal(t, "default", c1.Name)
	assert.Equal(t, "1.16.0", c1.Version)
	assert.Equal(t, 3, c1.Nodes)
	assert.Equal(t, "network.k8s", c1.Network)

	// validate networks
	assert.Len(t, c.Networks, 1)

	n1 := c.Networks[0]
	assert.Equal(t, "k8s", n1.Name)
	assert.Equal(t, "10.4.0.0/16", n1.Subnet)

	// validate helm charts
	assert.Len(t, c.HelmCharts, 1)

	h1 := c.HelmCharts[0]
	assert.Equal(t, "cluster.default", h1.Cluster)
	assert.Equal(t, "/User/yamcha/.shipyard/charts/consul", h1.Chart)
	assert.Equal(t, "./consul-values", h1.Values)
	assert.Equal(t, "component=server,app=consul", h1.HealthCheck.Pods[0])
	assert.Equal(t, "component=client,app=consul", h1.HealthCheck.Pods[1])

	// validate ingress
	assert.Len(t, c.Ingresses, 2)

	i1 := c.Ingresses[0]
	assert.Equal(t, "consul", i1.Name)
	assert.Equal(t, 8500, i1.Ports[0].Local)
	assert.Equal(t, 8500, i1.Ports[0].Remote)
	assert.Equal(t, 8500, i1.Ports[0].Host)

	i2 := c.Ingresses[1]
	assert.Equal(t, "web", i2.Name)

	// validate references
	err = ParseReferences(c)
	assert.NoError(t, err)

	assert.Equal(t, n1, c1.networkRef)
	assert.Equal(t, c1, h1.clusterRef)
	assert.Equal(t, i1.targetRef, c1)
	assert.Equal(t, i2.targetRef, c1)
}

func TestMultiCluster(t *testing.T) {
	tearDown := setup()
	defer tearDown()

	c, err := ParseFolder("./examples/multi-cluster")

	assert.NoError(t, err)
	assert.NotNil(t, c)

	// validate clusters
	assert.Len(t, c.Clusters, 2)

	c1 := c.Clusters[0]
	assert.Equal(t, "cloud", c1.Name)
	assert.Equal(t, "1.16.0", c1.Version)
	assert.Equal(t, 1, c1.Nodes)
	assert.Equal(t, "network.k8s", c1.Network)

	// validate containers
	assert.Len(t, c.Containers, 2)

	co1 := c.Containers[0]
	assert.Equal(t, "consul_nomad", co1.Name)
	assert.Equal(t, []string{"consul", "agent", "-config-file=/config/consul.hcl"}, co1.Command)
	assert.Equal(t, "./consul_config", co1.Volumes[0].Source)
	assert.Equal(t, "/config", co1.Volumes[0].Destination)
	assert.Equal(t, "network.nomad", co1.Network)
	assert.Equal(t, "10.6.0.2", co1.IPAddress)

	// validate ingress
	assert.Len(t, c.Ingresses, 6)

	i1 := testFindIngress("consul_nomad", c.Ingresses)
	assert.Equal(t, "consul_nomad", i1.Name)

	// validate references
	err = ParseReferences(c)
	assert.NoError(t, err)

	assert.Equal(t, co1, i1.targetRef)
}

func TestCorrectlyOrdersElements(t *testing.T) {
	n1 := &Network{Name: "network1"}
	c1 := &Container{Name: "container1", networkRef: n1}

	c := &Config{}
	c.Containers = []*Container{c1}
	c.Networks = []*Network{n1}

	// process the config
	oc := generateOrder(c)

	// first element should be a network
	assert.Len(t, oc, 2)

	el1, ok := oc[0].(*Network)
	assert.True(t, ok)
	assert.Equal(t, "network1", el1.Name)

	co1, ok := oc[1].(*Container)
	assert.True(t, ok)
	assert.Equal(t, "container1", co1.Name)
}

func testFindIngress(name string, ingress []*Ingress) *Ingress {
	for _, i := range ingress {
		if i.Name == name {
			return i
		}
	}

	return nil
}