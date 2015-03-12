package gazette

import (
	gc "github.com/go-check/check"
	"math"
)

type FragmentSuite struct {
}

func (s *FragmentSuite) TestContentName(c *gc.C) {
	fragment := Fragment{
		Begin: 1234567890,
		End:   math.MaxInt64,
		Sum: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
			11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
	}
	c.Assert(fragment.ContentName(), gc.Equals,
		"00000000499602d2-7fffffffffffffff-0102030405060708090a0b0c0d0e0f1011121314")
}

func (s *FragmentSuite) TestParsing(c *gc.C) {
	fragment, err := ParseFragment(
		"00000000499602d2-7fffffffffffffff-0102030405060708090a0b0c0d0e0f1011121314")
	c.Assert(err, gc.IsNil)
	c.Assert(fragment, gc.DeepEquals, Fragment{
		Begin: 1234567890,
		End:   math.MaxInt64,
		Sum: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
			11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
	})

	// Empty spool name (begin == end, and empty checksum).
	fragment, err = ParseFragment("00000000499602d2-00000000499602d2-")
	c.Assert(err, gc.IsNil)
	c.Assert(fragment, gc.DeepEquals, Fragment{
		Begin: 1234567890,
		End:   1234567890,
		Sum:   []byte{},
	})

	_, err = ParseFragment("00000000499602d2-7fffffffffffffff-010203040506")
	c.Assert(err, gc.ErrorMatches, "invalid checksum")
	// Empty checksum disallowed if begin != end.
	_, err = ParseFragment("00000000499602d2-7fffffffffffffff-")
	c.Assert(err, gc.ErrorMatches, "invalid checksum")
	// Populated checksum disallowed if begin == end.
	_, err = ParseFragment(
		"00000000499602d2-00000000499602d2-0102030405060708090a0b0c0d0e0f1011121314")
	c.Assert(err, gc.ErrorMatches, "invalid checksum")

	_, err = ParseFragment("2-1-0102030405060708090a0b0c0d0e0f1011121314")
	c.Assert(err, gc.ErrorMatches, "invalid content range")
	_, err = ParseFragment("1-0102030405060708090a0b0c0d0e0f1011121314")
	c.Assert(err, gc.ErrorMatches, "wrong format")
}

func (s *FragmentSuite) TestSetAddInsertAtEnd(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 201, End: 301}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 200, End: 300},
		{Begin: 201, End: 301},
	})
}

func (s *FragmentSuite) TestSetAddReplaceRangeAtEnd(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 400, End: 500}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 150, End: 500}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 150, End: 500},
	})
}

func (s *FragmentSuite) TestSetAddReplaceOneAtEnd(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 199, End: 300}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 199, End: 300},
	})

	set = FragmentSet{}
	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 200, End: 301}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 200, End: 301},
	})
}

func (s *FragmentSuite) TestSetAddReplaceRangeInMiddle(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 400, End: 500}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 150, End: 450}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 150, End: 450},
		{Begin: 400, End: 500},
	})
}

func (s *FragmentSuite) TestSetAddReplaceOneInMiddle(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 150, End: 350}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 150, End: 350},
		{Begin: 300, End: 400},
	})
}

func (s *FragmentSuite) TestSetAddInsertInMiddleExactBoundaries(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 200, End: 300},
		{Begin: 300, End: 400},
	})
}

func (s *FragmentSuite) TestSetAddInsertInMiddleCloseBoundaries(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 201, End: 299}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 201, End: 299},
		{Begin: 300, End: 400},
	})
}

func (s *FragmentSuite) TestSetAddReplaceRangeAtBeginning(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 100, End: 300}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 300},
		{Begin: 300, End: 400},
	})
}

func (s *FragmentSuite) TestSetAddReplaceOneAtBeginning(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true) // Replace.
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 99, End: 200}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 99, End: 200},
		{Begin: 200, End: 300},
		{Begin: 300, End: 400},
	})
}

func (s *FragmentSuite) TestSetAddInsertAtBeginning(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 199, End: 200}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 199, End: 200},
		{Begin: 200, End: 300},
		{Begin: 300, End: 400},
	})
}

func (s *FragmentSuite) TestSetAddOverlappingRanges(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 150}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 149, End: 201}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 250}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 250, End: 300}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 299, End: 351}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 350, End: 400}), gc.Equals, true)

	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, true)

	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200},
		{Begin: 149, End: 201},
		{Begin: 200, End: 300},
		{Begin: 299, End: 351},
		{Begin: 300, End: 400},
	})
}

