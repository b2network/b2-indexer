package model_test

import (
	"reflect"
	"testing"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/pkg/utils"
)

func TestValidateSinohopeColumn(t *testing.T) {
	var d model.Sinohope
	dc := model.Sinohope{}.Column()

	dFields := reflect.TypeOf(d)
	dcValues := reflect.ValueOf(dc)

	dJSONTags := []string{}
	for i := 0; i < dFields.NumField(); i++ {
		dField := dFields.Field(i)
		dJSONTag := dField.Tag.Get("json")
		dJSONTags = append(dJSONTags, dJSONTag)
	}

	for i := 0; i < dcValues.NumField(); i++ {
		dcValue := dcValues.Field(i).String()
		if !utils.StrInArray(dJSONTags, dcValue) {
			t.Fatalf("column field %s not found in sinohope %s", dcValue, dJSONTags)
		}
	}
}
