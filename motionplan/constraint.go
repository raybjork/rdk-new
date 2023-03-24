package motionplan

import (
	"errors"
	"math"

	"github.com/golang/geo/r3"

	"go.viam.com/rdk/referenceframe"
	spatial "go.viam.com/rdk/spatialmath"
)

// SegmentInput contains all the information a constraint needs to determine validity for a movement.
// It contains the starting inputs, the ending inputs, corresponding poses, and the frame it refers to.
// Pose fields may be empty, and may be filled in by a constraint that needs them.
type SegmentInput struct {
	StartPosition      spatial.Pose
	EndPosition        spatial.Pose
	StartConfiguration []referenceframe.Input
	EndConfiguration   []referenceframe.Input
	Frame              referenceframe.Frame
}

// Given a constraint input with only frames and input positions, calculates the corresponding poses as needed.
func (ci *SegmentInput) resolveInputsToPositions() error {
	if ci.StartPosition == nil {
		if ci.Frame != nil {
			if ci.StartConfiguration != nil {
				pos, err := ci.Frame.Transform(ci.StartConfiguration)
				if err == nil {
					ci.StartPosition = pos
				} else {
					return err
				}
			} else {
				return errors.New("invalid constraint input")
			}
		} else {
			return errors.New("invalid constraint input")
		}
	}
	if ci.EndPosition == nil {
		if ci.Frame != nil {
			if ci.EndConfiguration != nil {
				pos, err := ci.Frame.Transform(ci.EndConfiguration)
				if err == nil {
					ci.EndPosition = pos
				} else {
					return err
				}
			} else {
				return errors.New("invalid constraint input")
			}
		} else {
			return errors.New("invalid constraint input")
		}
	}
	return nil
}

// StateInput contains all the information a constraint needs to determine validity for a movement.
// It contains the starting inputs, the ending inputs, corresponding poses, and the frame it refers to.
// Pose fields may be empty, and may be filled in by a constraint that needs them.
type StateInput struct {
	Position      spatial.Pose
	Configuration []referenceframe.Input
	Frame         referenceframe.Frame
}

// Given a constraint input with only frames and input positions, calculates the corresponding poses as needed.
func (ci *StateInput) resolveInputsToPositions() error {
	if ci.Position == nil {
		if ci.Frame != nil {
			if ci.Configuration != nil {
				pos, err := ci.Frame.Transform(ci.Configuration)
				if err == nil {
					ci.Position = pos
				} else {
					return err
				}
			} else {
				return errors.New("invalid constraint input")
			}
		} else {
			return errors.New("invalid constraint input")
		}
	}
	return nil
}

// SegmentConstraint tests whether a transition from a starting robot configuration to an ending robot configuration is valid.
// If the returned bool is true, the constraint is satisfied and the segment is valid.
type SegmentConstraint func(*SegmentInput) bool

// StateConstraint tests whether a given robot configuration is valid
// If the returned bool is true, the constraint is satisfied and the state is valid.
type StateConstraint func(*StateInput) bool

// ConstraintHandler is a convenient wrapper for constraint handling which is likely to be common among most motion
// planners. Including a constraint handler as an anonymous struct member allows reuse.
type ConstraintHandler struct {
	segmentConstraints map[string]SegmentConstraint
	stateConstraints   map[string]StateConstraint
}

// CheckStateConstraints will check a given input against all state constraints.
// Return values are:
// -- a bool representing whether all constraints passed
// -- if failing, a string naming the failed constraint.
func (c *ConstraintHandler) CheckStateConstraints(cInput *StateInput) (bool, string) {
	for name, cFunc := range c.stateConstraints {
		pass := cFunc(cInput)
		if !pass {
			return false, name
		}
	}
	return true, ""
}

// CheckSegmentConstraints will check a given input against all segment constraints.
// Return values are:
// -- a bool representing whether all constraints passed
// -- if failing, a string naming the failed constraint.
func (c *ConstraintHandler) CheckSegmentConstraints(cInput *SegmentInput) (bool, string) {
	for name, cFunc := range c.segmentConstraints {
		pass := cFunc(cInput)
		if !pass {
			return false, name
		}
	}
	return true, ""
}

