package referenceframe

import (
	"math"
	"math/rand"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/pkg/errors"
	pb "go.viam.com/api/component/arm/v1"
	"go.viam.com/test"

	spatial "go.viam.com/rdk/spatialmath"
	"go.viam.com/rdk/utils"
)

func TestStaticFrame(t *testing.T) {
	// define a static transform
	expPose := spatial.NewPoseFromOrientation(r3.Vector{1, 2, 3}, &spatial.R4AA{math.Pi / 2, 0., 0., 1.})
	frame, err := NewStaticFrame("test", expPose)
	test.That(t, err, test.ShouldBeNil)
	// get expected transform back
	emptyInput := FloatsToInputs([]float64{})
	pose, err := frame.Transform(emptyInput)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, pose, test.ShouldResemble, expPose)
	// if you feed in non-empty input, should get err back
	nonEmptyInput := FloatsToInputs([]float64{0, 0, 0})
	_, err = frame.Transform(nonEmptyInput)
	test.That(t, err, test.ShouldNotBeNil)
	// check that there are no limits on the static frame
	limits := frame.DoF()
	test.That(t, limits, test.ShouldResemble, []Limit{})

	errExpect := errors.New("pose is not allowed to be nil")
	f, err := NewStaticFrame("test2", nil)
	test.That(t, err.Error(), test.ShouldEqual, errExpect.Error())
	test.That(t, f, test.ShouldBeNil)
}

func TestPrismaticFrame(t *testing.T) {
	// define a prismatic transform
	limit := Limit{Min: -30, Max: 30}
	frame, err := NewTranslationalFrame("test", r3.Vector{3, 4, 0}, limit)
	test.That(t, err, test.ShouldBeNil)

	// get expected transform back
	expPose := spatial.NewPoseFromPoint(r3.Vector{3, 4, 0})
	input := FloatsToInputs([]float64{5})
	pose, err := frame.Transform(input)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, spatial.PoseAlmostEqual(pose, expPose), test.ShouldBeTrue)

	// if you feed in too many inputs, should get an error back
	input = FloatsToInputs([]float64{0, 20, 0})
	_, err = frame.Transform(input)
	test.That(t, err, test.ShouldNotBeNil)

	// if you feed in empty input, should get an error
	input = FloatsToInputs([]float64{})
	_, err = frame.Transform(input)
	test.That(t, err, test.ShouldNotBeNil)

	// if you try to move beyond set limits, should get an error
	overLimit := 50.0
	input = FloatsToInputs([]float64{overLimit})
	_, err = frame.Transform(input)
	test.That(t, err, test.ShouldBeError, errors.Errorf("%.5f %s %.5f", overLimit, OOBErrString, frame.DoF()[0]))

	// gets the correct limits back
	frameLimits := frame.DoF()
	test.That(t, frameLimits[0], test.ShouldResemble, limit)

	randomInputs := RandomFrameInputs(frame, nil)
	test.That(t, len(randomInputs), test.ShouldEqual, len(frame.DoF()))
	restrictRandomInputs := RestrictedRandomFrameInputs(frame, nil, 0.001)
	test.That(t, len(restrictRandomInputs), test.ShouldEqual, len(frame.DoF()))
	test.That(t, restrictRandomInputs[0].Value, test.ShouldBeLessThan, 0.03)
	test.That(t, restrictRandomInputs[0].Value, test.ShouldBeGreaterThan, -0.03)
}

