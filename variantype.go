package main

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"reflect"
	"strconv"

	"github.com/mitchellh/hashstructure"
)

type VariantType int

const (
	VariantTypeInt VariantType = iota
	VariantTypeBool
	VariantTypeString
	VariantTypeSlice
	VariantTypeMap
)

type Variant interface {
	hashstructure.Hashable
	GetType() VariantType
	GetIntValue() *int
	GetBoolValue() *bool
	GetStringValue() *string
	GetSliceValue() []Variant
	GetMapValue() map[string]Variant
}

// -----------------------------------------------------------------------------
type VariantIntValue struct {
	value int
}

func NewVariantIntValue(_v int) *VariantIntValue {
	return &VariantIntValue{_v}
}

func (i *VariantIntValue) GetType() VariantType {
	return VariantTypeInt
}

func (i *VariantIntValue) GetIntValue() *int {
	return &i.value
}

func (i *VariantIntValue) GetBoolValue() *bool {
	return nil
}

func (i *VariantIntValue) GetStringValue() *string {
	return nil
}

func (i *VariantIntValue) GetSliceValue() []Variant {
	return nil
}

func (s *VariantIntValue) GetMapValue() map[string]Variant {
	return nil
}

func (i *VariantIntValue) Hash() (uint64, error) {
	s := strconv.Itoa(i.value)
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64(), nil
}

// -----------------------------------------------------------------------------
type VariantBoolValue struct {
	value bool
}

func NewVariantBoolValue(_v bool) *VariantBoolValue {
	return &VariantBoolValue{_v}
}

func (b *VariantBoolValue) GetType() VariantType {
	return VariantTypeBool
}

func (b *VariantBoolValue) GetIntValue() *int {
	return nil
}

func (b *VariantBoolValue) GetBoolValue() *bool {
	return &b.value
}

func (b *VariantBoolValue) GetStringValue() *string {
	return nil
}

func (b *VariantBoolValue) GetSliceValue() []Variant {
	return nil
}

func (b *VariantBoolValue) GetMapValue() map[string]Variant {
	return nil
}

func (b *VariantBoolValue) Hash() (uint64, error) {
	s := "false"
	if b.value {
		s = "true"
	}
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64(), nil
}

// -----------------------------------------------------------------------------
type VariantStringValue struct {
	value string
}

func NewVariantStringValue(_v string) *VariantStringValue {
	return &VariantStringValue{_v}
}

func (s *VariantStringValue) GetType() VariantType {
	return VariantTypeString
}

func (s *VariantStringValue) GetIntValue() *int {
	return nil
}

func (s *VariantStringValue) GetBoolValue() *bool {
	return nil
}

func (s *VariantStringValue) GetStringValue() *string {
	return &s.value
}

func (s *VariantStringValue) GetSliceValue() []Variant {
	return nil
}

func (s *VariantStringValue) GetMapValue() map[string]Variant {
	return nil
}

func (s *VariantStringValue) Hash() (uint64, error) {
	h := fnv.New64a()
	h.Write([]byte(s.value))
	return h.Sum64(), nil
}

// -----------------------------------------------------------------------------
type VariantSliceValue struct {
	value []Variant
}

func NewVariantSliceValue(_v []Variant) *VariantSliceValue {
	return &VariantSliceValue{_v}
}

func (s *VariantSliceValue) GetType() VariantType {
	return VariantTypeSlice
}

func (s *VariantSliceValue) GetIntValue() *int {
	return nil
}

func (s *VariantSliceValue) GetBoolValue() *bool {
	return nil
}

func (s *VariantSliceValue) GetStringValue() *string {
	return nil
}

func (s *VariantSliceValue) GetSliceValue() []Variant {
	return s.value
}

func (s *VariantSliceValue) GetMapValue() map[string]Variant {
	return nil
}

func (s *VariantSliceValue) Hash() (uint64, error) {
	h := fnv.New64a()

	for _, v := range s.value {
		a, err := v.Hash()
		if err != nil {
			return 0, err
		}
		binary.Write(h, binary.LittleEndian, a)
	}

	return h.Sum64(), nil
}