// CheckStateConstraintsAcrossSegment will interpolate the given input from the StartInput to the EndInput, and ensure that all intermediate
// states as well as both endpoints satisfy all state constraints. If all constraints are satisfied, then this will return `true, nil`.
// If any constraints fail, this will return false, and an SegmentInput representing the valid portion of the segment, if any. If no
// part of the segment is valid, then `false, nil` is returned.
func (c *ConstraintHandler) CheckStateConstraintsAcrossSegment(ci *SegmentInput, resolution float64) (bool, *SegmentInput) {
	// ensure we have cartesian positions
	err := ci.resolveInputsToPositions()
	if err != nil {
		return false, nil
	}
	steps := PathStepCount(ci.StartPosition, ci.EndPosition, resolution)

	var lastGood []referenceframe.Input

	for i := 0; i <= steps; i++ {
		interp := float64(i) / float64(steps)
		interpConfig := referenceframe.InterpolateInputs(ci.StartConfiguration, ci.EndConfiguration, interp)
		interpC := &StateInput{Frame: ci.Frame, Configuration: interpConfig}
		err = interpC.resolveInputsToPositions()
		if err != nil {
			return false, nil
		}
		pass, _ := c.CheckStateConstraints(interpC)
		if !pass {
			if i == 0 {
				// fail on start pos
				return false, nil
			}
			return false, &SegmentInput{StartConfiguration: ci.StartConfiguration, EndConfiguration: lastGood}
		}
		lastGood = interpC.Configuration
	}

	return true, nil
}

// CheckSegmentAndStateValidity will check an segment input and confirm that it 1) meets all segment constraints, and 2) meets all 
// state constraints across the segment at some resolution. If it fails an intermediate state, it will return the shortest valid segment,
// provided that segment also meets segment constraints.
func (c *ConstraintHandler) CheckSegmentAndStateValidity(cInput *SegmentInput, resolution float64) (bool, *SegmentInput) {
	valid, _ := c.CheckSegmentConstraints(cInput)
	if !valid {
		return false, nil
	}
	valid, subSegment := c.CheckStateConstraintsAcrossSegment(cInput, resolution)
	if !valid {
		if subSegment != nil {
			subSegmentValid, _ := c.CheckSegmentConstraints(subSegment)
			if subSegmentValid {
				return false, subSegment
			}
		}
		return false, nil
	}
	return true, nil
}

// AddStateConstraint will add or overwrite a constraint function with a given name. A constraint function should return true
// if the given position satisfies the constraint.
func (c *ConstraintHandler) AddStateConstraint(name string, cons StateConstraint) {
	if c.stateConstraints == nil {
		c.stateConstraints = map[string]StateConstraint{}
	}
	c.stateConstraints[name] = cons
}

// RemoveStateConstraint will remove the given constraint.
func (c *ConstraintHandler) RemoveStateConstraint(name string) {
	delete(c.stateConstraints, name)
}

// StateConstraints will list all state constraints by name.
func (c *ConstraintHandler) StateConstraints() []string {
	names := make([]string, 0, len(c.stateConstraints))
	for name := range c.stateConstraints {
		names = append(names, name)
	}
	return names
}

// AddSegmentConstraint will add or overwrite a constraint function with a given name. A constraint function should return true
// if the given position satisfies the constraint.
func (c *ConstraintHandler) AddSegmentConstraint(name string, cons SegmentConstraint) {
	if c.segmentConstraints == nil {
		c.segmentConstraints = map[string]SegmentConstraint{}
	}
	c.segmentConstraints[name] = cons
}

// RemoveSegmentConstraint will remove the given constraint.
func (c *ConstraintHandler) RemoveSegmentConstraint(name string) {
	delete(c.segmentConstraints, name)
}

// SegmentConstraints will list all segment constraints by name.
func (c *ConstraintHandler) SegmentConstraints() []string {
	names := make([]string, 0, len(c.segmentConstraints))
	for name := range c.segmentConstraints {
		names = append(names, name)
	}
	return names
}

