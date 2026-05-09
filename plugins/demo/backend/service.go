package main

type DemoService struct{}

func (s *DemoService) Name() string {
	return "demo-separated-service"
}
