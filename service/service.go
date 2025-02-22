package service

import (
	"fmt"

	"github.com/kardianos/service"
)

type Program struct{}

func (p *Program) Start(s service.Service) error {
	fmt.Println("Service started")
	return nil
}

func (p *Program) Stop(s service.Service) error {
	fmt.Println("Service stopped")
	return nil
}

func InstallService() error {
	prg := &Program{}
	svcConfig := &service.Config{
		Name:        "PartyService",
		DisplayName: "Party Service",
		Description: "This is a party service.",
	}

	svc, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("error creating service: %w", err)
	}

	err = svc.Install()
	if err != nil {
		return fmt.Errorf("error installing service: %w", err)
	}

	fmt.Println("Service installed successfully.")
	return nil
}

func UninstallService() error {
	prg := &Program{}
	svcConfig := &service.Config{
		Name: "PartyService",
	}

	svc, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("error creating service: %w", err)
	}

	err = svc.Uninstall()
	if err != nil {
		return fmt.Errorf("error uninstalling service: %w", err)
	}

	fmt.Println("Service uninstalled successfully.")
	return nil
}
