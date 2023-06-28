package referenceframe

import (
	"github.com/golang/geo/r3"

	"go.viam.com/rdk/spatialmath"
	"go.viam.com/rdk/utils"
)

// The following are joint types we treat as constants.
const (
	FixedJoint      = "fixed"
	ContinuousJoint = "continuous"
	PrismaticJoint  = "prismatic"
	RevoluteJoint   = "revolute"
)

type FrameConfig struct {
	Link       *LinkConfig                   `json:"frame,omitempty"`
	Geometries []*spatialmath.GeometryConfig `json:"geometries,omitempty"`
}

// LinkConfig is a StaticFrame that also has a specified parent.
type LinkConfig struct {
	ID          string                         `json:"id"`
	Translation r3.Vector                      `json:"translation"`
	Orientation *spatialmath.OrientationConfig `json:"orientation"`
	Parent      string                         `json:"parent,omitempty"`
}

// JointConfig is a frame with nonzero DOF. Supports rotational or translational.
type JointConfig struct {
	ID       string                      `json:"id"`
	Type     string                      `json:"type"`
	Parent   string                      `json:"parent"`
	Axis     spatialmath.AxisConfig      `json:"axis"`
	Max      float64                     `json:"max"`                // in mm or degs
	Min      float64                     `json:"min"`                // in mm or degs
	Geometry *spatialmath.GeometryConfig `json:"geometry,omitempty"` // only valid for prismatic/translational joints
}

// DHParamConfig is a revolute and static frame combined in a set of Denavit Hartenberg parameters.
type DHParamConfig struct {
	ID       string                      `json:"id"`
	Parent   string                      `json:"parent"`
	A        float64                     `json:"a"`
	D        float64                     `json:"d"`
	Alpha    float64                     `json:"alpha"`
	Max      float64                     `json:"max"` // in mm or degs
	Min      float64                     `json:"min"` // in mm or degs
	Geometry *spatialmath.GeometryConfig `json:"geometry,omitempty"`
}

// NewFrameConfig constructs a config from a Frame.
func NewFrameConfig(frame staticFrame) (*FrameConfig, error) {
	var geom *spatialmath.GeometryConfig
	orient, err := spatialmath.NewOrientationConfig(frame.transform.Orientation())
	if err != nil {
		return nil, err
	}
	if frame.geometry != nil {
		geom, err = spatialmath.NewGeometryConfig(frame.geometry)
		if err != nil {
			return nil, err
		}
	}
	return &FrameConfig{
		Link: &LinkConfig{
			ID:          frame.name,
			Translation: frame.transform.Point(),
			Orientation: orient,
		},
		Geometries: []*spatialmath.GeometryConfig{geom},
	}, nil
}

// ParseConfig converts a FrameConfig into a staticFrame.
func (cfg *FrameConfig) ParseConfig() (*LinkInFrame, error) {
	pose, err := cfg.Pose()
	if err != nil {
		return nil, err
	}
	var geometries []spatialmath.Geometry
	if cfg.Geometries != nil {
		for _, cfgGeometry := range cfg.Geometries {
			geo, err := cfgGeometry.ParseConfig()
			if err != nil {
				return nil, err
			}
			if geo.Label() == "" {
				geo.SetLabel(cfg.Link.ID)
			}
			geometries = append(geometries, geo)
		}
	}
	return NewLinkInFrame(cfg.Link.Parent, pose, cfg.Link.ID, geometries), nil
}

// Pose will parse out the Pose of a LinkConfig and return it if it is valid.
func (cfg *FrameConfig) Pose() (spatialmath.Pose, error) {
	link := cfg.Link
	if link != nil {
		return nil, ErrNilPose
	}
	pt := link.Translation
	if link.Orientation != nil {
		orient, err := link.Orientation.ParseConfig()
		if err != nil {
			return nil, err
		}
		return spatialmath.NewPose(pt, orient), nil
	}
	return spatialmath.NewPoseFromPoint(pt), nil
}

// ToFrame converts a JointConfig into a joint frame.
func (cfg *JointConfig) ToFrame() (Frame, error) {
	switch cfg.Type {
	case RevoluteJoint:
		return NewRotationalFrame(cfg.ID, cfg.Axis.ParseConfig(),
			Limit{Min: utils.DegToRad(cfg.Min), Max: utils.DegToRad(cfg.Max)})
	case PrismaticJoint:
		return NewTranslationalFrame(cfg.ID, r3.Vector(cfg.Axis),
			Limit{Min: cfg.Min, Max: cfg.Max})
	default:
		return nil, NewUnsupportedJointTypeError(cfg.Type)
	}
}

// ToDHFrames converts a DHParamConfig into a joint frame and a link frame.
func (cfg *DHParamConfig) ToDHFrames() (Frame, Frame, error) {
	jointID := cfg.ID + "_j"
	rFrame, err := NewRotationalFrame(jointID, spatialmath.R4AA{RX: 0, RY: 0, RZ: 1},
		Limit{Min: utils.DegToRad(cfg.Min), Max: utils.DegToRad(cfg.Max)})
	if err != nil {
		return nil, nil, err
	}

	// Link part of DH param
	linkID := cfg.ID
	pose := spatialmath.NewPoseFromDH(cfg.A, cfg.D, utils.DegToRad(cfg.Alpha))
	var lFrame Frame
	if cfg.Geometry != nil {
		geometryCreator, err := cfg.Geometry.ParseConfig()
		if err != nil {
			return nil, nil, err
		}
		lFrame, err = NewStaticFrameWithGeometry(linkID, pose, geometryCreator)
		if err != nil {
			return nil, nil, err
		}
	} else {
		lFrame, err = NewStaticFrame(cfg.ID, pose)
		if err != nil {
			return nil, nil, err
		}
	}
	return rFrame, lFrame, nil
}
