package model_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/pkg/utils"
)

func TestValidateEpsColumn(t *testing.T) {
	var d model.Eps
	dc := model.Eps{}.Column()

	dFields := reflect.TypeOf(d)
	dcValues := reflect.ValueOf(dc)

	dJSONTags := []string{}
	for i := 0; i < dFields.NumField(); i++ {
		dField := dFields.Field(i)
		dJSONTag := dField.Tag.Get("json")
		dJSONTags = append(dJSONTags, dJSONTag)
	}
	fmt.Println(dJSONTags)
	for i := 0; i < dcValues.NumField(); i++ {
		dcValue := dcValues.Field(i).String()
		if dcValue == "id" {
			continue
		}
		if !utils.StrInArray(dJSONTags, dcValue) {
			t.Fatalf("epsColumn field %s not found in eps %s", dcValue, dJSONTags)
		}
	}
}
