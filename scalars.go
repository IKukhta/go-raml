package raml

import (
	"math/big"
	"regexp"

	"gopkg.in/yaml.v3"
)

type EnumFacets struct {
	Enum []*Node
}

func MakeEnum(v *yaml.Node, location string) ([]*Node, error) {
	if v.Kind != yaml.SequenceNode {
		return nil, NewError("enum must be sequence node", location, WithNodePosition(v))
	}
	var enums []*Node = make([]*Node, len(v.Content))
	for i, v := range v.Content {
		n, err := MakeNode(v, location)
		if err != nil {
			return nil, NewWrappedError("make node enum", err, location, WithNodePosition(v))
		}
		enums[i] = n
	}
	return enums, nil
}

type FormatFacets struct {
	Format *string
}

type IntegerFacets struct {
	Minimum    *big.Int
	Maximum    *big.Int
	MultipleOf *int64
}

type IntegerShape struct {
	BaseShape

	EnumFacets
	FormatFacets
	IntegerFacets
}

func (s *IntegerShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *IntegerShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

// func (s *IntegerShape) Validate(v interface{}) error {
// 	i, ok := v.(int64)
// 	if !ok {
// 		return fmt.Errorf("invalid value")
// 	}

// 	if s.Minimum != nil && *s.Minimum < i {
// 		return fmt.Errorf("value must be in range")
// 	}
// 	if s.Maximum != nil && i > *s.Maximum {
// 		return fmt.Errorf("value must be in range")
// 	}

// 	return nil
// }

func (s *IntegerShape) Inherit(source Shape) (Shape, error) {
	ss, ok := source.(*IntegerShape)
	if !ok {
		return nil, NewError("merge shape type mismatch", s.Location, WithPosition(&s.Position), WithInfo("source", source.Base().Type), WithInfo("target", s.Base().Type))
	}
	if s.Minimum == nil {
		s.Minimum = ss.Minimum
	} else if ss.Minimum != nil && s.Minimum.Cmp(ss.Minimum) < 0 {
		return nil, NewError("minimum constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", *ss.Minimum), WithInfo("target", *s.Minimum))
	}
	if s.Maximum == nil {
		s.Maximum = ss.Maximum
	} else if ss.Maximum != nil && s.Maximum.Cmp(ss.Maximum) > 0 {
		return nil, NewError("maximum constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", *ss.Maximum), WithInfo("target", *s.Maximum))
	}
	// TODO: multipleOf validation
	if s.MultipleOf == nil {
		// TODO: Disallow multipleOf 0 to avoid division by zero during validation
		s.MultipleOf = ss.MultipleOf
	}
	if s.Enum == nil {
		s.Enum = ss.Enum
	} else if ss.Enum != nil && !IsOverridableEnum(ss.Enum, s.Enum) {
		return nil, NewError("enum constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", ss.Enum), WithInfo("target", s.Enum))
	}
	// TODO: Formats intersection
	if s.Format == nil {
		s.Format = ss.Format
	}
	return s, nil
}

func (s *IntegerShape) Check() error {
	return nil
}

