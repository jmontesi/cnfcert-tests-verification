package operator

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/redhat-best-practices-for-k8s/certsuite-qe/tests/globalhelper"
	"github.com/redhat-best-practices-for-k8s/certsuite-qe/tests/globalparameters"
	tshelper "github.com/redhat-best-practices-for-k8s/certsuite-qe/tests/operator/helper"
	tsparams "github.com/redhat-best-practices-for-k8s/certsuite-qe/tests/operator/parameters"
)

var _ = Describe("Operator pods automount token", func() {
	var randomNamespace string
	var randomReportDir string
	var randomCertsuiteConfigDir string

	BeforeEach(func() {
		// Create random namespace and keep original report and certsuite config directories
		randomNamespace, randomReportDir, randomCertsuiteConfigDir =
			globalhelper.BeforeEachSetupWithRandomNamespace(
				tsparams.OperatorNamespace)

		By("Define certsuite config file")
		err := globalhelper.DefineCertsuiteConfig(
			[]string{randomNamespace},
			[]string{tsparams.TestPodLabel},
			[]string{tsparams.CertsuiteTargetOperatorLabels},
			[]string{},
			tsparams.CertsuiteTargetCrdFilters, randomCertsuiteConfigDir)
		Expect(err).ToNot(HaveOccurred())

		By("Deploy operator group")
		err = tshelper.DeployTestOperatorGroup(randomNamespace, false)
		Expect(err).ToNot(HaveOccurred(), "Error deploying operator group")
	})

	AfterEach(func() {
		globalhelper.AfterEachCleanupWithRandomNamespace(randomNamespace,
			randomReportDir, randomCertsuiteConfigDir, tsparams.Timeout)
	})

	It("Operator pods should not have automount token [negative]", func() {
		// Deploy an operator that does not have automount token
		// Note: The service account that gets deployed as part of the nginx operator
		// contains a service account that leaves the SA default/nil and that defaults to true.
		// The SA should contain a automountServiceAccountToken field that is set explicitly to false.
		By("Deploy operator group")
		err := tshelper.DeployTestOperatorGroup(randomNamespace, false)
		Expect(err).ToNot(HaveOccurred(), "Error deploying operator group")

		By("Query the packagemanifest for the " + tsparams.CertifiedOperatorPrefixNginx)
		version, err := globalhelper.QueryPackageManifestForVersion(tsparams.CertifiedOperatorPrefixNginx, randomNamespace)
		Expect(err).ToNot(HaveOccurred(), "Error querying package manifest for nginx-ingress-operator")

		By(fmt.Sprintf("Deploy nginx-ingress-operator%s for testing", "."+version))
		// nginx-ingress-operator: in certified-operators group and version is certified
		err = tshelper.DeployOperatorSubscription(
			tsparams.CertifiedOperatorPrefixNginx,
			"alpha",
			randomNamespace,
			tsparams.CertifiedOperatorGroup,
			tsparams.OperatorSourceNamespace,
			tsparams.CertifiedOperatorPrefixNginx+".v"+version,
			v1alpha1.ApprovalAutomatic,
		)
		Expect(err).ToNot(HaveOccurred(), ErrorDeployOperatorStr+
			tsparams.CertifiedOperatorPrefixNginx)

		err = waitUntilOperatorIsReady(tsparams.CertifiedOperatorPrefixNginx,
			randomNamespace)
		Expect(err).ToNot(HaveOccurred(), "Operator "+tsparams.CertifiedOperatorPrefixNginx+".v"+version+
			" is not ready")

		By("Label operator")
		Eventually(func() error {
			return tshelper.AddLabelToInstalledCSV(
				tsparams.CertifiedOperatorPrefixNginx,
				randomNamespace,
				tsparams.OperatorLabel)
		}, tsparams.TimeoutLabelCsv, tsparams.PollingInterval).Should(Not(HaveOccurred()),
			ErrorLabelingOperatorStr+tsparams.CertifiedOperatorPrefixNginx)

		By("Assert that the manager pod has automount token nil or false")
		controllerPod, err := globalhelper.GetControllerPodFromOperator(randomNamespace, tsparams.CertifiedOperatorPrefixNginx)
		Expect(err).ToNot(HaveOccurred(), "Error getting controller pod")

		By(fmt.Sprintf("Checking if pod %s has automount token nil or false", controllerPod.Name))
		if controllerPod.Spec.AutomountServiceAccountToken != nil {
			Expect(*controllerPod.Spec.AutomountServiceAccountToken).To(BeFalse())
		} else {
			Expect(controllerPod.Spec.AutomountServiceAccountToken).To(BeNil())
		}

		By("Start test")
		err = globalhelper.LaunchTests(
			tsparams.CertsuiteOperatorPodAutomountToken,
			globalhelper.ConvertSpecNameToFileName(CurrentSpecReport().FullText()), randomReportDir, randomCertsuiteConfigDir)
		Expect(err).ToNot(HaveOccurred())

		By("Verify test case status in Claim report")
		err = globalhelper.ValidateIfReportsAreValid(
			tsparams.CertsuiteOperatorPodAutomountToken,
			globalparameters.TestCaseFailed, randomReportDir)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Operator pods have automount token [negative]", func() {
		// Deploy an operator that explicitly has automount token
		// TODO: Find an operator that has automount token set explicitly
	})
})