func (s *FragmentSuite) TestSetAddNoAction(c *gc.C) {
	var set FragmentSet

	c.Check(set.Add(Fragment{Begin: 100, End: 200, Sum: []byte{1}}),
		gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 200, End: 300, Sum: []byte{2}}),
		gc.Equals, true)
	c.Check(set.Add(Fragment{Begin: 300, End: 400, Sum: []byte{3}}),
		gc.Equals, true)

	// Precondition.
	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200, Sum: []byte{1}},
		{Begin: 200, End: 300, Sum: []byte{2}},
		{Begin: 300, End: 400, Sum: []byte{3}},
	})

	c.Check(set.Add(Fragment{Begin: 100, End: 200}), gc.Equals, false)
	c.Check(set.Add(Fragment{Begin: 101, End: 200}), gc.Equals, false)
	c.Check(set.Add(Fragment{Begin: 100, End: 199}), gc.Equals, false)
	c.Check(set.Add(Fragment{Begin: 200, End: 300}), gc.Equals, false)
	c.Check(set.Add(Fragment{Begin: 201, End: 300}), gc.Equals, false)
	c.Check(set.Add(Fragment{Begin: 200, End: 299}), gc.Equals, false)
	c.Check(set.Add(Fragment{Begin: 300, End: 400}), gc.Equals, false)
	c.Check(set.Add(Fragment{Begin: 301, End: 400}), gc.Equals, false)

	// Postcondition. No change.
	c.Check(set, gc.DeepEquals, FragmentSet{
		{Begin: 100, End: 200, Sum: []byte{1}},
		{Begin: 200, End: 300, Sum: []byte{2}},
		{Begin: 300, End: 400, Sum: []byte{3}},
	})
}

func (s *FragmentSuite) TestOffset(c *gc.C) {
	var set FragmentSet
	c.Check(set.BeginOffset(), gc.Equals, int64(0))
	c.Check(set.EndOffset(), gc.Equals, int64(0))

	set.Add(Fragment{Begin: 100, End: 150})
	c.Check(set.BeginOffset(), gc.Equals, int64(100))
	c.Check(set.EndOffset(), gc.Equals, int64(150))

	set.Add(Fragment{Begin: 140, End: 250})
	c.Check(set.BeginOffset(), gc.Equals, int64(100))
	c.Check(set.EndOffset(), gc.Equals, int64(250))

	set.Add(Fragment{Begin: 50, End: 100})
	c.Check(set.BeginOffset(), gc.Equals, int64(50))
	c.Check(set.EndOffset(), gc.Equals, int64(250))
}

func (s *FragmentSuite) TestLongestOverlappingFragment(c *gc.C) {
	var set FragmentSet

	set.Add(Fragment{Begin: 100, End: 200})
	set.Add(Fragment{Begin: 149, End: 201})
	set.Add(Fragment{Begin: 200, End: 300})
	set.Add(Fragment{Begin: 299, End: 351})
	set.Add(Fragment{Begin: 300, End: 400})
	set.Add(Fragment{Begin: 500, End: 600})

	c.Check(set.LongestOverlappingFragment(0), gc.Equals, 0)
	c.Check(set.LongestOverlappingFragment(100), gc.Equals, 0)
	c.Check(set.LongestOverlappingFragment(148), gc.Equals, 0)
	c.Check(set.LongestOverlappingFragment(149), gc.Equals, 1)
	c.Check(set.LongestOverlappingFragment(199), gc.Equals, 1)
	c.Check(set.LongestOverlappingFragment(200), gc.Equals, 2)
	c.Check(set.LongestOverlappingFragment(298), gc.Equals, 2)
	c.Check(set.LongestOverlappingFragment(299), gc.Equals, 3)
	c.Check(set.LongestOverlappingFragment(300), gc.Equals, 4)
	c.Check(set.LongestOverlappingFragment(300), gc.Equals, 4)
	c.Check(set.LongestOverlappingFragment(400), gc.Equals, 5) // Not covered.
	c.Check(set.LongestOverlappingFragment(401), gc.Equals, 5) // Not covered.
	c.Check(set.LongestOverlappingFragment(599), gc.Equals, 5)
	c.Check(set.LongestOverlappingFragment(600), gc.Equals, 6)
}

var _ = gc.Suite(&FragmentSuite{})
