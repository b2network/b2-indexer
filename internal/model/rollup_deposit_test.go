package model_test

import (
	"reflect"
	"testing"

	"github.com/b2network/b2-indexer/internal/model"
	"github.com/b2network/b2-indexer/pkg/utils"
)

func TestValidateRollupDepositColumn(t *testing.T) {
	var d model.RollupDeposit
	dc := model.RollupDeposit{}.Column()

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
			t.Fatalf("rollupdepositColumn field %s not found in rollup_deposit %s", dcValue, dJSONTags)
		}
	}
}