func (s *IntegerShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	for i := 0; i != len(v); i += 2 {
		node := v[i]
		valueNode := v[i+1]
		if node.Value == "minimum" {
			if valueNode.Tag != "!!int" {
				return NewError("minimum must be integer", s.Location, WithNodePosition(valueNode))
			}

			num, ok := big.NewInt(0).SetString(valueNode.Value, 10)
			if !ok {
				return NewError("invalid minimum value", s.Location, WithNodePosition(valueNode))
			}
			s.Minimum = num
		} else if node.Value == "maximum" {
			if valueNode.Tag != "!!int" {
				return NewError("maximum must be integer", s.Location, WithNodePosition(valueNode))
			}

			num, ok := big.NewInt(0).SetString(valueNode.Value, 10)
			if !ok {
				return NewError("invalid maximum value", s.Location, WithNodePosition(valueNode))
			}
			s.Maximum = num
		} else if node.Value == "multipleOf" {
			if valueNode.Tag != "!!int" {
				return NewError("multipleOf must be integer", s.Location, WithNodePosition(valueNode))
			}

			if err := valueNode.Decode(&s.MultipleOf); err != nil {
				return NewWrappedError("decode multipleOf", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "format" {
			if err := valueNode.Decode(&s.Format); err != nil {
				return NewWrappedError("decode format", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "enum" {
			enums, err := MakeEnum(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make enum", err, s.Location, WithNodePosition(valueNode))
			}
			s.Enum = enums
		} else {
			n, err := MakeNode(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make node", err, s.Location, WithNodePosition(valueNode))
			}
			s.CustomShapeFacets[node.Value] = n
		}
	}
	return nil
}

type NumberFacets struct {
	// Minimum and maximum are unset since there's no theoretical minimum and maximum for numbers by default
	Minimum    *float64
	Maximum    *float64
	MultipleOf *float64
}

type NumberShape struct {
	BaseShape

	EnumFacets
	FormatFacets
	NumberFacets
}

func (s *NumberShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *NumberShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *NumberShape) Inherit(source Shape) (Shape, error) {
	ss, ok := source.(*NumberShape)
	if !ok {
		return nil, NewError("merge shape type mismatch", s.Location, WithPosition(&s.Position), WithInfo("source", source.Base().Type), WithInfo("target", s.Base().Type))
	}
	if s.Minimum == nil {
		s.Minimum = ss.Minimum
	} else if ss.Minimum != nil && *s.Minimum < *ss.Minimum {
		return nil, NewError("minimum constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", *ss.Minimum), WithInfo("target", *s.Minimum))
	}
	if s.Maximum == nil {
		s.Maximum = ss.Maximum
	} else if ss.Maximum != nil && *s.Maximum > *ss.Maximum {
		return nil, NewError("maximum constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", *ss.Maximum), WithInfo("target", *s.Maximum))
	}
	// TODO: multipleOf validation
	if ss.MultipleOf != nil {
		// TODO: Disallow multipleOf 0 to avoid division by zero during validation
		s.MultipleOf = ss.MultipleOf
	}
	if s.Enum == nil {
		s.Enum = ss.Enum
	} else if ss.Enum != nil && !IsOverridableEnum(ss.Enum, s.Enum) {
		return nil, NewError("enum constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", ss.Enum), WithInfo("target", s.Enum))
	}
	// TODO: Formats intersection validation
	if s.Format == nil {
		s.Format = ss.Format
	}
	return s, nil
}

func (s *NumberShape) Check() error {
	return nil
}

func (s *NumberShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	for i := 0; i != len(v); i += 2 {
		node := v[i]
		valueNode := v[i+1]
		if node.Value == "minimum" {
			if err := valueNode.Decode(&s.Minimum); err != nil {
				return NewWrappedError("decode minimum", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "maximum" {
			if err := valueNode.Decode(&s.Maximum); err != nil {
				return NewWrappedError("decode maximum", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "format" {
			if err := valueNode.Decode(&s.Format); err != nil {
				return NewWrappedError("decode format", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "enum" {
			enums, err := MakeEnum(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make enum", err, s.Location, WithNodePosition(valueNode))
			}
			s.Enum = enums
		} else if node.Value == "multipleOf" {
			if err := valueNode.Decode(&s.MultipleOf); err != nil {
				return NewWrappedError("decode multipleOf", err, s.Location, WithNodePosition(valueNode))
			}
		} else {
			n, err := MakeNode(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make node", err, s.Location, WithNodePosition(valueNode))
			}
			s.CustomShapeFacets[node.Value] = n
		}
	}
	return nil
}

type LengthFacets struct {
	MaxLength *uint64
	MinLength *uint64
}

type StringFacets struct {
	LengthFacets
	Pattern *regexp.Regexp
}

type StringShape struct {
	BaseShape

	EnumFacets
	StringFacets
}

func (s *StringShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *StringShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *StringShape) Inherit(source Shape) (Shape, error) {
	ss, ok := source.(*StringShape)
	if !ok {
		return nil, NewError("merge shape type mismatch", s.Location, WithPosition(&s.Position), WithInfo("source", source.Base().Type), WithInfo("target", s.Base().Type))
	}
	if s.MinLength == nil {
		s.MinLength = ss.MinLength
	} else if ss.MinLength != nil && *s.MinLength < *ss.MinLength {
		return nil, NewError("minLength constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", *ss.MinLength), WithInfo("target", *s.MinLength))
	}
	if s.MaxLength == nil {
		s.MaxLength = ss.MaxLength
	} else if ss.MaxLength != nil && *s.MaxLength > *ss.MaxLength {
		return nil, NewError("maxLength constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", *ss.MaxLength), WithInfo("target", *s.MaxLength))
	}
	// FIXME: Patterns are merged unconditionally, but ideally they should be validated against intersection of their DFAs
	if s.Pattern == nil {
		s.Pattern = ss.Pattern
	}
	if s.Enum == nil {
		s.Enum = ss.Enum
	} else if ss.Enum != nil && !IsOverridableEnum(ss.Enum, s.Enum) {
		return nil, NewError("enum constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", ss.Enum), WithInfo("target", s.Enum))
	}
	return s, nil
}

func (s *StringShape) Check() error {
	return nil
}

func (s *StringShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	for i := 0; i != len(v); i += 2 {
		node := v[i]
		valueNode := v[i+1]

		if node.Value == "minLength" {
			if err := valueNode.Decode(&s.MinLength); err != nil {
				return NewWrappedError("decode minLength", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "maxLength" {
			if err := valueNode.Decode(&s.MaxLength); err != nil {
				return NewWrappedError("decode maxLength", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "pattern" {
			if valueNode.Tag != "!!str" {
				return NewError("pattern must be string", s.Location, WithNodePosition(valueNode))
			}

			re, err := regexp.Compile(valueNode.Value)
			if err != nil {
				return NewWrappedError("decode pattern", err, s.Location, WithNodePosition(valueNode))
			}
			s.Pattern = re
		} else if node.Value == "enum" {
			enums, err := MakeEnum(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make enum", err, s.Location, WithNodePosition(valueNode))
			}
			s.Enum = enums
		} else {
			n, err := MakeNode(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make node", err, s.Location, WithNodePosition(valueNode))
			}
			s.CustomShapeFacets[node.Value] = n
		}
	}
	return nil
}

type FileFacets struct {
	FileTypes []*Node
}

type FileShape struct {
	BaseShape

	LengthFacets
	FileFacets
}

func (s *FileShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *FileShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *FileShape) Inherit(source Shape) (Shape, error) {
	ss, ok := source.(*FileShape)
	if !ok {
		return nil, NewError("merge shape type mismatch", s.Location, WithPosition(&s.Position), WithInfo("source", source.Base().Type), WithInfo("target", s.Base().Type))
	}
	if s.MinLength == nil {
		s.MinLength = ss.MinLength
	} else if ss.MinLength != nil && *s.MinLength < *ss.MinLength {
		return nil, NewError("minLength constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", *ss.MinLength), WithInfo("target", *s.MinLength))
	}
	if s.MaxLength == nil {
		s.MaxLength = ss.MaxLength
	} else if ss.MaxLength != nil && *s.MaxLength > *ss.MaxLength {
		return nil, NewError("maxLength constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", *ss.MaxLength), WithInfo("target", *s.MaxLength))
	}
	// TODO: FileTypes intersection validation
	if s.FileTypes == nil {
		s.FileTypes = ss.FileTypes
	}
	return s, nil
}

func (s *FileShape) Check() error {
	return nil
}

func (s *FileShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	for i := 0; i != len(v); i += 2 {
		node := v[i]
		valueNode := v[i+1]

		if node.Value == "minLength" {
			if err := valueNode.Decode(&s.MinLength); err != nil {
				return NewWrappedError("decode minLength", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "maxLength" {
			if err := valueNode.Decode(&s.MaxLength); err != nil {
				return NewWrappedError("decode maxLength", err, s.Location, WithNodePosition(valueNode))
			}
		} else if node.Value == "fileTypes" {
			if valueNode.Kind != yaml.SequenceNode {
				return NewError("fileTypes must be sequence node", s.Location, WithNodePosition(valueNode))
			}
			var fileTypes []*Node = make([]*Node, len(valueNode.Content))
			for i, v := range valueNode.Content {
				if v.Tag != "!!str" {
					return NewError("member of fileTypes must be string", s.Location, WithNodePosition(v))
				}
				n, err := MakeNode(v, s.Location)
				if err != nil {
					return NewWrappedError("make node fileTypes", err, s.Location, WithNodePosition(v))
				}
				fileTypes[i] = n
			}
			s.FileTypes = fileTypes
		} else {
			n, err := MakeNode(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make node", err, s.Location, WithNodePosition(valueNode))
			}
			s.CustomShapeFacets[node.Value] = n
		}
	}
	return nil
}

type BooleanShape struct {
	BaseShape

	EnumFacets
}

func (s *BooleanShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *BooleanShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *BooleanShape) Inherit(source Shape) (Shape, error) {
	ss, ok := source.(*BooleanShape)
	if !ok {
		return nil, NewError("merge shape type mismatch", s.Location, WithPosition(&s.Position), WithInfo("source", source.Base().Type), WithInfo("target", s.Base().Type))
	}
	if s.Enum == nil {
		s.Enum = ss.Enum
	} else if ss.Enum != nil && !IsOverridableEnum(ss.Enum, s.Enum) {
		return nil, NewError("enum constraint violation", s.Location, WithPosition(&s.Position), WithInfo("source", ss.Enum), WithInfo("target", s.Enum))
	}
	return s, nil
}

func (s *BooleanShape) Check() error {
	return nil
}

func (s *BooleanShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	for i := 0; i != len(v); i += 2 {
		node := v[i]
		valueNode := v[i+1]

		if node.Value == "enum" {
			enums, err := MakeEnum(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make enum", err, s.Location, WithNodePosition(valueNode))
			}
			s.Enum = enums
		} else {
			n, err := MakeNode(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make node", err, s.Location, WithNodePosition(valueNode))
			}
			s.CustomShapeFacets[node.Value] = n
		}
	}

	return nil
}

type DateTimeShape struct {
	BaseShape

	FormatFacets
}

func (s *DateTimeShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *DateTimeShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *DateTimeShape) Inherit(source Shape) (Shape, error) {
	ss, ok := source.(*DateTimeShape)
	if !ok {
		return nil, NewError("merge shape type mismatch", s.Location, WithPosition(&s.Position), WithInfo("source", source.Base().Type), WithInfo("target", s.Base().Type))
	}
	// TODO: Formats intersection
	if s.Format == nil {
		s.Format = ss.Format
	}
	return s, nil
}

func (s *DateTimeShape) Check() error {
	return nil
}

func (s *DateTimeShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	for i := 0; i != len(v); i += 2 {
		node := v[i]
		valueNode := v[i+1]
		if node.Value == "format" {
			if err := valueNode.Decode(&s.Format); err != nil {
				return NewWrappedError("decode format", err, s.Location, WithNodePosition(valueNode))
			}
		} else {
			n, err := MakeNode(valueNode, s.Location)
			if err != nil {
				return NewWrappedError("make node", err, s.Location, WithNodePosition(valueNode))
			}
			s.CustomShapeFacets[node.Value] = n
		}
	}
	return nil
}

type DateTimeOnlyShape struct {
	BaseShape
}

func (s *DateTimeOnlyShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *DateTimeOnlyShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *DateTimeOnlyShape) Inherit(source Shape) (Shape, error) {
	return s, nil
}

func (s *DateTimeOnlyShape) Check() error {
	return nil
}

func (s *DateTimeOnlyShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	return nil
}

type DateOnlyShape struct {
	BaseShape
}

func (s *DateOnlyShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *DateOnlyShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *DateOnlyShape) Inherit(source Shape) (Shape, error) {
	return s, nil
}

func (s *DateOnlyShape) Check() error {
	return nil
}

func (s *DateOnlyShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	return nil
}

type TimeOnlyShape struct {
	BaseShape
}

func (s *TimeOnlyShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *TimeOnlyShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *TimeOnlyShape) Inherit(source Shape) (Shape, error) {
	return s, nil
}

func (s *TimeOnlyShape) Check() error {
	return nil
}

func (s *TimeOnlyShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	return nil
}

type AnyShape struct {
	BaseShape
}

func (s *AnyShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *AnyShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *AnyShape) Inherit(source Shape) (Shape, error) {
	return s, nil
}

func (s *AnyShape) Check() error {
	return nil
}

func (s *AnyShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	return nil
}

type NilShape struct {
	BaseShape
}

func (s *NilShape) Base() *BaseShape {
	return &s.BaseShape
}

func (s *NilShape) Clone() Shape {
	c := *s
	c.Id = GenerateShapeId()
	return &c
}

func (s *NilShape) Inherit(source Shape) (Shape, error) {
	return s, nil
}

func (s *NilShape) Check() error {
	return nil
}

func (s *NilShape) UnmarshalYAMLNodes(v []*yaml.Node) error {
	return nil
}
