package referenceframe

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/edaniels/golog"
	"github.com/golang/geo/r3"
	commonpb "go.viam.com/api/common/v1"
	robotpb "go.viam.com/api/robot/v1"
	"go.viam.com/test"
	"go.viam.com/utils/protoutils"

	spatial "go.viam.com/rdk/spatialmath"
	rdkutils "go.viam.com/rdk/utils"
)

func TestFrameModelPart(t *testing.T) {
	jsonData, err := os.ReadFile(rdkutils.ResolveFile("config/data/model_frame.json"))
	test.That(t, err, test.ShouldBeNil)
	var modelJSONMap map[string]interface{}
	json.Unmarshal(jsonData, &modelJSONMap)
	kinematics, err := protoutils.StructToStructPb(modelJSONMap)
	test.That(t, err, test.ShouldBeNil)
	model, err := UnmarshalModelJSON(jsonData, "")
	test.That(t, err, test.ShouldBeNil)

	// minimally specified part
	part := &FrameSystemPart{
		FrameConfig: &LinkInFrame{PoseInFrame: &PoseInFrame{name: "test"}},
		ModelFrame:  nil,
	}
	_, err = part.ToProtobuf()
	test.That(t, err, test.ShouldBeNil)

	// slightly specified part
	part = &FrameSystemPart{
		FrameConfig: &LinkInFrame{PoseInFrame: &PoseInFrame{name: "test", parent: "world"}},
		ModelFrame:  nil,
	}
	result, err := part.ToProtobuf()
	test.That(t, err, test.ShouldBeNil)
	pose := &commonpb.Pose{} // zero pose
	exp := &robotpb.FrameSystemConfig{
		Frame: &commonpb.StaticFrame{
			Transform: &commonpb.Transform{
				ReferenceFrame: "test",
				PoseInObserverFrame: &commonpb.PoseInFrame{
					ReferenceFrame: "world",
					Pose:           pose,
				},
			},
		},
	}
	test.That(t, result.Frame.Transform.ReferenceFrame, test.ShouldEqual, exp.Frame.Transform.ReferenceFrame)
	test.That(t, result.Frame.Transform.PoseInObserverFrame, test.ShouldResemble, exp.Frame.Transform.PoseInObserverFrame)
	// exp.Kinematics is nil, but the struct in the struct PB
	expKin, err := protoutils.StructToStructPb(exp.Kinematics)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, result.Kinematics, test.ShouldResemble, expKin)
	// return to FrameSystemPart
	partAgain, err := ProtobufToFrameSystemPart(result)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, partAgain.FrameConfig.name, test.ShouldEqual, part.FrameConfig.name)
	test.That(t, partAgain.FrameConfig.parent, test.ShouldEqual, part.FrameConfig.parent)
	test.That(t, partAgain.FrameConfig.pose, test.ShouldResemble, spatial.NewZeroPose())
	// nil orientations become specified as zero orientations
	test.That(t, partAgain.FrameConfig.pose.Orientation(), test.ShouldResemble, spatial.NewZeroOrientation())
	test.That(t, partAgain.ModelFrame, test.ShouldResemble, part.ModelFrame)

	orientConf, err := spatial.NewOrientationConfig(spatial.NewZeroOrientation())
	test.That(t, err, test.ShouldBeNil)

	lc := &LinkConfig{
		ID:          "test",
		Parent:      "world",
		Translation: r3.Vector{1, 2, 3},
		Orientation: orientConf,
	}
	lif, err := lc.ParseConfig()
	test.That(t, err, test.ShouldBeNil)
	// fully specified part
	part = &FrameSystemPart{
		FrameConfig: lif,
		ModelFrame:  model,
	}
	result, err = part.ToProtobuf()
	test.That(t, err, test.ShouldBeNil)
	pose = &commonpb.Pose{X: 1, Y: 2, Z: 3, OZ: 1, Theta: 0}
	exp = &robotpb.FrameSystemConfig{
		Frame: &commonpb.StaticFrame{
			Transform: &commonpb.Transform{
				ReferenceFrame: "test",
				PoseInObserverFrame: &commonpb.PoseInFrame{
					ReferenceFrame: "world",
					Pose:           pose,
				},
			},
		},
		Kinematics: kinematics,
	}
	test.That(t, result.Frame.Transform.ReferenceFrame, test.ShouldEqual, exp.Frame.Transform.ReferenceFrame)
	test.That(t, result.Frame.Transform.PoseInObserverFrame, test.ShouldResemble, exp.Frame.Transform.PoseInObserverFrame)
	test.That(t, result.Kinematics, test.ShouldNotBeNil)
	// return to FrameSystemPart
	partAgain, err = ProtobufToFrameSystemPart(result)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, partAgain.FrameConfig.name, test.ShouldEqual, part.FrameConfig.name)
	test.That(t, partAgain.FrameConfig.parent, test.ShouldEqual, part.FrameConfig.parent)
	test.That(t, partAgain.FrameConfig.pose, test.ShouldResemble, part.FrameConfig.pose)
	test.That(t, partAgain.ModelFrame.Name, test.ShouldEqual, part.ModelFrame.Name)
	test.That(t,
		len(partAgain.ModelFrame.(*SimpleModel).OrdTransforms),
		test.ShouldEqual,
		len(part.ModelFrame.(*SimpleModel).OrdTransforms),
	)
}

