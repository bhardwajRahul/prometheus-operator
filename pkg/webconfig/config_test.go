// Copyright 2021 The prometheus-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package webconfig_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/golden"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/webconfig"
)

func TestCreateOrUpdateWebConfigSecret(t *testing.T) {
	tc := []struct {
		name                string
		webConfigFileFields monitoringv1.WebConfigFileFields
		golden              string
	}{
		{
			name:                "tls config not defined",
			webConfigFileFields: monitoringv1.WebConfigFileFields{},
			golden:              "tls_config_not_defined.golden",
		},
		{
			name: "minimal TLS config with certificate from secret",
			webConfigFileFields: monitoringv1.WebConfigFileFields{
				TLSConfig: &monitoringv1.WebTLSConfig{
					Cert: monitoringv1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "tls.crt",
						},
					},
					KeySecret: v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "tls.key",
					},
				},
			},
			golden: "minimal_TLS_config_with_certificate_from_secret.golden",
		},
		{
			name: "minimal TLS config with certificate from configmap",
			webConfigFileFields: monitoringv1.WebConfigFileFields{
				TLSConfig: &monitoringv1.WebTLSConfig{
					Cert: monitoringv1.SecretOrConfigMap{
						ConfigMap: &v1.ConfigMapKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test-configmap",
							},
							Key: "tls.crt",
						},
					},
					KeySecret: v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "tls.key",
					},
				},
			},
			golden: "minimal_TLS_config_with_certificate_from_configmap.golden",
		},
		{
			name: "minimal TLS config with client CA from configmap",
			webConfigFileFields: monitoringv1.WebConfigFileFields{
				TLSConfig: &monitoringv1.WebTLSConfig{
					Cert: monitoringv1.SecretOrConfigMap{
						ConfigMap: &v1.ConfigMapKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test-configmap",
							},
							Key: "tls.crt",
						},
					},
					KeySecret: v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "tls.key",
					},
					ClientCA: monitoringv1.SecretOrConfigMap{
						ConfigMap: &v1.ConfigMapKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test-configmap",
							},
							Key: "tls.client_ca",
						},
					},
				},
			},
			golden: "minimal_TLS_config_with_client_CA_from_configmap.golden",
		},
		{
			name: "TLS config with all parameters from secrets",
			webConfigFileFields: monitoringv1.WebConfigFileFields{
				TLSConfig: &monitoringv1.WebTLSConfig{
					ClientCA: monitoringv1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "tls.ca",
						},
					},
					Cert: monitoringv1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "test-secret",
							},
							Key: "tls.crt",
						},
					},
					KeySecret: v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "tls.keySecret",
					},
					ClientAuthType:           ptr.To("RequireAnyClientCert"),
					MinVersion:               ptr.To("TLS11"),
					MaxVersion:               ptr.To("TLS13"),
					CipherSuites:             []string{"cipher-1", "cipher-2"},
					PreferServerCipherSuites: ptr.To(false),
					CurvePreferences:         []string{"curve-1", "curve-2"},
				},
			},
			golden: "TLS_config_with_all_parameters_from_secrets.golden",
		},
		{
			name: "TLS config with client CA, cert and key files",
			webConfigFileFields: monitoringv1.WebConfigFileFields{
				TLSConfig: &monitoringv1.WebTLSConfig{
					ClientCAFile: ptr.To("/etc/ssl/certs/tls.client_ca"),
					CertFile:     ptr.To("/etc/ssl/certs/tls.crt"),
					KeyFile:      ptr.To("/etc/ssl/secrets/tls.key"),
				},
			},
			golden: "TLS_config_with_client_CA_cert_and_key_files.golden",
		},
		{
			name: "HTTP config with all parameters",
			webConfigFileFields: monitoringv1.WebConfigFileFields{
				HTTPConfig: &monitoringv1.WebHTTPConfig{
					HTTP2: ptr.To(false),
					Headers: &monitoringv1.WebHTTPHeaders{
						ContentSecurityPolicy:   "test",
						StrictTransportSecurity: "test",
						XContentTypeOptions:     "NoSniff",
						XFrameOptions:           "SameOrigin",
						XXSSProtection:          "test",
					},
				},
			},
			golden: "HTTP_config_with_all_parameters.golden",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			secretName := "test-secret"
			config, err := webconfig.New("/web_certs_path_prefix", secretName, tt.webConfigFileFields)
			require.NoError(t, err)

			var (
				s            = v1.Secret{}
				secretClient = fake.NewSimpleClientset().CoreV1().Secrets("default")
			)
			err = config.CreateOrUpdateWebConfigSecret(context.Background(), secretClient, &s)
			require.NoError(t, err)

			secret, err := secretClient.Get(context.Background(), secretName, metav1.GetOptions{})
			require.NoError(t, err)

			golden.Assert(t, string(secret.Data["web-config.yaml"]), tt.golden)
		})
	}
}

