package consul

import (
	"fmt"

	utils "github.com/bjorand/velocidb/utils"
	consul "github.com/hashicorp/consul/api"
)

func RegisterPeerService(id string, serviceName string, serviceAddr string, consulAddr string) {
	serviceHost, servicePort, err := utils.SplitHostPort(serviceAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(serviceHost)
	// Get a new client
	config := &consul.Config{
		Address: consulAddr,
	}
	client, err := consul.NewClient(config)
	if err != nil {
		fmt.Println(err)
		return
	}
	agent := client.Agent()
	service := &consul.AgentServiceRegistration{
		ID:   fmt.Sprintf("%s:%s:%d", serviceName, serviceAddr, servicePort),
		Name: serviceName,
		// Address:           "0.0.0.0",
		Port:              int(servicePort),
		EnableTagOverride: true,
		Tags:              []string{"peers"},
		Check: &consul.AgentServiceCheck{
			TCP:      fmt.Sprintf("%s:%d", "127.0.0.1", 4300),
			Interval: "3s",
		},
	}
	if err := agent.ServiceRegister(service); err != nil {
		fmt.Println(err)
		return
	}
}
