package iterator

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/node"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/confluence/plumber"
	"github.com/arya-analytics/x/errutil"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
)

func newLocalIterator(
	db cesium.DB,
	host node.ID,
	rng telem.TimeRange,
	keys channel.Keys,
) (confluence.Segment[Request, Response], error) {
	iter := db.NewRetrieve().WhereTimeRange(rng).WhereChannels(keys.Cesium()...).Iterate()
	if iter.Error() != nil {
		return nil, errors.Wrap(iter.Error(), "[segment.iterator] - server failed to open cesium iterator")
	}

	pipe := plumber.New()

	// cesiumRes receives segments from the iterator.
	plumber.SetSource[cesium.RetrieveResponse](pipe, "iterator", iter)

	// executor executes requests as method calls on the iterator. Pipes
	// synchronous acknowledgements out to the response pipeline.
	te := newRequestExecutor(host, iter)
	plumber.SetSegment[Request, Response](pipe, "executor", te)

	// translator translates cesium res from the iterator source into
	// res transportable over the network.
	ts := newCesiumResponseTranslator(keys.CesiumMap())
	plumber.SetSegment[cesium.RetrieveResponse, Response](pipe, "translator", ts)

	c := errutil.NewCatchSimple()

	c.Exec(plumber.UnaryRouter[cesium.RetrieveResponse]{
		SourceTarget: "iterator",
		SinkTarget:   "translator",
	}.PreRoute(pipe))

	if c.Error() != nil {
		panic(c.Error())
	}

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}

	if err := seg.RouteInletTo("executor"); err != nil {
		panic(err)
	}

	if err := seg.RouteOutletFrom("translator", "executor"); err != nil {
		panic(err)
	}

	return seg, nil
}

type requestExecutor struct {
	host node.ID
	iter cesium.StreamIterator
	confluence.LinearTransform[Request, Response]
}

func newRequestExecutor(
	host node.ID,
	iter cesium.StreamIterator,
) confluence.Segment[Request, Response] {
	te := &requestExecutor{iter: iter, host: host}
	te.LinearTransform.ApplyTransform = te.execute
	return te
}

func (te *requestExecutor) execute(ctx signal.Context, req Request) (Response, bool, error) {
	res := executeRequest(ctx, te.host, te.iter, req)
	// If we don't have a valid response, don't send it.
	return res, res.Variant != 0, nil
}

type cesiumResponseTranslator struct {
	wrapper *core.CesiumWrapper
	confluence.LinearTransform[cesium.RetrieveResponse, Response]
}

func newCesiumResponseTranslator(
	keyMap map[cesium.ChannelKey]channel.Key,
) confluence.Segment[cesium.RetrieveResponse, Response] {
	wrapper := &core.CesiumWrapper{KeyMap: keyMap}
	ts := &cesiumResponseTranslator{wrapper: wrapper}
	ts.LinearTransform.ApplyTransform = ts.translate
	return ts
}

func (te *cesiumResponseTranslator) Flow(ctx signal.Context,
	opts ...confluence.Option) {
	te.LinearTransform.Flow(ctx, append(opts, confluence.Defer(func() {
		//log.Info("Translator exiting")
	}))...)

}

func (te *cesiumResponseTranslator) translate(
	ctx signal.Context,
	res cesium.RetrieveResponse,
) (Response, bool, error) {
	return Response{Variant: DataResponse, Segments: te.wrapper.Wrap(res.Segments)}, true, nil
}
