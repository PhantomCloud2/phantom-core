package v2

import (
	pb "internal-libcore/corerpc"
	"github.com/sagernet/sing-box/experimental/libbox"
)

func SelectOutbound(in *pb.SelectOutboundRequest) (*pb.Response, error) {
	err := libbox.NewStandaloneCommandClient().SelectOutbound(in.GetGroupTag(), in.GetOutboundTag())
	if err != nil {
		return &pb.Response{
			ResponseCode: pb.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}

	return &pb.Response{
		ResponseCode: pb.ResponseCode_OK,
		Message:      "",
	}, nil
}

func URLTest(in *pb.UrlTestRequest) (*pb.Response, error) {
	err := libbox.NewStandaloneCommandClient().URLTest(in.GetGroupTag())
	if err != nil {
		return &pb.Response{
			ResponseCode: pb.ResponseCode_FAILED,
			Message:      err.Error(),
		}, err
	}

	return &pb.Response{
		ResponseCode: pb.ResponseCode_OK,
		Message:      "",
	}, nil
}
