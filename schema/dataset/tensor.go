package dataset

import (
	"encoding/csv"
	"path"
	"strconv"

	"github.com/kshedden/gonpy"
)

// Tensor is.
type Tensor struct {
	Name       string
	Dimensions []int
	Data       interface{}
	subtype    string
}

// Type is.
func (f Tensor) Type() string { return "tensor" }

// Subtype is.
func (f Tensor) Subtype() string { return f.subtype }

func loadTensor(root string, relPath string, name string, opener Opener, metadataOnly bool, subtype string) (File, error) {
	path := path.Join(relPath, name+TypeExtensions["tensor"][subtype])
	file, err := opener.GetFile(root, path, true, false)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if subtype == "default" {

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

		return &Tensor{Name: name, Dimensions: reader.Shape, Data: data, subtype: subtype}, nil

	} else if subtype == "csv" {

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			return nil, err
		}
		numLines := len(records)
		if numLines == 0 {
			return nil, &datasetError{err: "Found empty CSV file.", path: path}
		}
		lineLength := len(records[0])
		shape := []int{numLines, lineLength}
		if numLines == 1 {
			shape = []int{lineLength}
		}
		data := make([]float64, numLines*lineLength)
		pos := 0
		for i := range records {
			if lineLength != len(records[i]) {
				return nil, &datasetError{err: "Each row of the CSV file must have the same number of elements.", path: path}
			}
			for j := range records[i] {
				var err error
				data[pos], err = strconv.ParseFloat(records[i][j], 64)
				if err != nil {
					return nil, err
				}
				pos++
			}
		}

		return &Tensor{Name: name, Dimensions: shape, Data: data, subtype: subtype}, nil

	} else {
		return nil, &datasetError{err: "Unknown tensor subtype '" + subtype + "'.", path: path}
	}
}

func (f *Tensor) dump(root string, relPath string, name string, opener Opener) error {
	path := path.Join(relPath, name) + TypeExtensions["tensor"][f.Subtype()]
	file, err := opener.GetFile(root, path, false, false)
	if err != nil {
		return err
	}
	defer file.Close()

	if f.Subtype() == "default" {

		writer, err := gonpy.NewWriter(file)
		if err != nil {
			return err
		}
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

	} else if f.Subtype() == "csv" {

		numLines := 1
		lineLength := f.Dimensions[0]
		if len(f.Dimensions) > 1 {
			numLines = f.Dimensions[0]
			lineLength = f.Dimensions[1]
		}
		pos := 0
		records := make([][]string, numLines)
		for i := 0; i < numLines; i++ {
			records[i] = make([]string, lineLength)
			for j := 0; j < lineLength; j++ {

				if float64data, ok := f.Data.([]float64); ok {
					records[i][j] = strconv.FormatFloat(float64data[pos], 'E', -1, 64)
				} else {
					panic("Unknown data")
				}
				pos++
			}
		}

		writer := csv.NewWriter(file)
		err := writer.WriteAll(records)
		if err != nil {
			return err
		}

	} else {
		return &datasetError{err: "Unknown tensor subtype '" + f.Subtype() + "'.", path: path}
	}

	return nil
}
