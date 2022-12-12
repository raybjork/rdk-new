package motionplan

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/edaniels/golog"
	"go.viam.com/utils"

	"go.viam.com/rdk/referenceframe"
	"go.viam.com/rdk/spatialmath"
)

// InverseKinematicsSolver defines an interface which is used to solve inverse kinematics queries.
type InverseKinematicsSolver interface {
	solve(context.Context, chan<- []referenceframe.Input, spatialmath.Pose, []referenceframe.Input, Metric, int) error
	frame() referenceframe.Frame
	options() *ikOptions
}

type ikSolver struct {
	model  referenceframe.Frame
	logger golog.Logger
	opts   *ikOptions
}

func (ik *ikSolver) frame() referenceframe.Frame {
	return ik.model
}

func (ik *ikSolver) options() *ikOptions {
	return ik.opts
}

// NewIKSolver instantiates an InverseKinematicsSolver according to the configuration parameters defined in ikConfig.
func NewIKSolver(frame referenceframe.Frame, logger golog.Logger, ikConfig map[string]interface{}) (InverseKinematicsSolver, error) {
	// Start with normal options
	opt := newBasicIKOptions()
	opt.extra = ikConfig

	// convert map to json, then to a struct, overwriting present defaults
	jsonString, err := json.Marshal(ikConfig)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonString, opt)
	if err != nil {
		return nil, err
	}

	// infer IK solver to build based on number of threads allowed
	if opt.NumThreads <= 1 {
		return newNLOptIKSolver(frame, logger, opt)
	}
	return newEnsembleIKSolver(frame, logger, opt)
}

// BestIKSolutions takes an InverseKinematicsSolver and a goal location and calculates a number of solutions to achieve this goal, scored
// by proximity to some reference input that is also specified by the user.  Finally, a WorldState argument allows users to
// disallow or allow regions of state space through defining obstacles or interaction spaces respectively.
func BestIKSolutions(
	ctx context.Context,
	ik InverseKinematicsSolver,
	goal spatialmath.Pose,
	input []referenceframe.Input,
	worldState *referenceframe.WorldState,
	randseed int,
	nSolutions int,
) ([][]referenceframe.Input, error) {
	// Build an ephemeral framesystem and make a map of the inputs to it
	model := ik.frame()
	fs := referenceframe.NewEmptySimpleFrameSystem("temp")
	if err := fs.AddFrame(model, fs.Frame(referenceframe.World)); err != nil {
		return nil, err
	}
	inputMap := make(map[string][]referenceframe.Input, 0)
	inputMap[model.Name()] = input

	// Add a constraint for the worldState
	collisionConstraint, err := NewCollisionConstraintFromWorldState(model, fs, worldState, inputMap, false)
	if err != nil {
		return nil, err
	}
	ik.options().AddConstraint(defaultCollisionConstraintName, collisionConstraint)
	defer ik.options().RemoveConstraint(defaultCollisionConstraintName)

	// Convert list of nodes to Input
	nodes, err := getSolutions(ctx, ik, goal, input, randseed, nSolutions)
	if err != nil {
		return nil, err
	}
	solutions := make([][]referenceframe.Input, 0)
	for _, node := range nodes {
		solutions = append(solutions, node.Q())
	}
	return solutions, nil
}

func getSolutions(
	ctx context.Context,
	ik InverseKinematicsSolver,
	goal spatialmath.Pose,
	input []referenceframe.Input,
	randseed int,
	nSolutions int,
) ([]*costNode, error) {
	seedPos, err := ik.frame().Transform(input)
	if err != nil {
		return nil, err
	}
	goalPos := fixOvIncrement(goal, seedPos)

	solutionGen := make(chan []referenceframe.Input)
	ikErr := make(chan error, 1)
	defer func() { <-ikErr }()

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	// Spawn the IK solver to generate solutions until done
	utils.PanicCapturingGo(func() {
		defer close(ikErr)
		ikErr <- ik.solve(ctxWithCancel, solutionGen, goalPos, input, ik.options().metric, randseed)
	})

	solutions := map[float64][]referenceframe.Input{}

	// Solve the IK solver. Loop labels are required because `break` etc in a `select` will break only the `select`.
IK:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		select {
		case step := <-solutionGen:
			cPass, cScore := ik.options().CheckConstraints(&ConstraintInput{
				StartPos:   seedPos,
				EndPos:     goalPos,
				StartInput: input,
				EndInput:   step,
				Frame:      ik.frame(),
			})
			endPass, _ := ik.options().CheckConstraints(&ConstraintInput{
				StartPos:   goalPos,
				EndPos:     goalPos,
				StartInput: step,
				EndInput:   step,
				Frame:      ik.frame(),
			})

			if cPass && endPass {
				if cScore < ik.options().MinScore && ik.options().MinScore > 0 {
					solutions = map[float64][]referenceframe.Input{}
					solutions[cScore] = step
					// good solution, stopping early
					break IK
				}

				solutions[cScore] = step
				if len(solutions) >= nSolutions {
					// sufficient solutions found, stopping early
					break IK
				}
			}
			// Skip the return check below until we have nothing left to read from solutionGen
			continue IK
		default:
		}

		select {
		case <-ikErr:
			// If we have a return from the IK solver, there are no more solutions, so we finish processing above
			// until we've drained the channel
			break IK
		default:
		}
	}
	if len(solutions) == 0 {
		return nil, errIKSolve
	}

	keys := make([]float64, 0, len(solutions))
	for k := range solutions {
		keys = append(keys, k)
	}
	sort.Float64s(keys)

	orderedSolutions := make([]*costNode, 0)
	for _, key := range keys {
		orderedSolutions = append(orderedSolutions, newCostNode(solutions[key], key))
	}
	return orderedSolutions, nil
}
