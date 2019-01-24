package broker

import (
	"context"
	"time"

	"github.com/LiveRamp/gazette/v2/pkg/fragment"
	pb "github.com/LiveRamp/gazette/v2/pkg/protocol"
)

var (
	defaultSignatureTTL = time.Hour * 24
	defaultPageLimit    = 100
)

// Fragments dispatches the JournalServer.Fragments API.
func (svc *Service) Fragments(ctx context.Context, req *pb.FragmentsRequest) (*pb.FragmentsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var res, err = svc.resolver.resolve(resolveArgs{
		ctx:                   ctx,
		journal:               req.Journal,
		mayProxy:              !req.DoNotProxy,
		requirePrimary:        false,
		requireFullAssignment: false,
		proxyHeader:           req.Header,
	})

	if err != nil {
		return nil, err
	} else if res.status != pb.Status_OK {
		return &pb.FragmentsResponse{Status: res.status, Header: &res.Header}, nil
	} else if !res.journalSpec.Flags.MayRead() {
		return (&pb.FragmentsResponse{Status: pb.Status_NOT_ALLOWED, Header: &res.Header}), nil
	} else if res.replica == nil {
		req.Header = &res.Header // Attach resolved Header to |req|, which we'll forward.
		ctx = pb.WithDispatchRoute(ctx, req.Header.Route, req.Header.ProcessId)

		return svc.jc.Fragments(ctx, req)
	}

	var tuples []*pb.FragmentsResponse_FragmentTuple
	tuples, err = serveFragments(ctx, req, res.journalSpec, res.replica.index)
	if err != nil {
		return nil, pb.ExtendContext(err, "error fetching fragments")
	}

	var resp = &pb.FragmentsResponse{
		Status:    pb.Status_OK,
		Fragments: tuples,
	}
	if len(tuples) > 0 {
		resp.PageToken = tuples[len(tuples)-1].Fragment.End + 1
	} else {
		resp.PageToken = req.PageToken
	}

	return resp, nil
}

func serveFragments(
	ctx context.Context,
	req *pb.FragmentsRequest,
	spec *pb.JournalSpec,
	index *fragment.Index,
) ([]*pb.FragmentsResponse_FragmentTuple, error) {
	var signatureTTL time.Duration
	if req.SignatureTTL != nil {
		signatureTTL = *req.SignatureTTL
	} else {
		signatureTTL = defaultSignatureTTL
	}
	var pageLimit int32
	if req.PageLimit != 0 {
		pageLimit = req.PageLimit
	} else {
		pageLimit = int32(defaultPageLimit)
	}

	// If there are no stores return an empty list of fragments.
	if len(spec.Fragment.Stores) == 0 {
		return []*pb.FragmentsResponse_FragmentTuple{}, nil
	}

	var tuples = make([]*pb.FragmentsResponse_FragmentTuple, 0, pageLimit)
	var fragmentSet, err = fragment.WalkAllStores(ctx, req.Journal, spec.Fragment.Stores)
	if err != nil {
		return tuples, err
	}

	var ind, found = fragmentSet.LongestOverlappingFragment(req.PageToken)
	// If the PageToken offset is larger than the largest fragment all valid fragments
	// have been returned in previous pages. Return empty slice of fragment tuples.
	if !found && ind == len(fragmentSet) {
		return tuples, nil
	}

	for i, f := range fragmentSet[ind:] {
		if f.ModTime.Equal(req.Begin) || f.ModTime.After(req.Begin) {
			continue
		}
		ind = i
		break
	}

	for _, f := range fragmentSet[ind:] {
		// break if page is full.
		if len(tuples) == cap(tuples) {
			break
		}

		// If there is no BackingStore or ModeTime this is a live fragment and can be added to
		// the list of tuples, but no signedURL can be consutructed for this fragment.
		if f.BackingStore != "" && !f.ModTime.IsZero() {
			tuples = append(tuples, &pb.FragmentsResponse_FragmentTuple{Fragment: &f.Fragment})
			continue
		}
		// If the req.End is zero the query is considered unbounded. Otherwise
		// if the current fragment is modified after req.End we have found the end of the query range,
		// do not look further.
		if !req.End.IsZero() && f.ModTime.After(req.End) {
			break
		}

		var tupel, err = buildFragmentTuple(f.Fragment, signatureTTL)
		if err != nil {
			return tuples, err
		}
		tuples = append(tuples, tupel)
	}

	return tuples, nil
}

func buildFragmentTuple(f pb.Fragment, ttl time.Duration) (*pb.FragmentsResponse_FragmentTuple, error) {
	var signedURL string
	var err error
	if f.BackingStore != "" {
		signedURL, err = fragment.SignGetURL(f, ttl)
		if err != nil {
			return nil, err
		}
	}

	return &pb.FragmentsResponse_FragmentTuple{
		Fragment:  &f,
		SingedURL: signedURL,
	}, nil
}
