package channel

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	channelv1 "github.com/arya-analytics/delta/pkg/transport/grpc/gen/proto/go/distribution/channel/v1"
	"github.com/arya-analytics/x/address"
	grpcx "github.com/arya-analytics/x/grpc"
	"github.com/arya-analytics/x/telem"
	"go/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type core struct {
	*grpcx.Pool
}

func (c core) String() string { return "grpc" }

type transport struct {
	channelv1.UnimplementedChannelServiceServer
	core
	handle func(context.Context, channel.CreateRequest) (types.Nil, error)
}

var (
	_ channel.CreateTransport                     = (*transport)(nil)
	_ channelv1.UnimplementedChannelServiceServer = (*transport)(nil)
)

func (t *transport) Send(
	ctx context.Context,
	addr address.Address,
	msg channel.CreateRequest,
) (types.Nil, error) {
	conn, err := t.Pool.Acquire(addr)
	if err != nil {
		return types.Nil{}, err
	}
	_, err = channelv1.NewChannelServiceClient(conn).Create(
		ctx,
		t.translateBackward(msg),
	)
	return types.Nil{}, err
}

func (t *transport) Handle(
	handle func(context.Context, channel.CreateRequest) (types.Nil, error),
) {
	t.handle = handle
}

func (t *transport) Create(
	ctx context.Context,
	req *channelv1.CreateRequest,
) (*emptypb.Empty, error) {
	_, err := t.handle(ctx, t.translateForward(req))
	return &emptypb.Empty{}, err
}

func (t *transport) translateForward(
	req *channelv1.CreateRequest,
) (tr channel.CreateRequest) {
	for _, ch := range req.Channels {
		tr.Channels = append(tr.Channels, channel.Channel{
			Name:   ch.Name,
			NodeID: node.ID(ch.NodeId),
			Cesium: cesium.Channel{
				Key:      cesium.ChannelKey(ch.Key),
				DataRate: telem.DataRate(ch.DataRate),
				DataType: telem.DataType(ch.Density),
			},
		})
	}
	return tr
}

func (t *transport) translateBackward(
	req channel.CreateRequest,
) *channelv1.CreateRequest {
	tr := &channelv1.CreateRequest{}
	for _, ch := range req.Channels {
		tr.Channels = append(tr.Channels, &channelv1.Channel{
			Name:     ch.Name,
			NodeId:   int32(ch.NodeID),
			Key:      int32(ch.Cesium.Key),
			DataRate: float64(ch.Cesium.DataRate),
			Density:  int32(ch.Cesium.DataType),
		})
	}
	return tr
}

func New(server *grpc.Server, pool *grpcx.Pool) channel.CreateTransport {
	t := &transport{
		core: core{Pool: pool},
	}
	channelv1.RegisterChannelServiceServer(server, t)
	return t
}
