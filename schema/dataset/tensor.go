package dataset

import (
	"path"

	"github.com/kshedden/gonpy"
)

// Tensor is.
type Tensor struct {
	Name       string
	Dimensions []int
	Data       interface{}
}

// Type is.
func (f Tensor) Type() string { return "tensor" }

func loadTensor(root string, relPath string, name string, opener Opener, metadataOnly bool) (*Tensor, error) {
	path := path.Join(relPath, name+TypeExtensions["tensor"])
	file, err := opener.GetFile(root, path, true, false)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader, err := gonpy.NewReader(file)
	var data interface{}

	if reader.Dtype != "f8" {
		return nil, &datasetError{err: "Tensor datatype must be float64.", path: path}
	}

	if metadataOnly == false {
		switch reader.Dtype {
		case "f8":
			data, err = reader.GetFloat64()
			if err != nil {
				return nil, err
			}
		case "f4":
			data, err = reader.GetFloat32()
			if err != nil {
				return nil, err
			}
		default:
			return nil, &datasetError{err: "Tensor datatype must be float64.", path: path}
		}
	}

	return &Tensor{Name: name, Dimensions: reader.Shape, Data: data}, nil
}

func (f *Tensor) dump(root string, relPath string, name string, opener Opener) error {
	path := path.Join(relPath, name) + TypeExtensions["tensor"]
	file, err := opener.GetFile(root, path, false, false)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := gonpy.NewWriter(file)
	writer.Shape = f.Dimensions

	if f.Data == nil {
		err := writer.WriteFloat64([]float64{})
		if err != nil {
			return err
		}
	} else if float64data, ok := f.Data.([]float64); ok {
		err := writer.WriteFloat64(float64data)
		if err != nil {
			return err
		}
	} else if float32data, ok := f.Data.([]float32); ok {
		err := writer.WriteFloat32(float32data)
		if err != nil {
			return err
		}
	} else {
		panic("Unknown data")
	}

	return nil
}