// newSelfCollisionConstraint creates a constraint that will be violated if geometries constituting the given frame ever come
// into collision with themselves outside of the collisions present for the observationInput.
// Collisions specified as collisionSpecifications will also be ignored
// if reportDistances is false, this check will be done as fast as possible, if true maximum information will be available for debugging.
func newSelfCollisionConstraint(
	frame referenceframe.Frame,
	observationInput map[string][]referenceframe.Input,
	collisionSpecifications []*Collision,
) (StateConstraint, error) {
	return newCollisionConstraint(frame, nil, observationInput, collisionSpecifications)
}

// newObstacleConstraint creates a constraint that will be violated if geometries constituting the given frame ever come
// into collision with worldState geometries outside of the collisions present for the observationInput.
// Collisions specified as collisionSpecifications will also be ignored
// if reportDistances is false, this check will be done as fast as possible, if true maximum information will be available for debugging.
func newObstacleConstraint(frame referenceframe.Frame,
	fs referenceframe.FrameSystem,
	worldState *referenceframe.WorldState,
	observationInput map[string][]referenceframe.Input,
	collisionSpecifications []*Collision,
) (StateConstraint, error) {
	// TODO(rb) it is bad practice to assume that the current inputs of the robot correspond to the passed in world state
	// the state that observed the worldState should ultimately be included as part of the worldState message
	worldState, err := worldState.ToWorldFrame(fs, observationInput)
	if err != nil {
		return nil, err
	}
	// can use zeroth element of worldState.Obstacles because ToWorldFrame returns only one GeometriesInFrame
	return newCollisionConstraint(frame, worldState.Obstacles[0].Geometries(), observationInput, collisionSpecifications)
}

// newCollisionConstraint is the most general method to create a collision constraint, which ill be violated if geometries constituting
// the given frame ever come into collision with obstacle geometries outside of the collisions present for the observationInput.
// Collisions specified as collisionSpecifications will also be ignored
// if reportDistances is false, this check will be done as fast as possible, if true maximum information will be available for debugging.
func newCollisionConstraint(
	frame referenceframe.Frame,
	obstacles []spatial.Geometry,
	observationInput map[string][]referenceframe.Input,
	collisionSpecifications []*Collision,
) (StateConstraint, error) {
	// extract inputs corresponding to the frame
	var goodInputs []referenceframe.Input
	var err error
	switch f := frame.(type) {
	case *solverFrame:
		goodInputs, err = f.mapToSlice(observationInput)
	default:
		goodInputs, err = referenceframe.GetFrameInputs(f, observationInput)
	}
	if err != nil {
		return nil, err
	}

	// create robot collision entities
	zeroVols, err := frame.Geometries(goodInputs)
	if err != nil && len(zeroVols.Geometries()) == 0 {
		return nil, err // no geometries defined for frame
	}

	// create the reference collisionGraph
	zeroCG, err := newCollisionGraph(zeroVols.Geometries(), obstacles, nil, true)
	if err != nil {
		return nil, err
	}
	for _, specification := range collisionSpecifications {
		zeroCG.addCollisionSpecification(specification)
	}

	// create constraint from reference collision graph
	constraint := func(cInput *StateInput) bool {
		internal, err := cInput.Frame.Geometries(cInput.Configuration)
		if err != nil && internal == nil {
			return false
		}

		cg, err := newCollisionGraph(internal.Geometries(), obstacles, zeroCG, false)
		if err != nil {
			return false
		}

		collisions := cg.collisions()
		return len(collisions) == 0
	}
	return constraint, nil
}

// NewAbsoluteLinearInterpolatingConstraint provides a Constraint whose valid manifold allows a specified amount of deviation from the
// shortest straight-line path between the start and the goal. linTol is the allowed linear deviation in mm, orientTol is the allowed
// orientation deviation measured by norm of the R3AA orientation difference to the slerp path between start/goal orientations.
func NewAbsoluteLinearInterpolatingConstraint(from, to spatial.Pose, linTol, orientTol float64) (StateConstraint, StateMetric) {
	orientConstraint, orientMetric := NewSlerpOrientationConstraint(from, to, orientTol)
	lineConstraint, lineMetric := NewLineConstraint(from.Point(), to.Point(), linTol)
	interpMetric := CombineMetrics(orientMetric, lineMetric)

	f := func(cInput *StateInput) bool {
		return orientConstraint(cInput) && lineConstraint(cInput)
	}
	return f, interpMetric
}

