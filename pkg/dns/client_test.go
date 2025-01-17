package dns

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/mesh/v2/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckDNSProvider(t *testing.T) {
	tests := []struct {
		desc        string
		mockFile    string
		expProvider Provider
		expErr      bool
	}{
		{
			desc:        "CoreDNS supported version",
			mockFile:    "checkdnsprovider_supported_version.yaml",
			expProvider: CoreDNS,
			expErr:      false,
		},
		{
			desc:        "CoreDNS supported version with suffix",
			mockFile:    "checkdnsprovider_supported_version_suffix.yaml",
			expProvider: CoreDNS,
			expErr:      false,
		},
		{
			desc:        "KubeDNS",
			mockFile:    "checkdnsprovider_kubedns.yaml",
			expProvider: KubeDNS,
			expErr:      false,
		},
		{
			desc:        "CoreDNS unsupported version",
			mockFile:    "checkdnsprovider_unsupported_version.yaml",
			expProvider: UnknownDNS,
			expErr:      true,
		},
		{
			desc:        "No known DNS provider",
			mockFile:    "checkdnsprovider_no_provider.yaml",
			expProvider: UnknownDNS,
			expErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			k8sClient := k8s.NewClientMock(test.mockFile)

			logger := logrus.New()

			logger.SetOutput(os.Stdout)
			logger.SetLevel(logrus.DebugLevel)

			client := NewClient(logger, k8sClient.KubernetesClient())

			provider, err := client.CheckDNSProvider(ctx)
			if test.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expProvider, provider)
		})
	}
}

func TestConfigureCoreDNS(t *testing.T) {
	tests := []struct {
		desc        string
		mockFile    string
		expCorefile string
		expCustoms  map[string]string
		expErr      bool
		expRestart  bool
	}{
		{
			desc:        "First time config of CoreDNS",
			mockFile:    "configurecoredns_not_patched.yaml",
			expErr:      false,
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n\n#### Begin Traefik Mesh Block\ntraefik.mesh:53 {\n    errors\n    cache 30\n    forward . 10.10.10.10:53\n}\n#### End Traefik Mesh Block\n",
			expRestart:  true,
		},
		{
			desc:        "Already patched CoreDNS config",
			mockFile:    "configurecoredns_already_patched.yaml",
			expErr:      false,
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n\n#### Begin Traefik Mesh Block\ntraefik.mesh:53 {\n    errors\n    cache 30\n    forward . 10.10.10.10:53\n}\n#### End Traefik Mesh Block\n",
			expRestart:  false,
		},
		{
			desc:       "Missing Corefile configmap",
			mockFile:   "configurecoredns_missing_configmap.yaml",
			expErr:     true,
			expRestart: false,
		},
		{
			desc:        "First time config of CoreDNS custom",
			mockFile:    "configurecoredns_custom_not_patched.yaml",
			expErr:      false,
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n",
			expCustoms: map[string]string{
				"traefik.mesh.server": "\n#### Begin Traefik Mesh Block\ntraefik.mesh:53 {\n    errors\n    cache 30\n    forward . 10.10.10.10:53\n}\n#### End Traefik Mesh Block\n",
			},
			expRestart: true,
		},
		{
			desc:        "Already patched CoreDNS custom config",
			mockFile:    "configurecoredns_custom_already_patched.yaml",
			expErr:      false,
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n",
			expCustoms: map[string]string{
				"traefik.mesh.server": "#### Begin Traefik Mesh Block\ntraefik.mesh:53 {\n    errors\n    cache 30\n    forward . 10.10.10.10:53\n}\n#### End Traefik Mesh Block\n",
			},
			expRestart: false,
		},
		{
			desc:        "Config of CoreDNS 1.3",
			mockFile:    "configurecoredns_1_3.yaml",
			expErr:      false,
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    proxy . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n\n#### Begin Traefik Mesh Block\ntraefik.mesh:53 {\n    errors\n    cache 30\n    proxy . 10.10.10.10:53\n}\n#### End Traefik Mesh Block\n",
			expRestart:  true,
		},
		{
			desc:        "CoreDNS 1.4 already patched for an older version of CoreDNS",
			mockFile:    "configurecoredns_1_4_already_patched.yaml",
			expErr:      false,
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n\n\n#### Begin Traefik Mesh Block\ntraefik.mesh:53 {\n    errors\n    cache 30\n    forward . 10.10.10.10:53\n}\n#### End Traefik Mesh Block\n",
			expRestart:  true,
		},
		{
			desc:       "Missing CoreDNS deployment",
			mockFile:   "configurecoredns_missing_deployment.yaml",
			expErr:     true,
			expRestart: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			k8sClient := k8s.NewClientMock(test.mockFile)

			logger := logrus.New()

			logger.SetOutput(os.Stdout)
			logger.SetLevel(logrus.DebugLevel)

			client := NewClient(logger, k8sClient.KubernetesClient())

			err := client.ConfigureCoreDNS(ctx, "traefik-mesh", "traefik-mesh-dns", 53)
			if test.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			cfgMap, err := k8sClient.KubernetesClient().CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns", metav1.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, test.expCorefile, cfgMap.Data["Corefile"])

			if len(test.expCustoms) > 0 {
				var customCfgMap *corev1.ConfigMap

				customCfgMap, err = k8sClient.KubernetesClient().CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns-custom", metav1.GetOptions{})
				require.NoError(t, err)

				for key, value := range test.expCustoms {
					assert.Equal(t, value, customCfgMap.Data[key])
				}
			}

			coreDNSDeployment, err := k8sClient.KubernetesClient().AppsV1().Deployments("kube-system").Get(ctx, "coredns", metav1.GetOptions{})
			require.NoError(t, err)

			restarted := coreDNSDeployment.Spec.Template.Annotations["traefik-mesh-hash"] != ""
			assert.Equal(t, test.expRestart, restarted)
		})
	}
}

