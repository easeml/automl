package schema

// Link is.
type Link struct {
	LBound int
	UBound int
}

// IsUnbounded is.
func (l *Link) IsUnbounded() bool { return l.UBound == 0 }

func (l *Link) match(source *Link) bool {

	if source == nil {
		return false
	}

	if l.LBound > source.LBound {
		return false
	}
	if l.UBound == 0 {
		return true
	} else if source.UBound == 0 || l.UBound < source.UBound {
		return false
	}
	return true
}

func loadLink(input interface{}) (result *Link, err *schemaError) {

	if intDim, ok := getInt(input); ok {

		if intDim < 1 {
			err = &schemaError{err: "Link dimension must be a positive integer."}
			return
		}

		result = &Link{LBound: intDim, UBound: intDim}
		return

	} else if listDim, ok := input.([]interface{}); ok {

		if len(listDim) != 2 {
			err = &schemaError{err: "Link dimension must be a list of two elements representing the upper and lower bound."}
			return
		}

		lBound, ok := getInt(listDim[0])
		if ok == false || lBound < 0 {
			err = &schemaError{err: "Link lower bound must be a non-negative integer."}
			return
		}

		uBound, ok := getInt(listDim[1])
		if ok == false {
			uBoundStr, ok := listDim[1].(string)
			if ok == false || uBoundStr != "inf" {
				err = &schemaError{err: "Link upper bound must be a positive integer or 'inf'."}
				return
			}
			uBound = 0

		} else if uBound <= 0 {
			err = &schemaError{err: "Link upper bound must be a positive integer or 'inf'."}
			return
		}

		if uBound != 0 && lBound > uBound {
			err = &schemaError{err: "Link lower bound cannot be greater than the upper bound."}
			return
		}

		result = &Link{LBound: lBound, UBound: uBound}
		return

	} else {
		err = &schemaError{err: "Link dimension must be either a positive integer or a two-element list."}
		return
	}
}

func (l *Link) dump() interface{} {
	if l.UBound == 0 {
		return []interface{}{l.LBound, "inf"}
	}
	return []interface{}{l.LBound, l.UBound}
}
