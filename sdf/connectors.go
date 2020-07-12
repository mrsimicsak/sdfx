package sdf

// Connector3d stores the information needed to connector to another part
type Connector3d struct {
	Position V3
	Vector   V3
	Angle    float64
}

// Transform3DConnector applies a transformation matrix to an SDF3 and a connector.
// func Transform3DConnector(sdf SDF3, connectors map[string]Connector3d, matrix M44) (SDF3, map[string]Connector3d) {
// 	s := TransformSDF3{}
// 	s.sdf = sdf
// 	s.matrix = matrix
// 	s.inverse = matrix.Inverse()
// 	s.bb = matrix.MulBox(sdf.BoundingBox())

// 	for key := range connectors {
// 		connector := connectors[key]
// 		connector.Position = matrix.MulPosition(connectors[key].Position)
// 		connectors[key] = connector
// 	}
// 	return &s, connectors
// }

// ConnectorizedSDF3 is an SDF3 that can store connectors
type ConnectorizedSDF3 interface {
	SDF3
	Connectors() map[string]Connector3d
	AddConnector(name string, connector Connector3d)
	Connect(parentConnector string, child ConnectorizedSDF3, childConnector string) ConnectorizedSDF3
}

// SDF3WithConnectors is a SDF3 with connectors
type SDF3WithConnectors struct {
	SDF3
	connectors map[string]Connector3d
}

// Connectors returns all of the connectors
func (s *SDF3WithConnectors) Connectors() map[string]Connector3d {

	return s.connectors

}

// AddConnector add a Connector3d to an SDF3
func (s *SDF3WithConnectors) AddConnector(name string, connector Connector3d) {

	if s.connectors == nil {
		s.connectors = make(map[string]Connector3d)
	}

	s.connectors[name] = connector

}

// Connect moves a child SDF so the specified connectors on the parent and child align, unions them and returns the union.
func (s *SDF3WithConnectors) Connect(parentConnector string, child ConnectorizedSDF3, childConnector string) ConnectorizedSDF3 {

	possDiff := s.connectors[parentConnector].Position.Sub(child.Connectors()[childConnector].Position)

	transformedChild := Transform3D(child, Translate3d(possDiff))

	s2 := UnionConnectorizedSDF3{}

	s2.sdf = []SDF3{s, transformedChild}

	// work out the bounding box
	s2.bb = s.BoundingBox().Extend(transformedChild.BoundingBox())
	s2.min = Min

	s2.connectors = s.Connectors()
	return &s2
}

// UnionConnectorizedSDF3 is a union of SDF3s.
type UnionConnectorizedSDF3 struct {
	sdf        []SDF3
	connectors map[string]Connector3d
	min        MinFunc
	bb         Box3
}

// Evaluate returns the minimum distance to an SDF3 union.
func (s *UnionConnectorizedSDF3) Evaluate(p V3) float64 {
	var d float64
	for i, x := range s.sdf {
		if i == 0 {
			d = x.Evaluate(p)
		} else {
			d = s.min(d, x.Evaluate(p))
		}
	}
	return d
}

// BoundingBox returns the bounding box of an SDF3 union.
func (s *UnionConnectorizedSDF3) BoundingBox() Box3 {
	return s.bb
}

// SetMin is used to control blending
func (s *UnionConnectorizedSDF3) SetMin(min MinFunc) {
	s.min = min
}

// AddConnector add a Connector3d to an SDF3
func (s *UnionConnectorizedSDF3) AddConnector(name string, connector Connector3d) {

	s.connectors[name] = connector

}

// Connect returns the union of multiple SDF3 objects.
func (s *UnionConnectorizedSDF3) Connect(parentConnector string, child ConnectorizedSDF3, childConnector string) ConnectorizedSDF3 {

	possDiff := s.connectors[parentConnector].Position.Sub(child.Connectors()[childConnector].Position)

	transformedChild := Transform3D(child, Translate3d(possDiff))

	s2 := UnionConnectorizedSDF3{}

	s2.sdf = append(s.sdf, transformedChild)

	// work out the bounding box
	s2.bb = s.BoundingBox().Extend(transformedChild.BoundingBox())
	s2.min = Min

	s2.connectors = s.Connectors()
	return &s2
}

// Connectors returns the map of Connector3ds associated with the SDF
func (s *UnionConnectorizedSDF3) Connectors() map[string]Connector3d {

	if s.connectors == nil {
		s.connectors = make(map[string]Connector3d)
	}

	return s.connectors

}
