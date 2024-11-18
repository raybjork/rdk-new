// Package register registers all relevant MovementSensors
package register

import (
	// Load all movementsensors.
	_ "go.viam.com/rdk/components/movementsensor/adxl345"
	_ "go.viam.com/rdk/components/movementsensor/fake"
	_ "go.viam.com/rdk/components/movementsensor/imuwit"
	_ "go.viam.com/rdk/components/movementsensor/merged"
	_ "go.viam.com/rdk/components/movementsensor/mpu6050"
	_ "go.viam.com/rdk/components/movementsensor/replay"
	_ "go.viam.com/rdk/components/movementsensor/wheeledodometry"
)
