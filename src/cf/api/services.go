package api

import (
	"bytes"
	"cf"
	"cf/configuration"
	"cf/net"
	"encoding/json"
	"fmt"
	"strings"
)

type ServiceRepository interface {
	GetServiceOfferings() (offerings []cf.ServiceOffering, apiResponse net.ApiResponse)
	CreateServiceInstance(name string, plan cf.ServicePlan) (identicalAlreadyExists bool, apiResponse net.ApiResponse)
	CreateUserProvidedServiceInstance(name string, params map[string]string) (apiResponse net.ApiResponse)
	FindInstanceByName(name string) (instance cf.ServiceInstance, apiResponse net.ApiResponse)
	BindService(instance cf.ServiceInstance, app cf.Application) (apiResponse net.ApiResponse)
	UnbindService(instance cf.ServiceInstance, app cf.Application) (found bool, apiResponse net.ApiResponse)
	DeleteService(instance cf.ServiceInstance) (apiResponse net.ApiResponse)
	RenameService(instance cf.ServiceInstance, newName string) (apiResponse net.ApiResponse)
}

type CloudControllerServiceRepository struct {
	config  *configuration.Configuration
	gateway net.Gateway
}

func NewCloudControllerServiceRepository(config *configuration.Configuration, gateway net.Gateway) (repo CloudControllerServiceRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerServiceRepository) GetServiceOfferings() (offerings []cf.ServiceOffering, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/services?inline-relations-depth=1", repo.config.Target)
	request, apiResponse := repo.gateway.NewRequest("GET", path, repo.config.AccessToken, nil)
	if apiResponse.IsNotSuccessful() {
		return
	}

	response := new(ServiceOfferingsApiResponse)

	_, apiResponse = repo.gateway.PerformRequestForJSONResponse(request, response)
	if apiResponse.IsNotSuccessful() {
		return
	}

	for _, r := range response.Resources {
		plans := []cf.ServicePlan{}
		for _, p := range r.Entity.ServicePlans {
			plans = append(plans, cf.ServicePlan{Name: p.Entity.Name, Guid: p.Metadata.Guid})
		}
		offerings = append(offerings, cf.ServiceOffering{
			Label:       r.Entity.Label,
			Version:     r.Entity.Version,
			Provider:    r.Entity.Provider,
			Description: r.Entity.Description,
			Guid:        r.Metadata.Guid,
			Plans:       plans,
		})
	}

	return
}

func (repo CloudControllerServiceRepository) CreateServiceInstance(name string, plan cf.ServicePlan) (identicalAlreadyExists bool, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_instances", repo.config.Target)

	data := fmt.Sprintf(
		`{"name":"%s","service_plan_guid":"%s","space_guid":"%s"}`,
		name, plan.Guid, repo.config.Space.Guid,
	)
	request, apiResponse := repo.gateway.NewRequest("POST", path, repo.config.AccessToken, strings.NewReader(data))
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(request)

	if apiResponse.IsNotSuccessful() && apiResponse.ErrorCode == SERVICE_INSTANCE_NAME_TAKEN {

		serviceInstance, findInstanceApiStatus := repo.FindInstanceByName(name)

		if !findInstanceApiStatus.IsNotSuccessful() &&
			serviceInstance.ServicePlan.Guid == plan.Guid {
			apiResponse = net.ApiResponse{}
			identicalAlreadyExists = true
			return
		}
	}

	return
}

