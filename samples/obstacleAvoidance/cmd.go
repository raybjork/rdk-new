// Package main is an obstacle avoidance utility.
package main

import (
	"context"
	"flag"

	"github.com/edaniels/golog"
	"github.com/golang/geo/r3"
	"github.com/viamrobotics/visualization"
	"go.viam.com/utils"
	"go.viam.com/utils/rpc"

	"go.viam.com/rdk/components/arm"
	"go.viam.com/rdk/components/arm/fake"
	"go.viam.com/rdk/components/arm/xarm"
	"go.viam.com/rdk/config"
	"go.viam.com/rdk/grpc/client"
	pb "go.viam.com/rdk/proto/api/common/v1"
	frame "go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/robot"
	robotimpl "go.viam.com/rdk/robot/impl"
	math "go.viam.com/rdk/spatialmath"
	rdkutils "go.viam.com/rdk/utils"
)

var (
	logger  = golog.NewDevelopmentLogger("client")
	armName = xarm.ModelName(6)
)

func main() {
	utils.ContextualMain(mainWithArgs, logger)
}

func mainWithArgs(ctx context.Context, args []string, logger golog.Logger) error {
	// parse command line input
	hardware := flag.Bool("hardware", false, "choose to run on hardware")
	visualize := flag.Bool("visualize", false, "choose to display visualization")
	flag.Parse()

	// connect to the robot and get arm
	robotClient, err := connect(ctx, *hardware)
	if err != nil {
		return err
	}
	xArm, err := arm.FromRobot(robotClient, armName)
	if err != nil {
		return err
	}

	// setup planning problem - the idea is to move from one position to the other while avoiding obstalces
	position1 := r3.Vector{0, -600, 100}
	position2 := r3.Vector{-600, -300, 100}
	box, _ := math.NewBox(math.NewPoseFromPoint(r3.Vector{-400, -550, 150}), r3.Vector{300, 300, 300})
	table, _ := math.NewBox(math.NewPoseFromPoint(r3.Vector{0, 0, 0}), r3.Vector{1500, 1500, 50})
	ws, _ := math.NewBox(math.NewPoseFromPoint(r3.Vector{0, 0, 0}), r3.Vector{1500, 1500, 1000})

	// construct world state message
	obstacles := make(map[string]math.Geometry)
	obstacles["box"] = box
	obstacles["table"] = table
	workspace := make(map[string]math.Geometry)
	workspace["workspace"] = ws
	worldState := &pb.WorldState{
		Obstacles:         []*pb.GeometriesInFrame{frame.GeometriesInFrameToProtobuf(frame.NewGeometriesInFrame(frame.World, obstacles))},
		InteractionSpaces: []*pb.GeometriesInFrame{frame.GeometriesInFrameToProtobuf(frame.NewGeometriesInFrame(frame.World, workspace))},
	}

	// determine which position to assign the start and which the goal
	currentPose, err := xArm.GetEndPosition(ctx, nil)
	if err != nil {
		return err
	}
	eePosition := math.NewPoseFromProtobuf(currentPose).Point()
	delta1 := eePosition.Sub(position1).Norm()
	delta2 := eePosition.Sub(position2).Norm()
	start := math.PoseToProtobuf(math.NewPoseFromPoint(position1))
	goal := math.PoseToProtobuf(math.NewPoseFromPoint(position2))
	start.OZ = -1
	goal.OZ = -1
	if delta1 > delta2 {
		start, goal = goal, start
	}

	// ensure that the arm starts in the correct position
	inputs, err := xArm.GetJointPositions(ctx, nil)
	if err != nil {
		return err
	}
	visualization.VisualizeStep(xArm.ModelFrame(), worldState, xArm.ModelFrame().InputFromProtobuf(inputs))
	if err := xArm.MoveToPosition(ctx, start, worldState, nil); err != nil {
		return err
	}

	// move it to the goal
	solution, err := arm.Plan(ctx, robotClient, xArm, goal, nil)
	if err != nil {
		return err
	}
	if *visualize {
		// visualize if specified by flag
		if err := visualization.VisualizePlan(ctx, solution, xArm.ModelFrame(), worldState); err != nil {
			return err
		}
	}
	arm.GoToWaypoints(ctx, xArm, solution)
	return nil
}

func connect(ctx context.Context, hardware bool) (robotClient robot.Robot, err error) {
	if hardware {
		return client.New(
			context.Background(),
			"ray-pi-main.tcz8zh8cf6.viam.cloud",
			logger,
			client.WithDialOptions(rpc.WithCredentials(rpc.Credentials{
				Type:    rdkutils.CredentialsTypeRobotLocationSecret,
				Payload: "ewvmwn3qs6wqcrbnewwe1g231nvzlx5k5r5g34c31n6f7hs8",
			})),
		)
	}
	return robotimpl.RobotFromConfig(
		ctx,
		&config.Config{Components: []config.Component{{
			Name:                armName,
			Namespace:           resource.ResourceNamespaceRDK,
			Type:                arm.SubtypeName,
			Model:               "fake_ik",
			Frame:               &config.Frame{Parent: frame.World},
			ConvertedAttributes: &fake.AttrConfig{ArmModel: armName},
		}}},
		logger,
	)
}