// -----------------------------------------------------------------------------
type VariantMapValue struct {
	value map[string]Variant
}

func NewVariantMapValue(_v map[string]Variant) *VariantMapValue {
	return &VariantMapValue{_v}
}

func (s *VariantMapValue) GetType() VariantType {
	return VariantTypeMap
}

func (s *VariantMapValue) GetIntValue() *int {
	return nil
}

func (s *VariantMapValue) GetBoolValue() *bool {
	return nil
}

func (s *VariantMapValue) GetStringValue() *string {
	return nil
}

func (s *VariantMapValue) GetSliceValue() []Variant {
	return nil
}

func (s *VariantMapValue) GetMapValue() map[string]Variant {
	return s.value
}

func (s *VariantMapValue) Hash() (uint64, error) {
	h := fnv.New64a()

	for k, v := range s.value {
		_, err := h.Write([]byte(k))
		if err != nil {
			return 0, err
		}

		a, err := v.Hash()
		if err != nil {
			return 0, err
		}
		binary.Write(h, binary.LittleEndian, a)
	}

	return h.Sum64(), nil
}

// -----------------------------------------------------------------------------
type VariantYamlUnmarshaled struct {
	value Variant
}

func NewVariantFromAnyType(anytype interface{}) (Variant, error) {
	var value Variant

	switch reflect.TypeOf(anytype).Kind() {
	case reflect.Int:
		value = NewVariantIntValue(int(reflect.ValueOf(anytype).Int()))
	case reflect.Bool:
		value = NewVariantBoolValue(reflect.ValueOf(anytype).Bool())
	case reflect.String:
		value = NewVariantStringValue(reflect.ValueOf(anytype).String())
	case reflect.Slice:
		reflectedVal := reflect.ValueOf(anytype)
		anyslice := make([]Variant, 0, reflectedVal.Len())

		for i := 0; i < reflectedVal.Len(); i++ {
			svalue, lerr := NewVariantFromAnyType(reflectedVal.Index(i).Interface())
			if lerr != nil {
				return nil, lerr
			}
			anyslice = append(anyslice, svalue)
		}

		value = NewVariantSliceValue(anyslice)
	case reflect.Map:
		reflectedVal := reflect.ValueOf(anytype)
		anymap := make(map[string]Variant)
		mapIter := reflectedVal.MapRange()

		for mapIter.Next() {
			if mapIter.Key().Kind() != reflect.String {
				return nil, fmt.Errorf("map keys must be strings")
			}

			svalue, lerr := NewVariantFromAnyType(mapIter.Value().Interface())
			if lerr != nil {
				return nil, lerr
			}

			anymap[mapIter.Key().String()] = svalue
		}

		value = NewVariantMapValue(anymap)
	default:
		return nil, fmt.Errorf("unsupported type for unmarshal")
	}

	return value, nil
}

func (s *VariantYamlUnmarshaled) GetType() VariantType {
	return s.value.GetType()
}

func (s *VariantYamlUnmarshaled) GetIntValue() *int {
	return s.value.GetIntValue()
}

func (s *VariantYamlUnmarshaled) GetBoolValue() *bool {
	return s.value.GetBoolValue()
}

func (s *VariantYamlUnmarshaled) GetStringValue() *string {
	return s.value.GetStringValue()
}

func (s *VariantYamlUnmarshaled) GetSliceValue() []Variant {
	return s.value.GetSliceValue()
}

func (s *VariantYamlUnmarshaled) GetMapValue() map[string]Variant {
	return s.value.GetMapValue()
}

func (s *VariantYamlUnmarshaled) Hash() (uint64, error) {
	return s.value.Hash()
}

func (s *VariantYamlUnmarshaled) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var anytype interface{}
	unmarshal(&anytype)

	value, lerr := NewVariantFromAnyType(anytype)
	if lerr != nil {
		return lerr
	}

	s.value = value

	return nil
}
