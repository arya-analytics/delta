package segment

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	segmentv1 "github.com/arya-analytics/delta/pkg/transport/grpc/gen/proto/go/distribution/segment/v1"
	"github.com/arya-analytics/x/address"
	grpcx "github.com/arya-analytics/x/grpc"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
)

type core struct {
	*grpcx.Pool
}

func (c core) String() string { return "grpc" }

type writerTransport struct {
	core
	segmentv1.UnimplementedWriterServiceServer
	handle func(context.Context, writer.Server) error
}

var (
	_ segmentv1.WriterServiceServer = (*writerTransport)(nil)
	_ writer.Transport              = (*writerTransport)(nil)
	_ writer.Client                 = (*writerClient)(nil)
	_ writer.Server                 = (*writerServer)(nil)
)

// Stream implements the writer.Transport interface.
func (w *writerTransport) Stream(
	ctx context.Context,
	target address.Address,
) (writer.Client, error) {
	conn, err := w.Pool.Acquire(target)
	if err != nil {
		return nil, err
	}
	stream, err := segmentv1.NewWriterServiceClient(conn).Write(ctx)
	if err != nil {
		return nil, err
	}
	return &writerClient{base: stream}, nil
}

// Handle implements the writer.Transport interface.
func (w *writerTransport) Handle(handle func(context.Context, writer.Server) error) {
	w.handle = handle
}

// Write implements the iterator.Transport interface.
func (w *writerTransport) Write(server segmentv1.WriterService_WriteServer) error {
	return w.handle(server.Context(), &writerServer{base: server})
}

type writerClient struct {
	writerTranslator
	base segmentv1.WriterService_WriteClient
}

// Receive implements the writer.Client interface.
func (sc *writerClient) Receive() (writer.Response, error) {
	req, err := sc.base.Recv()
	return sc.translateResponseForward(req), err
}

// Send implements the writer.Client interface.
func (sc *writerClient) Send(req writer.Request) error {
	return sc.base.Send(sc.translateRequestBackward(req))
}

// CloseSend implements the writer.Client interface.
func (sc *writerClient) CloseSend() error { return sc.base.CloseSend() }

type writerServer struct {
	writerTranslator
	base segmentv1.WriterService_WriteServer
}

// Send implements the writer.Server interface.
func (sr *writerServer) Send(req writer.Response) error {
	return sr.base.Send(sr.translateResponseBackward(req))
}

func (sr *writerServer) Receive() (writer.Request, error) {
	req, err := sr.base.Recv()
	return sr.translateRequestForward(req), err
}

type writerTranslator struct{}

func (t writerTranslator) translateRequestForward(
	req *segmentv1.WriterRequest,
) writer.Request {
	return writer.Request{Segments: translateSegmentsForward(req.Segments)}
}

func (t writerTranslator) translateRequestBackward(
	req writer.Request,
) (tr *segmentv1.WriterRequest) {
	return &segmentv1.WriterRequest{Segments: translateSegmentsBackward(req.Segments)}
}

func (t writerTranslator) translateResponseForward(
	req *segmentv1.WriterResponse,
) (tr writer.Response) {
	return writer.Response{Error: errors.New(req.Error)}
}

func (t writerTranslator) translateResponseBackward(
	req writer.Response,
) (tr *segmentv1.WriterResponse) {
	return &segmentv1.WriterResponse{Error: req.Error.Error()}
}

func translateSegmentsForward(segments []*segmentv1.Segment) []segment.Segment {
	tSegments := make([]segment.Segment, len(segments))
	for i, seg := range segments {
		key, err := channel.ParseKey(seg.ChannelKey)
		if err != nil {
			panic(err)
		}
		tSegments[i] = segment.Segment{
			ChannelKey: key,
			Segment: cesium.Segment{
				ChannelKey: key.Cesium(),
				Start:      telem.TimeStamp(seg.Start),
				Data:       seg.Data,
			},
		}
	}
	return tSegments
}

func translateSegmentsBackward(segments []segment.Segment) []*segmentv1.Segment {
	tSegments := make([]*segmentv1.Segment, len(segments))
	for i, seg := range segments {
		tSegments[i] = &segmentv1.Segment{
			ChannelKey: seg.ChannelKey.String(),
			Start:      int64(seg.Segment.Start),
			Data:       seg.Segment.Data,
		}
	}
	return tSegments
}

