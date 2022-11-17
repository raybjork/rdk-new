package yahboom

import (
	"context"
	"testing"

	"github.com/edaniels/golog"
	"github.com/golang/geo/r3"
	componentpb "go.viam.com/api/component/arm/v1"
	"go.viam.com/test"

	"go.viam.com/rdk/motionplan"
	"go.viam.com/rdk/spatialmath"
)

func TestJointConfig(t *testing.T) {
	test.That(t, joints[0].toValues(joints[0].toHw(-45)), test.ShouldAlmostEqual, -45, .1)
	test.That(t, joints[0].toValues(joints[0].toHw(0)), test.ShouldAlmostEqual, 0, .1)
	test.That(t, joints[0].toValues(joints[0].toHw(45)), test.ShouldAlmostEqual, 45, .1)
	test.That(t, joints[0].toValues(joints[0].toHw(90)), test.ShouldAlmostEqual, 90, .1)
	test.That(t, joints[0].toValues(joints[0].toHw(135)), test.ShouldAlmostEqual, 135, .1)
	test.That(t, joints[0].toValues(joints[0].toHw(200)), test.ShouldAlmostEqual, 200, .1)
}

func TestDofBotIK(t *testing.T) {
	ctx := context.Background()
	logger := golog.NewTestLogger(t)

	model, err := Model("test")
	test.That(t, err, test.ShouldBeNil)
	mp, err := motionplan.NewCBiRRTMotionPlanner(model, 4, logger)
	test.That(t, err, test.ShouldBeNil)

	goal := spatialmath.NewPoseFromOrientation(
		r3.Vector{X: 206.59, Y: -1.57, Z: 253.05},
		&spatialmath.OrientationVectorDegrees{Theta: -180, OX: -.53, OY: 0, OZ: .85},
	)
	opt := motionplan.NewBasicPlannerOptions()
	opt.SetMetric(motionplan.NewPositionOnlyMetric())
	_, err = mp.Plan(ctx, goal, model.InputFromProtobuf(&componentpb.JointPositions{Values: make([]float64, 5)}), opt)
	test.That(t, err, test.ShouldBeNil)
}