func TestGetMountParameters(t *testing.T) {
	ts := []struct {
		webConfigFileFields monitoringv1.WebConfigFileFields
		expectedVolumes     []v1.Volume
		expectedMounts      []v1.VolumeMount
	}{
		{
			webConfigFileFields: monitoringv1.WebConfigFileFields{},
			expectedVolumes: []v1.Volume{
				{
					Name: "web-config",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "web-config",
						},
					},
				},
			},
			expectedMounts: []v1.VolumeMount{
				{
					Name:             "web-config",
					ReadOnly:         true,
					MountPath:        "/etc/prometheus/web_config/web-config.yaml",
					SubPath:          "web-config.yaml",
					MountPropagation: nil,
					SubPathExpr:      "",
				},
			},
		},
		{
			webConfigFileFields: monitoringv1.WebConfigFileFields{
				TLSConfig: &monitoringv1.WebTLSConfig{
					KeySecret: v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "some-secret",
						},
						Key: "tls.key",
					},
					Cert: monitoringv1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "some-secret",
							},
							Key: "tls.crt",
						},
					},
					ClientCA: monitoringv1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "some-secret",
							},
							Key: "tls.client_ca",
						},
					},
				},
			},
			expectedVolumes: []v1.Volume{
				{
					Name: "web-config",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "web-config",
						},
					},
				},
				{
					Name: "web-config-tls-secret-key-some-secret-3556f148",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "some-secret",
						},
					},
				},
				{
					Name: "web-config-tls-secret-cert-some-secret-3556f148",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "some-secret",
						},
					},
				},
				{
					Name: "web-config-tls-secret-client-ca-some-secret-3556f148",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "some-secret",
						},
					},
				},
			},
			expectedMounts: []v1.VolumeMount{
				{
					Name:             "web-config",
					ReadOnly:         true,
					MountPath:        "/etc/prometheus/web_config/web-config.yaml",
					SubPath:          "web-config.yaml",
					MountPropagation: nil,
					SubPathExpr:      "",
				},
				{
					Name:             "web-config-tls-secret-key-some-secret-3556f148",
					ReadOnly:         true,
					MountPath:        "/etc/prometheus/web_config/secret/some-secret-key",
					MountPropagation: nil,
					SubPathExpr:      "",
				},
				{
					Name:             "web-config-tls-secret-cert-some-secret-3556f148",
					ReadOnly:         true,
					MountPath:        "/etc/prometheus/web_config/secret/some-secret-cert",
					MountPropagation: nil,
					SubPathExpr:      "",
				},
				{
					Name:             "web-config-tls-secret-client-ca-some-secret-3556f148",
					ReadOnly:         true,
					MountPath:        "/etc/prometheus/web_config/secret/some-secret-ca",
					MountPropagation: nil,
					SubPathExpr:      "",
				},
			},
		},
	}

	for _, tt := range ts {
		t.Run("", func(t *testing.T) {
			tlsAssets, err := webconfig.New("/etc/prometheus/web_config", "web-config", tt.webConfigFileFields)
			require.NoError(t, err)

			_, volumes, mounts, err := tlsAssets.GetMountParameters()
			require.NoError(t, err)

			require.Equal(t, tt.expectedVolumes, volumes)
			require.Equal(t, tt.expectedMounts, mounts)
		})
	}
}