func TestConfigureKubeDNS(t *testing.T) {
	tests := []struct {
		desc           string
		mockFile       string
		expStubDomains string
		expErr         bool
	}{
		{
			desc:     "should return an error if kube-dns deployment does not exist",
			mockFile: "configurekubedns_missing_deployment.yaml",
			expErr:   true,
		},
		{
			desc:           "should add stubdomains config in kube-dns configmap",
			mockFile:       "configurekubedns_not_patched.yaml",
			expStubDomains: `{"traefik.mesh":["10.10.10.10:53"]}`,
		},
		{
			desc:           "should replace stubdomains config in kube-dns configmap",
			mockFile:       "configurekubedns_already_patched.yaml",
			expStubDomains: `{"traefik.mesh":["10.10.10.10:53"]}`,
		},
		{
			desc:           "should create optional kube-dns configmap and add stubdomains config",
			mockFile:       "configurekubedns_optional_configmap.yaml",
			expStubDomains: `{"traefik.mesh":["10.10.10.10:53"]}`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			k8sClient := k8s.NewClientMock(test.mockFile)

			logger := logrus.New()

			logger.SetOutput(os.Stdout)
			logger.SetLevel(logrus.DebugLevel)

			client := NewClient(logger, k8sClient.KubernetesClient())

			err := client.ConfigureKubeDNS(ctx, "traefik-mesh", "traefik-mesh-dns", 53)
			if test.expErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			cfgMap, err := k8sClient.KubernetesClient().CoreV1().ConfigMaps("kube-system").Get(ctx, "kube-dns", metav1.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, test.expStubDomains, cfgMap.Data["stubDomains"])
		})
	}
}

func TestRestoreCoreDNS(t *testing.T) {
	tests := []struct {
		desc        string
		mockFile    string
		hasCustom   bool
		expCorefile string
	}{
		{
			desc:        "CoreDNS config patched",
			mockFile:    "restorecoredns_patched.yaml",
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n\n# This is test data that must be present\n",
		},
		{
			desc:        "CoreDNS config not patched",
			mockFile:    "restorecoredns_not_patched.yaml",
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n",
		},
		{
			desc:        "CoreDNS custom config patched",
			mockFile:    "restorecoredns_custom_patched.yaml",
			hasCustom:   true,
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n",
		},
		{
			desc:        "CoreDNS custom config not patched",
			mockFile:    "restorecoredns_custom_not_patched.yaml",
			hasCustom:   true,
			expCorefile: ".:53 {\n    errors\n    health {\n        lameduck 5s\n    }\n    ready\n    kubernetes {{ pillar['dns_domain'] }} in-addr.arpa ip6.arpa {\n        pods insecure\n        fallthrough in-addr.arpa ip6.arpa\n        ttl 30\n    }\n    prometheus :9153\n    forward . /etc/resolv.conf\n    cache 30\n    loop\n    reload\n    loadbalance\n}\n",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			k8sClient := k8s.NewClientMock(test.mockFile)

			logger := logrus.New()

			logger.SetOutput(os.Stdout)
			logger.SetLevel(logrus.DebugLevel)

			client := NewClient(logger, k8sClient.KubernetesClient())

			err := client.RestoreCoreDNS(ctx)
			require.NoError(t, err)

			cfgMap, err := k8sClient.KubernetesClient().CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns", metav1.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, test.expCorefile, cfgMap.Data["Corefile"])

			if test.hasCustom {
				customCfgMap, err := k8sClient.KubernetesClient().CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns-custom", metav1.GetOptions{})
				require.NoError(t, err)

				_, exists := customCfgMap.Data["traefik.mesh.server"]
				assert.False(t, exists)

				_, exists = customCfgMap.Data["test.server"]
				assert.True(t, exists)
			}
		})
	}
}

func TestRestoreKubeDNS(t *testing.T) {
	tests := []struct {
		desc           string
		mockFile       string
		expStubDomains string
	}{
		{
			desc:           "Not patched",
			mockFile:       "restorekubedns_not_patched.yaml",
			expStubDomains: "",
		},
		{
			desc:           "Already patched",
			mockFile:       "restorekubedns_already_patched.yaml",
			expStubDomains: `{"test":["5.6.7.8"]}`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			k8sClient := k8s.NewClientMock(test.mockFile)

			logger := logrus.New()

			logger.SetOutput(os.Stdout)
			logger.SetLevel(logrus.DebugLevel)

			client := NewClient(logger, k8sClient.KubernetesClient())

			err := client.RestoreKubeDNS(ctx)
			require.NoError(t, err)

			cfgMap, err := k8sClient.KubernetesClient().CoreV1().ConfigMaps("kube-system").Get(ctx, "kube-dns", metav1.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, test.expStubDomains, cfgMap.Data["stubDomains"])
		})
	}
}