func TestRevoluteFrame(t *testing.T) {
	axis := r3.Vector{1, 0, 0}                                                                // axis of rotation is x axis
	frame := &rotationalFrame{&baseFrame{"test", []Limit{{-math.Pi / 2, math.Pi / 2}}}, axis} // limits between -90 and 90 degrees
	// expected output
	expPose := spatial.NewPoseFromOrientation(r3.Vector{0, 0, 0}, &spatial.R4AA{math.Pi / 4, 1, 0, 0}) // 45 degrees
	// get expected transform back
	input := frame.InputFromProtobuf(&pb.JointPositions{Values: []float64{45}})
	pose, err := frame.Transform(input)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, pose, test.ShouldResemble, expPose)
	// if you feed in too many inputs, should get error back
	input = frame.InputFromProtobuf(&pb.JointPositions{Values: []float64{45, 55}})
	_, err = frame.Transform(input)
	test.That(t, err, test.ShouldNotBeNil)
	// if you feed in empty input, should get errr back
	input = frame.InputFromProtobuf(&pb.JointPositions{Values: []float64{}})
	_, err = frame.Transform(input)
	test.That(t, err, test.ShouldNotBeNil)
	// if you try to move beyond set limits, should get an error
	overLimit := 100.0 // degrees
	input = frame.InputFromProtobuf(&pb.JointPositions{Values: []float64{overLimit}})
	_, err = frame.Transform(input)
	test.That(t, err, test.ShouldBeError, errors.Errorf("%.5f %s %.5f", utils.DegToRad(overLimit), OOBErrString, frame.DoF()[0]))
	// gets the correct limits back
	limit := frame.DoF()
	expLimit := []Limit{{Min: -math.Pi / 2, Max: math.Pi / 2}}
	test.That(t, limit, test.ShouldHaveLength, 1)
	test.That(t, limit[0], test.ShouldResemble, expLimit[0])
}

func TestLinearlyActuatedRotationalFrame(t *testing.T) {
	// mapping function for a three bar linkage frame.  Theta is the output, and is opposite side c on a triangle, with sides a, b, c
	// side c is the actuated side and corresponds to a linear actuator
	// sides a and b are the distances the linear actuator is mounted away from the vertex corresponding with theta
	larf, err := NewLinearlyActuatedRotationalFrame("bar", spatial.R4AA{RZ: 1}, Limit{Min: 2, Max: 6}, 3, 4)
	test.That(t, err, test.ShouldBeNil)

	// construct model with mappedFrame
	offset := spatial.NewPoseFromPoint(r3.Vector{X: 10})
	test.That(t, err, test.ShouldBeNil)

	// this is a 3-4-5 triangle, so making the input 5 will correspond to a right trangle (theta=90 degrees) w/ new pose of X=0, Y=10, Z=0
	tf, err := larf.Transform([]Input{{5.}})
	test.That(t, err, test.ShouldBeNil)
	test.That(t, spatial.PoseAlmostCoincident(spatial.Compose(tf, offset), spatial.NewPoseFromPoint(r3.Vector{Y: 10})), test.ShouldBeTrue)

	// test that entering input outside limits will result in an OOB error
	tf, err = larf.Transform([]Input{{1.8}})
	test.That(t, err, test.ShouldNotBeNil)
	test.That(t, err.Error(), test.ShouldContainSubstring, OOBErrString)
	test.That(t,
		spatial.PoseAlmostCoincidentEps(spatial.Compose(tf, offset), spatial.NewPoseFromPoint(r3.Vector{X: 9, Y: 4.22}), .1),
		test.ShouldBeTrue,
	)
}

func TestMobile2DFrame(t *testing.T) {
	expLimit := []Limit{{-10, 10}, {-10, 10}}
	frame, err := NewMobile2DFrame("test", expLimit)
	test.That(t, err, test.ShouldBeNil)

	// expected output
	expPose := spatial.NewPoseFromPoint(r3.Vector{3, 5, 0})
	// get expected transform back
	pose, err := frame.Transform(FloatsToInputs([]float64{3, 5}))
	test.That(t, err, test.ShouldBeNil)
	test.That(t, pose, test.ShouldResemble, expPose)
	// if you feed in too many inputs, should get error back
	_, err = frame.Transform(FloatsToInputs([]float64{3, 5, 10}))
	test.That(t, err, test.ShouldNotBeNil)
	// if you feed in too few inputs, should get errr back
	_, err = frame.Transform(FloatsToInputs([]float64{3, 5, 10}))
	test.That(t, err, test.ShouldNotBeNil)
	// if you try to move beyond set limits, should get an error
	_, err = frame.Transform(FloatsToInputs([]float64{3, 100}))
	test.That(t, err, test.ShouldNotBeNil)
	// gets the correct limits back
	limit := frame.DoF()
	test.That(t, limit[0], test.ShouldResemble, expLimit[0])
}

