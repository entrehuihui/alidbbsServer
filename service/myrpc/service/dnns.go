package service

import (
	"context"
	"mydnns/service/myrpc/proto"
	"mydnns/service/operate"

	"google.golang.org/grpc/status"
)

// 设备类型列表
func (s Service) DnnsPost(ctx context.Context, in *proto.DnnsPostReq) (*proto.DnnsPostResp, error) {
	if in.AccessKeyId == "" || in.AccessKeySecret == "" || in.Endpoint == "" || in.RecordId == "" || in.RR == "" || in.Type == "" {
		return nil, status.Errorf(500, "参数错误")
	}
	return operate.DnnsPost(ctx, in)
}

// 获取公网IP
func (s Service) DnnsGet(ctx context.Context, in *proto.DnnsGetReq) (*proto.DnnsGetResp, error){
	return operate.DnnsGet(ctx, in)
}

