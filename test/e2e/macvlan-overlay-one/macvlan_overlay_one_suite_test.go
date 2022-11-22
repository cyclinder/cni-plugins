package macvlan_overlay_one_test

import (
	"context"
	"fmt"
	multus_v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/pkg/schema"
	"github.com/spidernet-io/cni-plugins/test/e2e/common"
	e2e "github.com/spidernet-io/e2eframework/framework"
	"github.com/spidernet-io/e2eframework/tools"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

func TestMacvlanOverlayOne(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MacvlanOverlayOne Suite")
}

var frame *e2e.Framework
var name, namespace, multuNs string

var podList *corev1.PodList
var dp *appsv1.Deployment
var labels, annotations = make(map[string]string), make(map[string]string)

var port int32 = 80
var nodePorts []int32
var podIPs, clusterIPs, nodeIPs []string

var retryTimes = 5

var _ = BeforeSuite(func() {
	defer GinkgoRecover()
	var e error
	frame, e = e2e.NewFramework(GinkgoT(), []func(*runtime.Scheme) error{multus_v1.AddToScheme, schema.SpiderPoolAddToScheme})
	Expect(e).NotTo(HaveOccurred())

	// init namespace name and create
	name = "one-macvlan-overlay"
	namespace = "ns" + tools.RandomName()
	multuNs = "kube-system"
	labels["app"] = name

	err := frame.CreateNamespace(namespace)
	Expect(err).NotTo(HaveOccurred(), "failed to create namespace %v", namespace)
	GinkgoWriter.Printf("create namespace %v \n", namespace)

	// get macvlan-standalone multus crd instance by name
	multusInstance, err := frame.GetMultusInstance(common.MacvlanOverlayVlan100Name, multuNs)
	Expect(err).NotTo(HaveOccurred())
	Expect(multusInstance).NotTo(BeNil())
	annotations[common.MultusAddonAnnotation_Key] = fmt.Sprintf("%s/%s", multuNs, common.MacvlanOverlayVlan100Name)
	annotations[common.SpiderPoolIPPoolAnnotationKey] = `{"interface": "net1", "ipv4": ["vlan100-v4"], "ipv6": ["vlan100-v6"] }`

	GinkgoWriter.Printf("create deploy: %v/%v \n", namespace, name)
	dp = common.GenerateDeploymentYaml(name, namespace, labels, annotations)
	Expect(frame.CreateDeployment(dp)).NotTo(HaveOccurred())

	ctx, cancel := context.WithTimeout(context.Background(), 2*common.CtxTimeout)
	defer cancel()

	err = frame.WaitPodListRunning(dp.Spec.Selector.MatchLabels, int(*dp.Spec.Replicas), ctx)
	Expect(err).NotTo(HaveOccurred())

	podList, err = frame.GetPodList([]client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(labels),
	}...)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(podList.Items)).To(BeEquivalentTo(int(*dp.Spec.Replicas)))

	// create nodePort service
	st := common.GenerateServiceYaml(name, namespace, port, dp.Spec.Selector.MatchLabels)
	err = frame.CreateService(st)
	Expect(err).NotTo(HaveOccurred())

	GinkgoWriter.Printf("succeed to create nodePort service: %s/%s\n", namespace, name)

	// get clusterIPs & nodePorts
	service, err := frame.GetService(name, namespace)
	Expect(err).NotTo(HaveOccurred())
	Expect(service).NotTo(BeNil(), "failed to get service: %s/%s", namespace, name)

	for _, ip := range service.Spec.ClusterIPs {
		if net.ParseIP(ip).To4() == nil && common.IPV6 {
			clusterIPs = append(clusterIPs, ip)
		}

		if net.ParseIP(ip).To4() != nil && common.IPV4 {
			clusterIPs = append(clusterIPs, ip)
		}
	}
	nodePorts = common.GetServiceNodePorts(service.Spec.Ports)
	GinkgoWriter.Printf("clusterIPs: %v\n", clusterIPs)

	// check service ready by get endpoint
	err = common.WaitEndpointReady(retryTimes, name, namespace, frame)
	Expect(err).NotTo(HaveOccurred())

	// get pod all ip
	podIPs, err = common.GetAllIPsFromPods(podList)
	Expect(err).NotTo(HaveOccurred())
	Expect(podIPs).NotTo(BeNil())
	GinkgoWriter.Printf("Get All PodIPs: %v\n", podIPs)

	nodeIPs, err = common.GetKindNodeIPs(context.TODO(), frame, frame.Info.KindNodeList)
	Expect(err).NotTo(HaveOccurred(), "failed to get all node ips: %v", err)
	Expect(nodeIPs).NotTo(BeNil())

	GinkgoWriter.Printf("Get All Node ips: %v\n", nodeIPs)
	GinkgoWriter.Printf("Node list : %v \n", frame.Info.KindNodeList)

	time.Sleep(10 * time.Second)
})

var _ = AfterSuite(func() {
	// delete deployment
	//err := frame.DeleteDeployment(name, namespace)
	//Expect(err).NotTo(HaveOccurred(), "failed to delete deployment %v/%v", namespace, name)
	//
	//// delete service
	//err = frame.DeleteService(name, namespace)
	//Expect(err).To(Succeed())
	//
	//err = frame.DeleteNamespace(namespace)
	//Expect(err).NotTo(HaveOccurred(), "failed to delete namespace %v", namespace)
})
