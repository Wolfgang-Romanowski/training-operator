// Copyright 2025 The Kubeflow Authors
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

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// TelemetryConfig holds all telemetry configuration
type TelemetryConfig struct {
	// Core settings
	Enabled          bool
	AsyncProcessing  bool
	ProcessingTimeout time.Duration
	Endpoint         string  // Telemetry endpoint URL
	Token            string  // Authentication token
	ClusterID        string  // Cluster identifier
	
	// Metric settings
	MaxCardinality   int
	CleanupInterval  time.Duration
	MaxEntryAge      time.Duration
	MaxEntries       int
	
	// Reserved for future use
}

var (
	config     *TelemetryConfig
	configOnce sync.Once
)

// Initialize sets up the telemetry configuration from environment variables
func Initialize() {
	configOnce.Do(func() {
		config = &TelemetryConfig{
			// Defaults
			Enabled:           true,
			AsyncProcessing:   true,
			ProcessingTimeout: 5 * time.Second,
			MaxCardinality:    10, // Red Hat limit per RHOAISTRAT-575
			CleanupInterval:   1 * time.Hour,
			MaxEntryAge:       24 * time.Hour,
			MaxEntries:        10000,
		}
		
		loadFromEnvironment()
		validateConfig()
		logConfig()
	})
}

// loadFromEnvironment loads configuration from environment variables
func loadFromEnvironment() {
	// Core settings
	if enabled := os.Getenv("TELEMETRY_ENABLED"); enabled != "" {
		config.Enabled = strings.ToLower(enabled) != "false"
	}
	
	if async := os.Getenv("TELEMETRY_ASYNC"); async != "" {
		config.AsyncProcessing = strings.ToLower(async) != "false"
	}
	
	if timeout := os.Getenv("TELEMETRY_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.ProcessingTimeout = d
		}
	}
	
	// Metric settings
	if cardinality := os.Getenv("TELEMETRY_MAX_CARDINALITY"); cardinality != "" {
		if c, err := strconv.Atoi(cardinality); err == nil && c > 0 {
			config.MaxCardinality = c
		}
	}
	
	if cleanup := os.Getenv("TELEMETRY_CLEANUP_INTERVAL"); cleanup != "" {
		if d, err := time.ParseDuration(cleanup); err == nil {
			config.CleanupInterval = d
		}
	}
	
	if maxAge := os.Getenv("TELEMETRY_MAX_ENTRY_AGE"); maxAge != "" {
		if d, err := time.ParseDuration(maxAge); err == nil {
			config.MaxEntryAge = d
		}
	}
	
	if maxEntries := os.Getenv("TELEMETRY_MAX_ENTRIES"); maxEntries != "" {
		if e, err := strconv.Atoi(maxEntries); err == nil && e > 0 {
			config.MaxEntries = e
		}
	}
}

// validateConfig ensures configuration values are reasonable
func validateConfig() {
	// Ensure reasonable timeouts
	if config.ProcessingTimeout < 1*time.Second {
		config.ProcessingTimeout = 1 * time.Second
	}
	if config.ProcessingTimeout > 30*time.Second {
		config.ProcessingTimeout = 30 * time.Second
	}
	
	// Ensure reasonable cardinality limits
	if config.MaxCardinality < 1 {
		config.MaxCardinality = 1
	}
	if config.MaxCardinality > 10 {
		klog.Warningf("High cardinality limit set: %d (Red Hat REQUIRES max 10 per RHOAISTRAT-575)", config.MaxCardinality)
	}
	
	// Ensure reasonable entry limits
	if config.MaxEntries < 100 {
		config.MaxEntries = 100
	}
	if config.MaxEntries > 100000 {
		config.MaxEntries = 100000
	}
}

// logConfig logs the current configuration (without sensitive data)
func logConfig() {
	klog.Infof("Telemetry configuration: enabled=%v, async=%v, timeout=%v, max_cardinality=%d", 
		config.Enabled, config.AsyncProcessing, config.ProcessingTimeout, config.MaxCardinality)
}

// Get returns the current telemetry configuration
func Get() *TelemetryConfig {
	if config == nil {
		Initialize()
	}
	return config
}

// IsTelemetryConfigEnabled returns whether telemetry is enabled in configuration.
// This only checks configuration, not initialization status.
func IsTelemetryConfigEnabled() bool {
	return Get().Enabled
}

// GetMaxCardinality returns the maximum allowed cardinality
func GetMaxCardinality() int {
	return Get().MaxCardinality
}

// GetProcessingTimeout returns the event processing timeout
func GetProcessingTimeout() time.Duration {
	return Get().ProcessingTimeout
}

// GetCleanupInterval returns the cleanup interval for old entries
func GetCleanupInterval() time.Duration {
	return Get().CleanupInterval
}

// GetMaxEntryAge returns the maximum age for cached entries
func GetMaxEntryAge() time.Duration {
	return Get().MaxEntryAge
}

// GetMaxEntries returns the maximum number of entries to cache
func GetMaxEntries() int {
	return Get().MaxEntries
}

// NOTE: Health monitoring functionality has been moved to health_endpoint.go
// to consolidate all health-related code in one place

