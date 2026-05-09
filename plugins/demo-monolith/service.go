package main

type Service struct{}

func (s *Service) Ping() string {
	return "pong"
}