type iteratorTransport struct {
	core
	segmentv1.UnimplementedIteratorServiceServer
	handle func(context.Context, iterator.Server) error
}

var (
	_ segmentv1.IteratorServiceServer = (*iteratorTransport)(nil)
	_ iterator.Transport              = (*iteratorTransport)(nil)
	_ iterator.Client                 = (*iteratorClient)(nil)
	_ iterator.Server                 = (*iteratorServer)(nil)
)

func (i *iteratorTransport) Stream(
	ctx context.Context,
	target address.Address,
) (iterator.Client, error) {
	conn, err := i.Pool.Acquire(target)
	if err != nil {
		return nil, err
	}
	stream, err := segmentv1.NewIteratorServiceClient(conn).Iterate(ctx)
	if err != nil {
		return nil, err
	}
	return &iteratorClient{base: stream}, nil
}

func (i *iteratorTransport) Handle(handle func(context.Context, iterator.Server) error) {
	i.handle = handle
}

func (i *iteratorTransport) Iterate(server segmentv1.IteratorService_IterateServer) error {
	return i.handle(server.Context(), &iteratorServer{base: server})
}

type iteratorClient struct {
	iteratorTranslator
	base segmentv1.IteratorService_IterateClient
}

func (i *iteratorClient) Send(req iterator.Request) error {
	return i.base.Send(i.translateRequestBackward(req))
}

func (i *iteratorClient) Receive() (iterator.Response, error) {
	req, err := i.base.Recv()
	return i.translateResponseForward(req), err
}

func (i *iteratorClient) CloseSend() error { return i.base.CloseSend() }

type iteratorServer struct {
	iteratorTranslator
	base segmentv1.IteratorService_IterateServer
}

func (i *iteratorServer) Send(req iterator.Response) error {
	return i.base.Send(i.translateResponseBackward(req))
}

func (i *iteratorServer) Receive() (iterator.Request, error) {
	req, err := i.base.Recv()
	return i.translateRequestForward(req), err
}

type iteratorTranslator struct{}

func (t iteratorTranslator) translateRequestForward(
	req *segmentv1.IteratorRequest,
) iterator.Request {
	keys, err := channel.ParseKeys(req.Keys)
	if err != nil {
		panic(err)
	}
	return iterator.Request{
		Command: iterator.Command(req.Command),
		Span:    telem.TimeSpan(req.Span),
		Range: telem.TimeRange{
			Start: telem.TimeStamp(req.Range.Start),
			End:   telem.TimeStamp(req.Range.End),
		},
		Stamp: telem.TimeStamp(req.Stamp),
		Keys:  keys,
	}
}

func (t iteratorTranslator) translateRequestBackward(
	req iterator.Request,
) (tr *segmentv1.IteratorRequest) {
	return &segmentv1.IteratorRequest{
		Command: int32(req.Command),
		Span:    int64(req.Span),
		Range: &segmentv1.TimeRange{
			Start: int64(req.Range.Start),
			End:   int64(req.Range.End),
		},
		Stamp: int64(req.Stamp),
		Keys:  req.Keys.Strings(),
	}
}

func (t iteratorTranslator) translateResponseForward(
	req *segmentv1.IteratorResponse,
) (tr iterator.Response) {
	return iterator.Response{
		Error:    errors.New(req.Error),
		Segments: translateSegmentsForward(req.Segments),
	}
}

func (t iteratorTranslator) translateResponseBackward(
	req iterator.Response,
) (tr *segmentv1.IteratorResponse) {
	return &segmentv1.IteratorResponse{
		Error:    req.Error.Error(),
		Segments: translateSegmentsBackward(req.Segments),
	}
}

type transport struct {
	server *grpc.Server
	pool   *grpcx.Pool
	writer *writerTransport
	iter   *iteratorTransport
}

func (t *transport) Writer() writer.Transport { return t.writer }

func (t *transport) Iterator() iterator.Transport { return t.iter }

func New(server *grpc.Server, pool *grpcx.Pool) segment.Transport {
	t := &transport{
		pool:   pool,
		server: server,
		writer: &writerTransport{core: core{Pool: pool}},
		iter:   &iteratorTransport{core: core{Pool: pool}},
	}
	segmentv1.RegisterWriterServiceServer(server, t.writer)
	segmentv1.RegisterIteratorServiceServer(server, t.iter)
	return t
}
