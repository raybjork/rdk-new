// Package fake implements a fake arm.
package fake

import (
	"context"
	// for arm model.
	_ "embed"

	"github.com/edaniels/golog"
	"github.com/pkg/errors"

	commonpb "go.viam.com/api/common/v1"
	pb "go.viam.com/api/component/arm/v1"
	"go.viam.com/rdk/components/arm"
	"go.viam.com/rdk/components/arm/eva"
	ur "go.viam.com/rdk/components/arm/universalrobots"
	"go.viam.com/rdk/components/arm/xarm"
	"go.viam.com/rdk/components/arm/yahboom"
	"go.viam.com/rdk/components/generic"
	"go.viam.com/rdk/config"
	"go.viam.com/rdk/motionplan"
	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/registry"
)

// ModelName is the string used to refer to the fake arm model.
const ModelName = "fake"

//go:embed fake_model.json
var fakeModelJSON []byte

// AttrConfig is used for converting config attributes.
type AttrConfig struct {
	ArmModel string `json:"arm-model"`
}

func init() {
	registry.RegisterComponent(arm.Subtype, ModelName, registry.Component{
		Constructor: func(ctx context.Context, _ registry.Dependencies, config config.Component, logger golog.Logger) (interface{}, error) {
			if config.Attributes.Bool("fail_new", false) {
				return nil, errors.New("whoops")
			}
			return NewArm(ctx, config, logger)
		},
	})

	config.RegisterComponentAttributeMapConverter(arm.SubtypeName, ModelName,
		func(attributes config.AttributeMap) (interface{}, error) {
			var conf AttrConfig
			return config.TransformAttributeMapToStruct(&conf, attributes)
		},
		&AttrConfig{},
	)
}

// NewArm returns a new fake arm.
func NewArm(ctx context.Context, cfg config.Component, logger golog.Logger) (arm.LocalArm, error) {
	var model referenceframe.Model
	var err error
	if cfg.ConvertedAttributes != nil {
		switch cfg.ConvertedAttributes.(*AttrConfig).ArmModel {
		case xarm.ModelName(6):
			model, err = xarm.Model(6, cfg.Name)
		case xarm.ModelName(7):
			model, err = xarm.Model(7, cfg.Name)
		case ur.ModelName:
			model, err = ur.Model(cfg.Name)
		case yahboom.ModelName:
			model, err = yahboom.Model(cfg.Name)
		case eva.ModelName:
			model, err = eva.Model(cfg.Name)
		case ModelName, "":
			model, err = referenceframe.UnmarshalModelJSON(fakeModelJSON, cfg.Name)
		default:
			return nil, errors.Errorf("fake arm cannot be created, unsupported arm_model: %s", cfg.ConvertedAttributes.(*AttrConfig).ArmModel)
		}
	} else {
		model, err = referenceframe.UnmarshalModelJSON(fakeModelJSON, cfg.Name)
	}
	if err != nil {
		return nil, err
	}

	mp, err := motionplan.NewCBiRRTMotionPlanner(model, 4, logger)
	if err != nil {
		return nil, err
	}

	return &Arm{
		Name:     cfg.Name,
		position: &commonpb.Pose{},
		joints:   &pb.JointPositions{Values: []float64{0, 0, 0, 0, 0, 0}},
		mp:       mp,
		model:    model,
	}, nil
}

// Arm is a fake arm that can simply read and set properties.
type Arm struct {
	generic.Echo
	Name       string
	position   *commonpb.Pose
	joints     *pb.JointPositions
	mp         motionplan.MotionPlanner
	CloseCount int
	model      referenceframe.Model
}

// ModelFrame returns the dynamic frame of the model.
func (a *Arm) ModelFrame() referenceframe.Model {
	return a.model
}

// GetEndPosition returns the set position.
func (a *Arm) EndPosition(ctx context.Context, extra map[string]interface{}) (*commonpb.Pose, error) {
	joints, err := a.JointPositions(ctx, extra)
	if err != nil {
		return nil, err
	}
	return motionplan.ComputePosition(a.mp.Frame(), joints)
}

// MoveToPosition sets the position.
func (a *Arm) MoveToPosition(
	ctx context.Context,
	pos *commonpb.Pose,
	worldState *commonpb.WorldState,
	extra map[string]interface{},
) error {
	joints, err := a.JointPositions(ctx, extra)
	if err != nil {
		return err
	}
	solution, err := a.mp.Plan(ctx, pos, a.model.InputFromProtobuf(joints), nil)
	if err != nil {
		return err
	}
	return arm.GoToWaypoints(ctx, a, solution)
}

// MoveToJointPositions sets the joints.
func (a *Arm) MoveToJointPositions(ctx context.Context, joints *pb.JointPositions, extra map[string]interface{}) error {
	inputs := a.model.InputFromProtobuf(joints)
	_, err := a.model.Transform(inputs)
	if err != nil {
		return err
	}

	copy(a.joints.Values, joints.Values)
	return nil
}

// GetJointPositions returns joints.
func (a *Arm) JointPositions(ctx context.Context, extra map[string]interface{}) (*pb.JointPositions, error) {
	retJoint := &pb.JointPositions{Values: a.joints.Values}
	return retJoint, nil
}

// Stop doesn't do anything for a fake arm.
func (a *Arm) Stop(ctx context.Context, extra map[string]interface{}) error {
	return nil
}

// IsMoving is always false for a fake arm.
func (a *Arm) IsMoving(ctx context.Context) (bool, error) {
	return false, nil
}

// CurrentInputs TODO.
func (a *Arm) CurrentInputs(ctx context.Context) ([]referenceframe.Input, error) {
	res, err := a.JointPositions(ctx, nil)
	if err != nil {
		return nil, err
	}
	return a.model.InputFromProtobuf(res), nil
}

// GoToInputs TODO.
func (a *Arm) GoToInputs(ctx context.Context, goal []referenceframe.Input) error {
	return a.MoveToJointPositions(ctx, a.model.ProtobufFromInput(goal), nil)
}

// Close does nothing.
func (a *Arm) Close() {
	a.CloseCount++
}
