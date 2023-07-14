package operate

import (
	"context"
	"log"
	"mydnns/service/myrpc/proto"
	"strings"

	alidns20150109 "github.com/alibabacloud-go/alidns-20150109/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// 设备类型列表
func DnnsPost(ctx context.Context, in *proto.DnnsPostReq) (*proto.DnnsPostResp, error) {
	if in.Value == "" {
		ip, err := getIP(ctx)
		if err != nil {
			return nil, err
		}
		in.Value = ip
	}

	config := &openapi.Config{
		AccessKeyId:     &in.AccessKeyId,
		AccessKeySecret: &in.AccessKeySecret,
		Endpoint:        &in.Endpoint,
	}
	client, err := alidns20150109.NewClient(config)
	if err != nil {
		return nil, err
	}
	updateDomainRecordRequest := &alidns20150109.UpdateDomainRecordRequest{
		RecordId: &in.RecordId,
		RR:       &in.RR,
		Type:     &in.Type,
		Value:    &in.Value,
	}
	runtime := &util.RuntimeOptions{}
	_, err = client.UpdateDomainRecordWithOptions(updateDomainRecordRequest, runtime)
	if err != nil {
		return nil, err
	}
	log.Println("解析修改成功")
	return &proto.DnnsPostResp{
		Message: "解析修改成功",
	}, nil
}

// getIP 获取IP地址
func getIP(ctx context.Context) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	ip := ""
	if len(md["x-forwarded-for"]) == 0 || md["x-forwarded-for"][0] == "" {
		p, _ := peer.FromContext(ctx)
		ip = strings.Split(p.Addr.String(), ":")[0]
	} else {
		ip = md["x-forwarded-for"][0]
	}

	return ip, nil
}

// 获取公网IP
func DnnsGet(ctx context.Context, in *proto.DnnsGetReq) (*proto.DnnsGetResp, error) {
	ip, err := getIP(ctx)
	if err != nil {
		return nil, err
	}
	return &proto.DnnsGetResp{
		Data: ip,
	}, nil
}
