// Package main shows a simple server with a fake arm.
package main

import (
	"context"

	"github.com/edaniels/golog"
	"github.com/edaniels/gostream/codec/x264"
	"go.viam.com/utils"

	"go.viam.com/rdk/components/arm"
	"go.viam.com/rdk/components/arm/fake"
	"go.viam.com/rdk/config"
	"go.viam.com/rdk/resource"
	robotimpl "go.viam.com/rdk/robot/impl"
	"go.viam.com/rdk/robot/web"
)

var logger = golog.NewDevelopmentLogger("simpleserver")

func main() {
	utils.ContextualMain(mainWithArgs, logger)
}

func mainWithArgs(ctx context.Context, args []string, logger golog.Logger) error {
	arm1Name := arm.Named("arm1")
	arm1, err := fake.NewArm(ctx, config.Component{Name: arm1Name.Name}, logger)
	if err != nil {
		return err
	}
	myRobot, err := robotimpl.RobotFromResources(
		ctx,
		map[resource.Name]interface{}{
			arm1Name: arm1,
		},
		logger,
		robotimpl.WithWebOptions(web.WithStreamConfig(x264.DefaultStreamConfig)),
	)
	if err != nil {
		return err
	}
	defer myRobot.Close(ctx)

	return web.RunWebWithConfig(ctx, myRobot, &config.Config{}, logger)
}
