package broker

import (
	"time"

	"github.com/LiveRamp/gazette/v2/pkg/fragment"
	pb "github.com/LiveRamp/gazette/v2/pkg/protocol"
	gc "github.com/go-check/check"
)

type FragmentsSuite struct{}

func (s *FragmentsSuite) TestGetFragmentTuplesBoundedRange(c *gc.C) {
	var req = pb.FragmentsRequest{
		Begin:        time.Unix(100, 0),
		End:          time.Unix(180, 0),
		PageToken:    0,
		PageLimit:    int32(defaultPageLimit),
		SignatureTTL: &defaultSignatureTTL,
	}

	var fragmentSet = buildFragmentSet(fragmentFixtures)
	var expected = []*pb.FragmentsResponse_FragmentTuple{
		{
			Fragment:  &fragmentFixtures[1],
			SignedURL: filePathFixture[1],
		},
	}
	var tuples, err = getFragmentTuples(&req, fragmentSet)
	c.Check(err, gc.IsNil)
	c.Check(expected, gc.DeepEquals, tuples)

	req = pb.FragmentsRequest{
		Begin:        time.Unix(120, 0),
		End:          time.Unix(300, 0),
		PageToken:    120,
		PageLimit:    2,
		SignatureTTL: &defaultSignatureTTL,
	}
	expected = []*pb.FragmentsResponse_FragmentTuple{
		{
			Fragment:  &fragmentFixtures[2],
			SignedURL: filePathFixture[2],
		},
		{
			Fragment:  &fragmentFixtures[3],
			SignedURL: filePathFixture[3],
		},
	}
	tuples, err = getFragmentTuples(&req, fragmentSet)
	c.Check(err, gc.IsNil)
	c.Check(expected, gc.DeepEquals, tuples)
}

func (s *FragmentsSuite) TestGetFragmentTuplesUnboundedRange(c *gc.C) {
	var req = pb.FragmentsRequest{
		Begin:        time.Time{},
		End:          time.Time{},
		PageToken:    0,
		PageLimit:    int32(defaultPageLimit),
		SignatureTTL: &defaultSignatureTTL,
	}

	var fragmentSet = buildFragmentSet(fragmentFixtures)
	var expected = []*pb.FragmentsResponse_FragmentTuple{
		{
			Fragment:  &fragmentFixtures[0],
			SignedURL: filePathFixture[0],
		},
		{
			Fragment:  &fragmentFixtures[1],
			SignedURL: filePathFixture[1],
		},
		{
			Fragment:  &fragmentFixtures[2],
			SignedURL: filePathFixture[2],
		},
		{
			Fragment:  &fragmentFixtures[3],
			SignedURL: filePathFixture[3],
		},
		{
			Fragment:  &fragmentFixtures[4],
			SignedURL: filePathFixture[4],
		},
		{
			Fragment: &fragmentFixtures[5],
		},
	}
	var tuples, err = getFragmentTuples(&req, fragmentSet)
	c.Check(err, gc.IsNil)
	c.Check(expected, gc.DeepEquals, tuples)
}

var filePathFixture = []string{
	"file:///root/one/validJournal/0000000000000000-0000000000000028-0000000000000000000000000000000000000000.raw",
	"file:///root/one/validJournal/0000000000000028-000000000000006e-0000000000000000000000000000000000000000.raw",
	"file:///root/one/validJournal/0000000000000063-0000000000000082-0000000000000000000000000000000000000000.raw",
	"file:///root/one/validJournal/0000000000000083-000000000000013e-0000000000000000000000000000000000000000.raw",
	"file:///root/one/validJournal/000000000000013f-0000000000000190-0000000000000000000000000000000000000000.raw",
}
var fragmentFixtures = []pb.Fragment{
	{
		Journal:          "validJournal",
		Begin:            0,
		End:              40,
		ModTime:          time.Time{},
		BackingStore:     pb.FragmentStore("file:///root/one/"),
		CompressionCodec: pb.CompressionCodec_NONE,
	},
	{
		Journal:          "validJournal",
		Begin:            40,
		End:              110,
		ModTime:          time.Unix(101, 0),
		BackingStore:     pb.FragmentStore("file:///root/one/"),
		CompressionCodec: pb.CompressionCodec_NONE,
	},
	{
		Journal:          "validJournal",
		Begin:            99,
		End:              130,
		ModTime:          time.Unix(200, 0),
		BackingStore:     pb.FragmentStore("file:///root/one/"),
		CompressionCodec: pb.CompressionCodec_NONE,
	},
	{
		Journal:          "validJournal",
		Begin:            131,
		End:              318,
		ModTime:          time.Unix(150, 0),
		BackingStore:     pb.FragmentStore("file:///root/one/"),
		CompressionCodec: pb.CompressionCodec_NONE,
	},
	{
		Journal:          "validJournal",
		Begin:            319,
		End:              400,
		ModTime:          time.Unix(290, 0),
		BackingStore:     pb.FragmentStore("file:///root/one/"),
		CompressionCodec: pb.CompressionCodec_NONE,
	},
	{
		Journal: "validJournal",
		Begin:   380,
		End:     600,
	},
}

func buildFragmentSet(fragments []pb.Fragment) fragment.CoverSet {
	var set = fragment.CoverSet{}
	for _, f := range fragments {
		set, _ = set.Add(fragment.Fragment{Fragment: f})
	}
	return set
}

var _ = gc.Suite(&FragmentsSuite{})