func TestFramesFromPart(t *testing.T) {
	logger := golog.NewTestLogger(t)
	jsonData, err := os.ReadFile(rdkutils.ResolveFile("config/data/model_frame.json"))
	test.That(t, err, test.ShouldBeNil)
	model, err := UnmarshalModelJSON(jsonData, "")
	test.That(t, err, test.ShouldBeNil)
	// minimally specified part
	part := &FrameSystemPart{
		FrameConfig: &LinkInFrame{PoseInFrame: &PoseInFrame{name: "test"}},
		ModelFrame:  nil,
	}
	_, _, err = CreateFramesFromPart(part, logger)
	test.That(t, err, test.ShouldBeNil)

	// slightly specified part
	part = &FrameSystemPart{
		FrameConfig: &LinkInFrame{PoseInFrame: &PoseInFrame{name: "test", parent: "world"}},
		ModelFrame:  nil,
	}
	modelFrame, originFrame, err := CreateFramesFromPart(part, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, modelFrame, test.ShouldResemble, NewZeroStaticFrame(part.FrameConfig.name))
	originTailFrame, ok := NewZeroStaticFrame(part.FrameConfig.name + "_origin").(*staticFrame)
	test.That(t, ok, test.ShouldBeTrue)
	test.That(t, originFrame, test.ShouldResemble, &tailGeometryStaticFrame{originTailFrame})
	orientConf, err := spatial.NewOrientationConfig(spatial.NewZeroOrientation())
	test.That(t, err, test.ShouldBeNil)

	lc := &LinkConfig{
		ID:          "test",
		Parent:      "world",
		Translation: r3.Vector{1, 2, 3},
		Orientation: orientConf,
	}
	lif, err := lc.ParseConfig()
	test.That(t, err, test.ShouldBeNil)

	// fully specified part
	part = &FrameSystemPart{
		FrameConfig: lif,
		ModelFrame:  model,
	}
	modelFrame, originFrame, err = CreateFramesFromPart(part, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, modelFrame.Name(), test.ShouldEqual, part.FrameConfig.name)
	test.That(t, modelFrame.DoF(), test.ShouldResemble, part.ModelFrame.DoF())
	test.That(t, originFrame.Name(), test.ShouldEqual, part.FrameConfig.name+"_origin")
	test.That(t, originFrame.DoF(), test.ShouldHaveLength, 0)
}

func TestConvertTransformProtobufToFrameSystemPart(t *testing.T) {
	t.Run("fails on missing reference frame name", func(t *testing.T) {
		transform := NewPoseInFrame("parent", spatial.NewZeroPose())
		part, err := PoseInFrameToFrameSystemPart(transform)
		test.That(t, err, test.ShouldBeError, ErrEmptyStringFrameName)
		test.That(t, part, test.ShouldBeNil)
	})
	t.Run("converts to frame system part", func(t *testing.T) {
		testPose := spatial.NewPoseFromOrientation(r3.Vector{X: 1., Y: 2., Z: 3.}, &spatial.R4AA{Theta: math.Pi / 2, RX: 0, RY: 1, RZ: 0})
		transform := NewNamedPoseInFrame("parent", testPose, "child")
		part, err := PoseInFrameToFrameSystemPart(transform)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, part.FrameConfig.name, test.ShouldEqual, transform.Name())
		test.That(t, part.FrameConfig.parent, test.ShouldEqual, transform.FrameName())
		test.That(t, spatial.R3VectorAlmostEqual(part.FrameConfig.pose.Point(), testPose.Point(), 1e-8), test.ShouldBeTrue)
		test.That(t, spatial.OrientationAlmostEqual(part.FrameConfig.pose.Orientation(), testPose.Orientation()), test.ShouldBeTrue)
	})
}
