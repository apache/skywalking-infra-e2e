package template

import (
	"reflect"
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		arg0           int
		arg1           int
		compareResults []bool
	}{
		{
			5, 10,
			[]bool{5 < 10, 5 <= 10, 5 >= 10, 5 > 10},
		},
		{
			10, 10,
			[]bool{10 < 10, 10 <= 10, 10 >= 10, 10 > 10},
		},
		{
			15, 10,
			[]bool{15 < 10, 15 <= 10, 15 >= 10, 15 > 10},
		},
	}

	type compare func(ar0, ar1 reflect.Value) (interface{}, error)
	compareNames := []string{
		"lt", "le", "ge", "gt",
	}
	compares := []compare{
		lt, le, ge, gt,
	}

	for _, data := range tests {
		for inx, comp := range compares {
			res, err := comp(reflect.ValueOf(data.arg0), reflect.ValueOf(data.arg1))
			validateSuccess := false
			// need be true
			if err == nil {
				if data.compareResults[inx] {
					validateSuccess = data.arg0 == int(res.(reflect.Value).Int())
				} else {
					validateSuccess = reflect.TypeOf(res).Kind() == reflect.String
				}
			} else {
				res = err
			}

			if !validateSuccess {
				t.Errorf("execute error, %d %s %d error: %v", data.arg0, compareNames[inx], data.arg1, res)
			}
		}
	}
}
