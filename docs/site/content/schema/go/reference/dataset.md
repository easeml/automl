---
title: "Dataset"
---


## dataset
--
    import "."


### Usage

```go
var LoaderFunctions = map[string]LoaderFunc{
	"tensor":   loadTensor,
	"category": loadCategory,
	"class":    loadClass,
	"links":    loadLinks,
}
```
LoaderFunctions is.

```go
var TypeExtensions = map[string]map[string]string{
	"tensor":   map[string]string{"default": ".ten.npy", "csv": ".ten.csv"},
	"category": map[string]string{"default": ".cat.txt"},
	"class":    map[string]string{"default": ".class.txt"},
	"links":    map[string]string{"default": ".links.csv"},
}
```
TypeExtensions is.

#### func  DumpTarOpener

```go
func DumpTarOpener(opener *TarOpener) (io.Reader, error)
```
DumpTarOpener converts a tar opener to a bytes reader.

#### func  RandomString

```go
func RandomString(size int, chars string) string
```
RandomString returns a random string of given length from a given set of
characters.

#### type Category

```go
type Category struct {
	Name       string
	Categories []string
}
```

Category is.

#### func (Category) Subtype

```go
func (f Category) Subtype() string
```
Subtype is.

#### func (Category) Type

```go
func (f Category) Type() string
```

Type is.

#### type Class

```go
type Class struct {
	Name       string
	Categories []string
}
```

Class is.

#### func (Class) Subtype

```go
func (f Class) Subtype() string
```
Subtype is.

#### func (Class) Type

```go
func (f Class) Type() string
```
Type is.

#### type DataBuffer

```go
type DataBuffer struct {
	Buffer   bytes.Buffer
	ReadOnly bool
}
```

DataBuffer contains data that can be read and written.

#### func (*DataBuffer) Close

```go
func (b *DataBuffer) Close() error
```
Close closes the reader.

#### func (*DataBuffer) Read

```go
func (b *DataBuffer) Read(p []byte) (n int, err error)
```
Read reads bytes from the buffer.

#### func (*DataBuffer) Write

```go
func (b *DataBuffer) Write(p []byte) (n int, err error)
```
Write writes to the buffer.

#### type Dataset

```go
type Dataset struct {
	Directory
	Root string
}
```

Dataset is.

#### func  GenerateFromSchema

```go
func GenerateFromSchema(root string, schema *sch.Schema, sampleNames []string, numNodeInstances int) (*Dataset, error)
```
GenerateFromSchema is.

#### func  Load

```go
func Load(root string, metadataOnly bool, opener Opener) (*Dataset, error)
```
Load is.

#### func (*Dataset) Dump

```go
func (d *Dataset) Dump(root string, opener Opener) error
```
Dump is.

#### func (*Dataset) InferSchema

```go
func (d *Dataset) InferSchema() (*sch.Schema, Error)
```
InferSchema is.

#### type DefaultOpener

```go
type DefaultOpener struct{}
```

DefaultOpener is.

#### func (DefaultOpener) GetDir

```go
func (DefaultOpener) GetDir(root string, relPath string, readOnly bool) ([]string, error)
```
GetDir is.

#### func (DefaultOpener) GetFile

```go
func (DefaultOpener) GetFile(root string, relPath string, readOnly bool, binary bool) (io.ReadWriteCloser, error)
```
GetFile is.

#### type Directory

```go
type Directory struct {
	Name     string
	Children map[string]File
}
```

Directory is.

#### func (Directory) Subtype

```go
func (f Directory) Subtype() string
```
Subtype is.

#### func (Directory) Type

```go
func (f Directory) Type() string
```
Type is.

#### type Error

```go
type Error interface {
	error
	Path() string
}
```

Error is.

#### type File

```go
type File interface {
	Type() string
	Subtype() string
}
```

File is.

#### type InstanceID

```go
type InstanceID struct {
	Node  string
	Index int
}
```

InstanceID is.

#### type Link

```go
type Link struct {
	Src InstanceID
	Dst InstanceID
}
```

Link is.

#### type Links

```go
type Links struct {
	Name  string
	Links map[Link]interface{}
}
```

Links is.

#### func (*Links) IsCyclic

```go
func (f *Links) IsCyclic(undirected bool) bool
```
IsCyclic is.

#### func (*Links) IsFanin

```go
func (f *Links) IsFanin(undirected bool) bool
```
IsFanin is.

#### func (*Links) IsUndirected

```go
func (f *Links) IsUndirected() bool
```
IsUndirected is.

#### func (Links) Subtype

```go
func (f Links) Subtype() string
```
Subtype is.

#### func (Links) Type

```go
func (f Links) Type() string
```
Type is.

#### type LoaderFunc

```go
type LoaderFunc func(string, string, string, Opener, bool, string) (File, error)
```

LoaderFunc is.

#### type Opener

```go
type Opener interface {
	GetFile(root string, relPath string, readOnly bool, binary bool) (io.ReadWriteCloser, error)
	GetDir(root string, relPath string, readOnly bool) ([]string, error)
}
```

Opener is.

#### type TarDir

```go
type TarDir map[string]interface{}
```

TarDir is.

#### type TarOpener

```go
type TarOpener struct {
	Root TarDir
}
```

TarOpener is.

#### func  LoadTarOpener

```go
func LoadTarOpener(reader *tar.Reader) (*TarOpener, error)
```
LoadTarOpener instantiates a new TarOpener from the given tar reader.

#### func  NewTarOpener

```go
func NewTarOpener() *TarOpener
```
NewTarOpener creates a new empty tar opener.

#### func (*TarOpener) GetDir

```go
func (opener *TarOpener) GetDir(root string, relPath string, readOnly bool) ([]string, error)
```
GetDir is.

#### func (*TarOpener) GetFile

```go
func (opener *TarOpener) GetFile(root string, relPath string, readOnly bool, binary bool) (io.ReadWriteCloser, error)
```
GetFile is.

#### type Tensor

```go
type Tensor struct {
	Name       string
	Dimensions []int
	Data       interface{}
}
```

Tensor is.

#### func (Tensor) Subtype

```go
func (f Tensor) Subtype() string
```
Subtype is.

#### func (Tensor) Type

```go
func (f Tensor) Type() string
```
Type is.
