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

	return serveFragments(ctx, req, res.journalSpec, res.replica.index)
}

func serveFragments(
	ctx context.Context,
	req *pb.FragmentsRequest,
	spec *pb.JournalSpec,
	index *fragment.Index,
) (*pb.FragmentsResponse, error) {
	if req.SignatureTTL == nil {
		req.SignatureTTL = &defaultSignatureTTL
	}
	if req.PageLimit == 0 {
		req.PageLimit = int32(defaultPageLimit)
	}

	var fragmentSet fragment.CoverSet
	var err error
	fragmentSet, err = fragment.WalkAllStores(ctx, req.Journal, spec.Fragment.Stores)
	if err != nil {
		return nil, err
	}

	var tuples []*pb.FragmentsResponse_FragmentTuple
	tuples, err = getFragmentTuples(req, fragmentSet)
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

func getFragmentTuples(
	req *pb.FragmentsRequest,
	fragmentSet fragment.CoverSet,
) ([]*pb.FragmentsResponse_FragmentTuple, error) {
	var tuples = make([]*pb.FragmentsResponse_FragmentTuple, 0, req.PageLimit)
	var ind, found = fragmentSet.LongestOverlappingFragment(req.PageToken)
	// If the PageToken offset is larger than the largest fragment all valid fragments
	// have been returned in previous pages. Return empty slice of fragment tuples.
	if !found && ind == len(fragmentSet) {
		return tuples, nil
	}

	// After finding the PageToken offset find the first value in the index where the
	// mode time is >= the req.Begin. The fragment set is ordered on offsets
	// rather than ModTime, and this API returns returns fragments that represent a contigous
	// chunk of the journal so once a fragment has been found to satisfy this check only evalute
	// whether a fragment happened before req.End.
	for i, f := range fragmentSet[ind:] {
		if f.ModTime.Before(req.Begin) {
			continue
		}
		ind = ind + i
		break
	}

	for _, f := range fragmentSet[ind:] {
		if len(tuples) == cap(tuples) {
			break
		}

		// If the query is unbounded and there is no BackingStore or ModeTime this is a live fragment
		// and can be added to the list of tuples, but no signedURL can be consutructed for this fragment.
		if req.End.IsZero() && (f.BackingStore == "" && f.ModTime.IsZero()) {
			tuples = append(tuples, &pb.FragmentsResponse_FragmentTuple{Fragment: &f.Fragment})
			continue
		}
		// If the req.End is zero the query is considered unbounded. Otherwise
		// if the current fragment is modified after req.End we have found the end of the query range,
		// do not look further. There may be more fragments which have a ModTime within the bound of
		// the query but including them would not mean the returned fragments would not be contiguous.
		// The "out of order" fragments will be returned in subsequent pages.
		if !req.End.IsZero() && f.ModTime.After(req.End) {
			break
		}

		var tupel, err = buildFragmentTuple(f.Fragment, *req.SignatureTTL)
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
		SignedURL: signedURL,
	}, nil
}