func TestGeometries(t *testing.T) {
	bc, err := spatial.NewBoxCreator(r3.Vector{1, 1, 1}, spatial.NewZeroPose(), "")
	test.That(t, err, test.ShouldBeNil)
	pose := spatial.NewPoseFromPoint(r3.Vector{0, 10, 0})
	expectedBox := bc.NewGeometry(pose)

	// test creating a new translational frame with a geometry
	tf, err := NewTranslationalFrameWithGeometry("", r3.Vector{0, 1, 0}, Limit{Min: -30, Max: 30}, bc)
	test.That(t, err, test.ShouldBeNil)
	geometries, err := tf.Geometries(FloatsToInputs([]float64{10}))
	test.That(t, err, test.ShouldBeNil)
	test.That(t, expectedBox.AlmostEqual(geometries.Geometries()[""]), test.ShouldBeTrue)

	// test erroring correctly from trying to create a geometry for a rotational frame
	rf, err := NewRotationalFrame("", spatial.R4AA{3.7, 2.1, 3.1, 4.1}, Limit{5, 6})
	test.That(t, err, test.ShouldBeNil)
	geometries, err = rf.Geometries([]Input{})
	test.That(t, err, test.ShouldContainSubstring)
	test.That(t, geometries, test.ShouldBeNil)

	// test creating a new mobile frame with a geometry
	mf, err := NewMobile2DFrameWithGeometry("", []Limit{{-10, 10}, {-10, 10}}, bc)
	test.That(t, err, test.ShouldBeNil)
	geometries, err = mf.Geometries(FloatsToInputs([]float64{0, 10}))
	test.That(t, err, test.ShouldBeNil)
	test.That(t, expectedBox.AlmostEqual(geometries.Geometries()[""]), test.ShouldBeTrue)

	// test creating a new static frame with a geometry
	expectedBox = bc.NewGeometry(spatial.NewZeroPose())
	sf, err := NewStaticFrameWithGeometry("", pose, bc)
	test.That(t, err, test.ShouldBeNil)
	geometries, err = sf.Geometries([]Input{})
	test.That(t, err, test.ShouldBeNil)
	test.That(t, expectedBox.AlmostEqual(geometries.Geometries()[""]), test.ShouldBeTrue)

	// test inheriting a geometry creator
	sf, err = NewStaticFrameFromFrame(tf, pose)
	test.That(t, err, test.ShouldBeNil)
	geometries, err = sf.Geometries([]Input{})
	test.That(t, err, test.ShouldBeNil)
	test.That(t, expectedBox.AlmostEqual(geometries.Geometries()[""]), test.ShouldBeTrue)
}

func TestSerialization(t *testing.T) {
	makeTestFrame := func(frame Frame, err error) Frame {
		return frame
	}
	testCases := []struct {
		name  string
		frame Frame
	}{
		{"static", makeTestFrame(NewStaticFrame("foo", spatial.NewPoseFromOrientation(r3.Vector{1, 2, 3}, &spatial.R4AA{math.Pi / 2, 4, 5, 6})))},
		{"translational", makeTestFrame(NewTranslationalFrame("foo", r3.Vector{1, 0, 0}, Limit{1, 2}))},
		{"rotational", makeTestFrame(NewRotationalFrame("foo", spatial.R4AA{3.7, 2.1, 3.1, 4.1}, Limit{5, 6}))},
		{"larf", makeTestFrame(NewLinearlyActuatedRotationalFrame("foo", spatial.R4AA{RZ: 1}, Limit{Min: 2, Max: 6}, 3, 4))},
		{"mobile2D", makeTestFrame(NewMobile2DFrame("foo", []Limit{{-10, 10}, {-10, 10}}))},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			data, err := testCase.frame.MarshalJSON()
			test.That(t, err, test.ShouldBeNil)
			f2, err := UnmarshalFrameJSON(data)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, testCase.frame.AlmostEquals(f2), test.ShouldBeTrue)
		})
	}
}

func TestRandomFrameInputs(t *testing.T) {
	frame, _ := NewMobile2DFrame("", []Limit{{-10, 10}, {-10, 10}})
	seed := rand.New(rand.NewSource(23))
	for i := 0; i < 100; i++ {
		_, err := frame.Transform(RandomFrameInputs(frame, seed))
		test.That(t, err, test.ShouldBeNil)
	}

	limitedFrame, _ := NewMobile2DFrame("", []Limit{{-2, 2}, {-2, 2}})
	for i := 0; i < 100; i++ {
		_, err := limitedFrame.Transform(RestrictedRandomFrameInputs(frame, seed, .2))
		test.That(t, err, test.ShouldBeNil)
	}
}