// NewProportionalLinearInterpolatingConstraint will provide the same metric and constraint as NewAbsoluteLinearInterpolatingConstraint,
// except that allowable linear and orientation deviation is scaled based on the distance from start to goal.
func NewProportionalLinearInterpolatingConstraint(from, to spatial.Pose, epsilon float64) (StateConstraint, StateMetric) {
	orientTol := epsilon * orientDist(from.Orientation(), to.Orientation())
	linTol := epsilon * from.Point().Distance(to.Point())

	return NewAbsoluteLinearInterpolatingConstraint(from, to, linTol, orientTol)
}

// NewSlerpOrientationConstraint will measure the orientation difference between the orientation of two poses, and return a constraint that
// returns whether a given orientation is within a given tolerance distance of the shortest segment between the two orientations, as 
// well as a metric which returns the distance to that valid region.
func NewSlerpOrientationConstraint(start, goal spatial.Pose, tolerance float64) (StateConstraint, StateMetric) {
	origDist := math.Max(orientDist(start.Orientation(), goal.Orientation()), defaultEpsilon)

	gradFunc := func(cInput *StateInput) float64 {
		sDist := orientDist(start.Orientation(), cInput.Position.Orientation())
		gDist := 0.

		// If origDist is less than or equal to defaultEpsilon, then the starting and ending orientations are the same and we do not need
		// to compute the distance to the ending orientation
		if origDist > defaultEpsilon {
			gDist = orientDist(goal.Orientation(), cInput.Position.Orientation())
		}
		return (sDist + gDist) - origDist
	}

	validFunc := func(cInput *StateInput) bool {
		err := cInput.resolveInputsToPositions()
		if err != nil {
			return false
		}
		return gradFunc(cInput) < tolerance
	}

	return validFunc, gradFunc
}

// NewPlaneConstraint is used to define a constraint space for a plane, and will return 1) a constraint
// function which will determine whether a point is on the plane and in a valid orientation, and 2) a distance function
// which will bring a pose into the valid constraint space. The plane normal is assumed to point towards the valid area.
// angle refers to the maximum unit sphere segment length deviation from the ov
// epsilon refers to the closeness to the plane necessary to be a valid pose.
func NewPlaneConstraint(pNorm, pt r3.Vector, writingAngle, epsilon float64) (StateConstraint, StateMetric) {
	// get the constant value for the plane
	pConst := -pt.Dot(pNorm)

	// invert the normal to get the valid AOA OV
	ov := &spatial.OrientationVector{OX: -pNorm.X, OY: -pNorm.Y, OZ: -pNorm.Z}
	ov.Normalize()

	dFunc := orientDistToRegion(ov, writingAngle)

	// distance from plane to point
	planeDist := func(pt r3.Vector) float64 {
		return math.Abs(pNorm.Dot(pt) + pConst)
	}

	// TODO: do we need to care about trajectory here? Probably, but not yet implemented
	gradFunc := func(cInput *StateInput) float64 {
		pDist := planeDist(cInput.Position.Point())
		oDist := dFunc(cInput.Position.Orientation())
		return pDist*pDist + oDist*oDist
	}

	validFunc := func(cInput *StateInput) bool {
		err := cInput.resolveInputsToPositions()
		if err != nil {
			return false
		}
		return gradFunc(cInput) < epsilon*epsilon
	}

	return validFunc, gradFunc
}

// NewLineConstraint is used to define a constraint space for a line, and will return 1) a constraint
// function which will determine whether a point is on the line, and 2) a distance function
// which will bring a pose into the valid constraint space.
// tolerance refers to the closeness to the line necessary to be a valid pose in mm.
func NewLineConstraint(pt1, pt2 r3.Vector, tolerance float64) (StateConstraint, StateMetric) {
	if pt1.Distance(pt2) < defaultEpsilon {
		tolerance = defaultEpsilon
	}

	gradFunc := func(cInput *StateInput) float64 {
		return math.Max(spatial.DistToLineSegment(pt1, pt2, cInput.Position.Point())-tolerance, 0)
	}

	validFunc := func(cInput *StateInput) bool {
		err := cInput.resolveInputsToPositions()
		if err != nil {
			return false
		}
		return gradFunc(cInput) == 0
	}

	return validFunc, gradFunc
}
