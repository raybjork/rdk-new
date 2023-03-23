package wheeled

import (
	"context"
	"testing"

	"github.com/edaniels/golog"
	"github.com/golang/geo/r3"
	"go.viam.com/test"

	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/services/slam"
	"go.viam.com/rdk/services/slam/fake"
	"go.viam.com/rdk/spatialmath"
)

func TestWrapWithKinematics(t *testing.T) {
	ctx := context.Background()
	logger := golog.NewTestLogger(t)

	label := "base"
	frame := &referenceframe.LinkConfig{
		Geometry: &spatialmath.GeometryConfig{
			R:                 5,
			X:                 8,
			Y:                 6,
			L:                 10,
			TranslationOffset: r3.Vector{X: 3, Y: 4, Z: 0},
			Label:             label,
		},
	}

	testCases := []struct {
		geoType spatialmath.GeometryType
		success bool
	}{
		{spatialmath.SphereType, true},
		{spatialmath.BoxType, true},
		{spatialmath.CapsuleType, true},
		{spatialmath.UnknownType, true},
		{spatialmath.GeometryType("bad"), false},
	}

	deps, err := testCfg.Validate("path")
	test.That(t, err, test.ShouldBeNil)
	motorDeps := fakeMotorDependencies(t, deps)
	kinematicCfg := testCfg

	expectedSphere, err := spatialmath.NewSphere(spatialmath.NewZeroPose(), 10, "")
	test.That(t, err, test.ShouldBeNil)

	for _, tc := range testCases {
		t.Run(string(tc.geoType), func(t *testing.T) {
			frame.Geometry.Type = tc.geoType
			kinematicCfg.Frame = frame
			basic, err := CreateWheeledBase(ctx, motorDeps, kinematicCfg, logger)
			test.That(t, err, test.ShouldBeNil)
			kb, err := WrapWithKinematics(ctx, basic.(*wheeledBase), "", fake.NewSLAM("", logger))
			test.That(t, err == nil, test.ShouldEqual, tc.success)
			if err != nil {
				return
			}
			kwb, ok := kb.(*kinematicWheeledBase)
			test.That(t, ok, test.ShouldBeTrue)
			limits := kwb.model.DoF()
			test.That(t, limits[0].Min, test.ShouldBeLessThan, 0)
			test.That(t, limits[1].Min, test.ShouldBeLessThan, 0)
			test.That(t, limits[1].Max, test.ShouldBeGreaterThan, 0)
			test.That(t, limits[1].Max, test.ShouldBeGreaterThan, 0)
			geometry, err := kwb.model.(*referenceframe.SimpleModel).Geometries(make([]referenceframe.Input, len(limits)))
			test.That(t, err, test.ShouldBeNil)
			test.That(t, geometry.GeometryByName(testCfg.Name+":"+label).AlmostEqual(expectedSphere), test.ShouldBeTrue)
		})
	}
}

func TestCurrentInputs(t *testing.T) {
	ctx := context.Background()
	logger := golog.NewTestLogger(t)

	sphere, err := spatialmath.NewSphere(spatialmath.NewZeroPose(), 400, "footprint")
	test.That(t, err, test.ShouldBeNil)
	base := &wheeledBase{
		widthMm:              400,
		wheelCircumferenceMm: 25,
		logger:               logger,
		name:                 "count basie",
		collisionGeometry:    sphere,
	}

	kb, err := WrapWithKinematics(ctx, base, "", fake.NewSLAM("", logger))
	test.That(t, err, test.ShouldBeNil)
	kwb, ok := kb.(*kinematicWheeledBase)
	test.That(t, ok, test.ShouldBeTrue)
	for i := 0; i < 100; i++ {
		slam.GetPointCloudMapFull(ctx, kwb.slam, kwb.slamName)
	}
	for i := 0; i < 10; i++ {
		inputs, err := kwb.CurrentInputs(ctx)
		test.That(t, err, test.ShouldBeNil)
		_ = inputs
		slam.GetPointCloudMapFull(ctx, kwb.slam, kwb.slamName)
	}
}

func TestGoToInputs(t *testing.T) {
	ctx := context.Background()
	logger := golog.NewTestLogger(t)

	sphere, err := spatialmath.NewSphere(spatialmath.NewZeroPose(), 400, "footprint")
	test.That(t, err, test.ShouldBeNil)
	base := &wheeledBase{
		widthMm:              400,
		wheelCircumferenceMm: 25,
		logger:               logger,
		name:                 "count basie",
		collisionGeometry:    sphere,
	}

	kb, err := WrapWithKinematics(ctx, base, "", fake.NewSLAM("", logger))
	test.That(t, err, test.ShouldBeNil)
	kwb, ok := kb.(*kinematicWheeledBase)
	test.That(t, ok, test.ShouldBeTrue)
	for i := 0; i < 10; i++ {
		err := kwb.GoToInputs(ctx, []referenceframe.Input{{10}, {10}})
		test.That(t, err, test.ShouldBeNil)
		slam.GetPointCloudMapFull(ctx, kwb.slam, kwb.slamName)
	}
}