func (repo CloudControllerServiceRepository) CreateUserProvidedServiceInstance(name string, params map[string]string) (apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/user_provided_service_instances", repo.config.Target)

	type RequestBody struct {
		Name        string            `json:"name"`
		Credentials map[string]string `json:"credentials"`
		SpaceGuid   string            `json:"space_guid"`
	}

	reqBody := RequestBody{name, params, repo.config.Space.Guid}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		apiResponse = net.NewApiStatusWithError("Error parsing response", err)
		return
	}

	request, apiResponse := repo.gateway.NewRequest("POST", path, repo.config.AccessToken, bytes.NewReader(jsonBytes))
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) FindInstanceByName(name string) (instance cf.ServiceInstance, apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/spaces/%s/service_instances?return_user_provided_service_instances=true&q=name%s&inline-relations-depth=2", repo.config.Target, repo.config.Space.Guid, "%3A"+name)
	request, apiResponse := repo.gateway.NewRequest("GET", path, repo.config.AccessToken, nil)
	if apiResponse.IsNotSuccessful() {
		return
	}

	response := new(ServiceInstancesApiResponse)
	_, apiResponse = repo.gateway.PerformRequestForJSONResponse(request, response)
	if apiResponse.IsNotSuccessful() {
		return
	}

	if len(response.Resources) == 0 {
		apiResponse = net.NewNotFoundApiStatus("Service instance", name)
		return
	}

	resource := response.Resources[0]
	serviceOfferingEntity := resource.Entity.ServicePlan.Entity.ServiceOffering.Entity
	instance.Guid = resource.Metadata.Guid
	instance.Name = resource.Entity.Name

	instance.ServiceOffering.Label = serviceOfferingEntity.Label
	instance.ServiceOffering.DocumentationUrl = serviceOfferingEntity.DocumentationUrl
	instance.ServiceOffering.Description = serviceOfferingEntity.Description

	instance.ServicePlan = cf.ServicePlan{
		Name: resource.Entity.ServicePlan.Entity.Name,
		Guid: resource.Entity.ServicePlan.Metadata.Guid,
	}
	instance.ServiceBindings = []cf.ServiceBinding{}

	for _, bindingResource := range resource.Entity.ServiceBindings {
		newBinding := cf.ServiceBinding{
			Url:     bindingResource.Metadata.Url,
			Guid:    bindingResource.Metadata.Guid,
			AppGuid: bindingResource.Entity.AppGuid,
		}
		instance.ServiceBindings = append(instance.ServiceBindings, newBinding)
	}

	return
}

func (repo CloudControllerServiceRepository) BindService(instance cf.ServiceInstance, app cf.Application) (apiResponse net.ApiResponse) {
	path := fmt.Sprintf("%s/v2/service_bindings", repo.config.Target)
	body := fmt.Sprintf(
		`{"app_guid":"%s","service_instance_guid":"%s"}`,
		app.Guid, instance.Guid,
	)
	request, apiResponse := repo.gateway.NewRequest("POST", path, repo.config.AccessToken, strings.NewReader(body))
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) UnbindService(instance cf.ServiceInstance, app cf.Application) (found bool, apiResponse net.ApiResponse) {
	var path string

	for _, binding := range instance.ServiceBindings {
		if binding.AppGuid == app.Guid {
			path = repo.config.Target + binding.Url
			break
		}
	}

	if path == "" {
		return
	} else {
		found = true
	}

	request, apiResponse := repo.gateway.NewRequest("DELETE", path, repo.config.AccessToken, nil)
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) DeleteService(instance cf.ServiceInstance) (apiResponse net.ApiResponse) {
	if len(instance.ServiceBindings) > 0 {
		return net.NewApiStatusWithMessage("Cannot delete service instance, apps are still bound to it")
	}

	path := fmt.Sprintf("%s/v2/service_instances/%s", repo.config.Target, instance.Guid)
	request, apiResponse := repo.gateway.NewRequest("DELETE", path, repo.config.AccessToken, nil)
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(request)
	return
}

func (repo CloudControllerServiceRepository) RenameService(instance cf.ServiceInstance, newName string) (apiResponse net.ApiResponse) {
	body := fmt.Sprintf(`{"name":"%s"}`, newName)
	path := fmt.Sprintf("%s/v2/service_instances/%s", repo.config.Target, instance.Guid)
	request, apiResponse := repo.gateway.NewRequest("PUT", path, repo.config.AccessToken, strings.NewReader(body))
	if apiResponse.IsNotSuccessful() {
		return
	}

	apiResponse = repo.gateway.PerformRequest(request)
	return
}
