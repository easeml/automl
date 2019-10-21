---
title: "Schema"
---


# schema
--
    import "."


## Usage

#### func  Load

```go
func Load(input interface{}) (*Schema, Error)
```
Load is.

#### type Category

```go
type Category struct {
	Class   string
	SrcName string
}
```

Category is.

#### func (*Category) Type

```go
func (f *Category) Type() string
```
Type is.

#### type Class

```go
type Class struct {
	Dim     Dim
	SrcName string
}
```

Class is.

#### type ConstDim

```go
type ConstDim struct {
	Value int
}
```

ConstDim is.

#### func (*ConstDim) Equals

```go
func (d *ConstDim) Equals(other Dim) bool
```
Equals is.

#### func (*ConstDim) IsVariable

```go
func (d *ConstDim) IsVariable() bool
```
IsVariable is.

#### func (*ConstDim) IsWildcard

```go
func (d *ConstDim) IsWildcard() bool
```
IsWildcard is.

#### func (*ConstDim) Match

```go
func (d *ConstDim) Match(source Dim, dimMap map[string]Dim) (bool, map[string]Dim)
```
Match is.

#### type Dim

```go
type Dim interface {
	Equals(other Dim) bool
	IsVariable() bool
	IsWildcard() bool
	Match(source Dim, dimMap map[string]Dim) (bool, map[string]Dim)
	// contains filtered or unexported methods
}
```

Dim is.

#### type Error

```go
type Error interface {
	error
	Path() string
}
```

Error is.

#### type Field

```go
type Field interface {
	Type() string
	// contains filtered or unexported methods
}
```

Field is.

#### type Link

```go
type Link struct {
	LBound int
	UBound int
}
```

Link is.

#### func (*Link) IsUnbounded

```go
func (l *Link) IsUnbounded() bool
```
IsUnbounded is.

#### type Node

```go
type Node struct {
	IsSingleton bool
	Fields      map[string]Field
	Links       map[string]*Link
	SrcName     string
}
```

Node is.

#### type Schema

```go
type Schema struct {
	Nodes        map[string]*Node
	Classes      map[string]*Class
	SrcDims      map[string]Dim
	IsUndirected bool
	IsCyclic     bool
	IsFanIn      bool
}
```

Schema is.

#### func (*Schema) Dump

```go
func (s *Schema) Dump() interface{}
```
Dump is.

#### func (*Schema) Match

```go
func (s *Schema) Match(source *Schema, buildMatching bool) (bool, *Schema)
```
Match is.

#### type Tensor

```go
type Tensor struct {
	Dim     []Dim
	SrcDim  []Dim
	SrcName string
}
```

Tensor is.

#### func (*Tensor) Type

```go
func (f *Tensor) Type() string
```
Type is.

#### type VarDim

```go
type VarDim struct {
	Value string
}
```

VarDim is.

#### func (*VarDim) Equals

```go
func (d *VarDim) Equals(other Dim) bool
```
Equals is.

#### func (*VarDim) IsVariable

```go
func (d *VarDim) IsVariable() bool
```
IsVariable is.

#### func (*VarDim) IsWildcard

```go
func (d *VarDim) IsWildcard() bool
```
IsWildcard is.

#### func (*VarDim) Match

```go
func (d *VarDim) Match(source Dim, dimMap map[string]Dim) (bool, map[string]Dim)
```
Match is.
