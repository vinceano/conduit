package k8s

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	healthcheckPb "github.com/runconduit/conduit/controller/gen/common/healthcheck"
	"github.com/runconduit/conduit/pkg/healthcheck"
	"github.com/runconduit/conduit/pkg/shell"
	"k8s.io/client-go/rest"
)

const (
	kubernetesConfigFilePathEnvVariable = "KUBECONFIG"
	KubeapiSubsystemName                = "kubernetes-api"
	KubeapiClientCheckDescription       = "can initialize the client"
	KubeapiAccessCheckDescription       = "can query the Kubernetes API"
)

type KubernetesApi interface {
	UrlFor(namespace string, extraPathStartingWithSlash string) (*url.URL, error)
	NewClient() (*http.Client, error)
	healthcheck.StatusChecker
}

type kubernetesApi struct {
	*rest.Config
}

func (kubeapi *kubernetesApi) NewClient() (*http.Client, error) {
	secureTransport, err := rest.TransportFor(kubeapi.Config)
	if err != nil {
		return nil, fmt.Errorf("error instantiating Kubernetes API client: %v", err)
	}

	return &http.Client{
		Transport: secureTransport,
	}, nil
}

func (kubeapi *kubernetesApi) SelfCheck() []*healthcheckPb.CheckResult {
	apiConnectivityCheck, client := kubeapi.checkApiConnectivity()
	apiAccessCheck := kubeapi.checkApiAccess(client)
	return []*healthcheckPb.CheckResult{apiConnectivityCheck, apiAccessCheck}
}

func (kubeapi *kubernetesApi) checkApiConnectivity() (*healthcheckPb.CheckResult, *http.Client) {
	checkResult := &healthcheckPb.CheckResult{
		Status:           healthcheckPb.CheckStatus_OK,
		SubsystemName:    KubeapiSubsystemName,
		CheckDescription: KubeapiClientCheckDescription,
	}

	client, err := kubeapi.NewClient()
	if err != nil {
		checkResult.Status = healthcheckPb.CheckStatus_ERROR
		checkResult.FriendlyMessageToUser = fmt.Sprintf("Error connecting to the API. Error message is [%s]", err.Error())
		return checkResult, client
	}

	return checkResult, client
}

func (kubeapi *kubernetesApi) checkApiAccess(client *http.Client) *healthcheckPb.CheckResult {
	checkResult := &healthcheckPb.CheckResult{
		Status:           healthcheckPb.CheckStatus_OK,
		SubsystemName:    KubeapiSubsystemName,
		CheckDescription: KubeapiAccessCheckDescription,
	}

	if client == nil {
		checkResult.Status = healthcheckPb.CheckStatus_ERROR
		checkResult.FriendlyMessageToUser = "Error building Kubernetes API client."
		return checkResult
	}

	endpointToCheck, err := generateBaseKubernetesApiUrl(kubeapi.Host)
	if err != nil {
		checkResult.Status = healthcheckPb.CheckStatus_ERROR
		checkResult.FriendlyMessageToUser = fmt.Sprintf("Error querying Kubernetes API. Configured host is [%s], error message is [%s]", kubeapi.Host, err.Error())
		return checkResult
	}

	resp, err := client.Get(endpointToCheck.String())
	if err != nil {
		checkResult.Status = healthcheckPb.CheckStatus_ERROR
		checkResult.FriendlyMessageToUser = fmt.Sprintf("HTTP GET request to endpoint [%s] resulted in error: [%s]", endpointToCheck, err.Error())
		return checkResult
	}

	statusCodeReturnedIsWithinSuccessRange := resp.StatusCode < 400
	if !statusCodeReturnedIsWithinSuccessRange {
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			checkResult.Status = healthcheckPb.CheckStatus_ERROR
			checkResult.FriendlyMessageToUser = fmt.Sprintf("HTTP GET request to endpoint [%s] resulted in invalid response: [%v]", endpointToCheck, resp)
			return checkResult
		}

		body := string(bytes)
		checkResult.Status = healthcheckPb.CheckStatus_FAIL
		checkResult.FriendlyMessageToUser = fmt.Sprintf("HTTP GET request to endpoint [%s] resulted in Status: [%s], body: [%s]", endpointToCheck, resp.Status, body)
		return checkResult
	}

	return checkResult
}

func (kubeapi *kubernetesApi) UrlFor(namespace string, extraPathStartingWithSlash string) (*url.URL, error) {
	return generateKubernetesApiBaseUrlFor(kubeapi.Host, namespace, extraPathStartingWithSlash)
}

func NewK8sAPI(shell shell.Shell, k8sConfigFilesystemPathOverride string) (KubernetesApi, error) {
	kubeconfigEnvVar := os.Getenv(kubernetesConfigFilePathEnvVariable)

	config, err := parseK8SConfig(findK8sConfigFile(k8sConfigFilesystemPathOverride, kubeconfigEnvVar, shell.HomeDir()))
	if err != nil {
		return nil, fmt.Errorf("error configuring Kubernetes API client: %v", err)
	}
	return &kubernetesApi{Config: config}, nil
}