// AutoPopulateTelemetrySecretFromCluster attempts to populate telemetry configuration from cluster secrets.
// This function extracts telemetry tokens from the OpenShift pull secret and cluster configuration.
func AutoPopulateTelemetrySecretFromCluster(ctx context.Context, kubeClient kubernetes.Interface, namespace string) error {
	klog.Info("Attempting to auto-populate telemetry configuration from cluster")

	// Try to get existing telemetry secret first
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, "training-operator-telemetry-secret", metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing telemetry secret: %w", err)
	}

	// If secret exists and has valid data, use it
	if err == nil && secret.Data != nil {
		if token, exists := secret.Data["telemetry-token"]; exists && string(token) != "placeholder" {
			config.Token = string(token)
		}
		if endpoint, exists := secret.Data["telemetry-endpoint"]; exists {
			config.Endpoint = string(endpoint)
		}
		if clusterID, exists := secret.Data["cluster-id"]; exists {
			config.ClusterID = string(clusterID)
		}
		
		if config.Token != "" && config.Endpoint != "" {
			klog.Info("Successfully loaded telemetry configuration from existing secret")
			return nil
		}
	}

	// Try to extract from pull secret
	telemetryToken, extractErr := extractTelemetryTokenFromPullSecret(ctx, kubeClient)
	if extractErr != nil {
		klog.Warningf("Could not extract telemetry token from pull secret: %v", extractErr)
		// Not a fatal error, continue with placeholders
		telemetryToken = ""
	}

	// Create or update the telemetry secret
	newSecretData := map[string][]byte{
		"telemetry-token":    []byte(telemetryToken),
		"telemetry-endpoint": []byte("https://infogw.api.openshift.com/metrics/v1/receive"),
		"cluster-id":         []byte(config.ClusterID),
	}

	if telemetryToken == "" {
		// Use placeholders if we couldn't get real values
		newSecretData["telemetry-token"] = []byte("placeholder")
		klog.Warning("Using placeholder values for telemetry secret - telemetry will not function until properly configured")
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "training-operator-telemetry-secret",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "training-operator",
				"app.kubernetes.io/component":  "telemetry",
				"app.kubernetes.io/managed-by": "training-operator",
			},
		},
		Data: newSecretData,
	}

	if err == nil {
		// Update existing secret
		newSecret.ResourceVersion = secret.ResourceVersion
		_, updateErr := kubeClient.CoreV1().Secrets(namespace).Update(ctx, newSecret, metav1.UpdateOptions{})
		if updateErr != nil {
			return fmt.Errorf("failed to update telemetry secret: %w", updateErr)
		}
		klog.Info("Updated telemetry secret")
	} else {
		// Create new secret
		_, createErr := kubeClient.CoreV1().Secrets(namespace).Create(ctx, newSecret, metav1.CreateOptions{})
		if createErr != nil && !errors.IsAlreadyExists(createErr) {
			return fmt.Errorf("failed to create telemetry secret: %w", createErr)
		}
		klog.Info("Created telemetry secret")
	}

	// Update config with the values
	if telemetryToken != "" {
		config.Token = telemetryToken
		config.Endpoint = "https://infogw.api.openshift.com/metrics/v1/receive"
	}

	return nil
}

// extractTelemetryTokenFromPullSecret extracts the telemetry token from the OpenShift pull secret.
func extractTelemetryTokenFromPullSecret(ctx context.Context, kubeClient kubernetes.Interface) (string, error) {
	// Get the pull secret from openshift-config namespace
	pullSecret, err := kubeClient.CoreV1().Secrets("openshift-config").Get(ctx, "pull-secret", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pull secret: %w", err)
	}

	// Extract .dockerconfigjson data
	dockerConfigJSON, exists := pullSecret.Data[".dockerconfigjson"]
	if !exists {
		return "", fmt.Errorf("pull secret does not contain .dockerconfigjson")
	}

	// Parse the docker config
	var dockerConfig struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}

	if err := json.Unmarshal(dockerConfigJSON, &dockerConfig); err != nil {
		return "", fmt.Errorf("failed to parse docker config: %w", err)
	}

	// Look for cloud.openshift.com auth which contains telemetry token
	if cloudAuth, exists := dockerConfig.Auths["cloud.openshift.com"]; exists && cloudAuth.Auth != "" {
		return cloudAuth.Auth, nil
	}

	// Fallback to registry.redhat.io auth
	if redhatAuth, exists := dockerConfig.Auths["registry.redhat.io"]; exists && redhatAuth.Auth != "" {
		return redhatAuth.Auth, nil
	}

	return "", fmt.Errorf("no telemetry token found in pull secret")
}

// GetTelemetryURL returns the configured telemetry endpoint URL.
// This is the URL where telemetry data will be sent.
func GetTelemetryURL() string {
	if config == nil {
		Initialize()
	}
	if config.Endpoint != "" {
		return config.Endpoint
	}
	return "https://infogw.api.openshift.com/metrics/v1/receive"
}

// IsTelemetrySecretConfigured checks if the telemetry authentication secret is configured.
// Returns true if a valid token has been configured for telemetry export.
func IsTelemetrySecretConfigured() bool {
	if config == nil {
		Initialize()
	}
	return config.Token != ""
}

// GetTelemetryNamespace returns the namespace where telemetry components are deployed.
// This is typically the namespace where the training operator is running.
func GetTelemetryNamespace() string {
	namespace := os.Getenv("TELEMETRY_NAMESPACE")
	if namespace != "" {
		return namespace
	}
	
	// Try to get the namespace from the pod's service account
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return strings.TrimSpace(string(data))
	}
	
	// Default to redhat-ods-applications for RHOAI deployments
	return "redhat-ods-applications"
}