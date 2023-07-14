package service

import "mydnns/service/myrpc/proto"

// Service .
type Service struct {
	// ##继承
	proto.UnimplementedDnnsServer
	// ##继承
}

// NewService .
func NewService() *Service {
	s := new(Service)
	return s
}
