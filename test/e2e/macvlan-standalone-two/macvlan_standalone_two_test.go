package macvlan_standalone_two_test

import (
	"context"
	"errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/cni-plugins/test/e2e/common"
	apitypes "k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("MacvlanStandaloneTwo", Label("standalone", "two-interface"), func() {

	It("spiderdoctor connectivity should be succeed", Label("spiderdoctor"), func() {
		// create task spiderdoctor crd
		task.Name = name
		// schedule
		plan.StartAfterMinute = 0
		plan.RoundNumber = 2
		plan.IntervalMinute = 2
		plan.TimeoutMinute = 2
		task.Spec.Schedule = plan
		// target
		targetAgent.TestIngress = true
		targetAgent.TestEndpoint = true
		targetAgent.TestClusterIp = true
		targetAgent.TestMultusInterface = true
		targetAgent.TestNodePort = true
		targetAgent.TestIPv4 = &common.IPV4
		targetAgent.TestIPv6 = &common.IPV6

		target.TargetAgent = targetAgent
		task.Spec.Target = target
		// request
		request.DurationInSecond = 2
		request.QPS = 1
		request.PerRequestTimeoutInSecond = 10

		task.Spec.Request = request
		// success condition

		condition.SuccessRate = &successRate
		condition.MeanAccessDelayInMs = &delayMs

		task.Spec.SuccessCondition = condition
		GinkgoWriter.Printf("spiderdoctor task: %+v", task)
		err := frame.CreateResource(task)
		Expect(err).NotTo(HaveOccurred(), " spiderdoctor nethttp crd create failed")

		err = frame.GetResource(apitypes.NamespacedName{Name: name}, task)
		Expect(err).NotTo(HaveOccurred(), " spiderdoctor nethttp crd get failed")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60*5)
		defer cancel()
		for run {
			select {
			case <-ctx.Done():
				run = false
				Expect(errors.New("wait nethttp test timeout")).NotTo(HaveOccurred(), " running spiderdoctor task timeout")
			default:
				err = frame.GetResource(apitypes.NamespacedName{Name: name}, task)
				Expect(err).NotTo(HaveOccurred(), " spiderdoctor nethttp crd get failed")
				if task.Status.Finish == true {
					for _, v := range task.Status.History {
						Expect(v.Status).To(Equal("succeed"), "round %d failed", v.RoundNumber)
					}
					run = false
				}
			}
			time.Sleep(time.Second * 5)
		}
	})
})
