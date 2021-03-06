package service

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

var (
	// https://docs.docker.com/engine/reference/api/docker_remote_api/
	// `docker version` -> Server API version  => Docker 1.12x
	defaultVersion = "1.24"
	defaultHeaders = map[string]string{"User-Agent": "amplifier-1.0"}
	dockerSock     = "unix:///var/run/docker.sock"
	defaultNetwork = "amp-public"
	docker         *client.Client
	err            error
)

const serviceRoleLabelName = "io.amp.role"

// Service is used to implement ServiceServer
type Service struct{}

// SwarmMode is needed to export isServiceSpec_Mode type, which consumers can use to
// create a variable and assign either a ServiceSpec_Replicated or ServiceSpec_Global struct
type SwarmMode isServiceSpec_Mode

func init() {
	docker, err = client.NewClient(dockerSock, defaultVersion, nil, defaultHeaders)
	if err != nil {
		// fail fast
		panic(err)
	}
}

// Create implements ServiceServer
func (s *Service) Create(ctx context.Context, req *ServiceCreateRequest) (*ServiceCreateResponse, error) {
	response, err := Create(ctx, req)
	return response, err
}

// Remove implements ServiceServer
func (s *Service) Remove(ctx context.Context, req *RemoveRequest) (*RemoveResponse, error) {
	err := Remove(ctx, req.Ident)
	if err != nil {
		return nil, err
	}

	response := &RemoveResponse{
		Ident: req.Ident,
	}

	return response, nil
}

// Create uses docker api to create a service
func Create(ctx context.Context, req *ServiceCreateRequest) (*ServiceCreateResponse, error) {

	serv := req.ServiceSpec

	var serviceMode swarm.ServiceMode
	switch mode := serv.Mode.(type) {
	case *ServiceSpec_Replicated:
		serviceMode = swarm.ServiceMode{
			Replicated: &swarm.ReplicatedService{
				Replicas: &mode.Replicated.Replicas,
			},
		}
	case *ServiceSpec_Global:
		serviceMode = swarm.ServiceMode{
			Global: &swarm.GlobalService{},
		}
	}

	service := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name:   serv.Name,
			Labels: serv.Labels,
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image:           serv.Image,
				Args:            nil, //[]string
				Env:             nil, //[]string
				Labels:          serv.ContainerLabels,
				Dir:             "",
				User:            "",
				Groups:          nil, //[]string
				Mounts:          nil, //[]mount.Mount
				StopGracePeriod: nil, //*time.Duration
			},
			Networks: []swarm.NetworkAttachmentConfig{
				{
					Target:  defaultNetwork,
					Aliases: []string{req.ServiceSpec.Name},
				},
			},
			Resources:     nil, //*ResourceRequirements
			RestartPolicy: nil, //*RestartPolicy
			Placement: &swarm.Placement{
				Constraints: nil, //[]string
			},
			LogDriver: nil, //*Driver
		},
		Networks: nil, //[]NetworkAttachmentConfig
		UpdateConfig: &swarm.UpdateConfig{
			Parallelism:   0,
			Delay:         0,
			FailureAction: "",
		},
		EndpointSpec: nil, // &EndpointSpec
		Mode:         serviceMode,
	}

	// add environment
	service.TaskTemplate.ContainerSpec.Env = serv.Env

	// ensure supplied service label map is not nil, then add custom amp labels
	if service.Annotations.Labels == nil {
		service.Annotations.Labels = make(map[string]string)
	}
	service.Annotations.Labels[serviceRoleLabelName] = "user"

	if req.ServiceSpec.PublishSpecs != nil {
		nn := len(req.ServiceSpec.PublishSpecs)
		if nn > 0 {
			service.EndpointSpec = &swarm.EndpointSpec{
				Mode:  swarm.ResolutionModeVIP,
				Ports: make([]swarm.PortConfig, nn, nn),
			}
			for i, publish := range req.ServiceSpec.PublishSpecs {
				service.EndpointSpec.Ports[i] = swarm.PortConfig{
					Name:          publish.Name,
					Protocol:      swarm.PortConfigProtocol(publish.Protocol),
					TargetPort:    publish.InternalPort,
					PublishedPort: publish.PublishPort,
				}
			}
		}
	}
	options := types.ServiceCreateOptions{}

	r, err := docker.ServiceCreate(ctx, service, options)
	if err != nil {
		return nil, err
	}

	resp := &ServiceCreateResponse{
		Id: r.ID,
	}
	fmt.Printf("Service: %s created, id=%s\n", req.ServiceSpec.Name, resp.Id)
	return resp, nil
}

// Remove uses docker api to remove a service
func Remove(ctx context.Context, ID string) error {
	fmt.Printf("Service removed %s\n", ID)
	return docker.ServiceRemove(ctx, ID)
}
