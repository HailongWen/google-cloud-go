/*
Copyright 2017 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spanner

import (
	"bytes"
	"math"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	proto3 "google.golang.org/protobuf/types/known/structpb"
)

func TestConvertParams(t *testing.T) {
	st := Statement{
		SQL:    "SELECT id from t_foo WHERE col = @var",
		Params: map[string]interface{}{"var": nil},
	}
	var (
		t1, _ = time.Parse(time.RFC3339Nano, "2016-11-15T15:04:05.999999999Z")
		// Boundaries
		t2, _ = time.Parse(time.RFC3339Nano, "0001-01-01T00:00:00.000000000Z")
		t3, _ = time.Parse(time.RFC3339Nano, "9999-12-31T23:59:59.999999999Z")
		d1, _ = civil.ParseDate("2016-11-15")
		// Boundaries
		d2, _ = civil.ParseDate("0001-01-01")
		d3, _ = civil.ParseDate("9999-12-31")
	)

	type staticStruct struct {
		Field int `spanner:"field"`
	}
	type CustomInt int64
	type staticStructWithCustomType struct {
		Field CustomInt `spanner:"field"`
	}

	var (
		s1 = staticStruct{10}
		s2 = staticStruct{20}
		s3 = staticStructWithCustomType{30}
	)

	for _, test := range []struct {
		val       interface{}
		wantField *proto3.Value
		wantType  *sppb.Type
	}{
		// bool
		{true, boolProto(true), boolType()},
		{NullBool{true, true}, boolProto(true), boolType()},
		{NullBool{true, false}, nullProto(), boolType()},
		{[]bool(nil), nullProto(), listType(boolType())},
		{[]bool{}, listProto(), listType(boolType())},
		{[]bool{true, false}, listProto(boolProto(true), boolProto(false)), listType(boolType())},
		{[]NullBool(nil), nullProto(), listType(boolType())},
		{[]NullBool{}, listProto(), listType(boolType())},
		{[]NullBool{{true, true}, {}}, listProto(boolProto(true), nullProto()), listType(boolType())},
		// int
		{int(1), intProto(1), intType()},
		{[]int(nil), nullProto(), listType(intType())},
		{[]int{}, listProto(), listType(intType())},
		{[]int{1, 2}, listProto(intProto(1), intProto(2)), listType(intType())},
		// int64
		{int64(1), intProto(1), intType()},
		{NullInt64{5, true}, intProto(5), intType()},
		{NullInt64{5, false}, nullProto(), intType()},
		{[]int64(nil), nullProto(), listType(intType())},
		{[]int64{}, listProto(), listType(intType())},
		{[]int64{1, 2}, listProto(intProto(1), intProto(2)), listType(intType())},
		{[]NullInt64(nil), nullProto(), listType(intType())},
		{[]NullInt64{}, listProto(), listType(intType())},
		{[]NullInt64{{1, true}, {}}, listProto(intProto(1), nullProto()), listType(intType())},
		// float64
		{0.0, floatProto(0.0), floatType()},
		{math.Inf(1), floatProto(math.Inf(1)), floatType()},
		{math.Inf(-1), floatProto(math.Inf(-1)), floatType()},
		{math.NaN(), floatProto(math.NaN()), floatType()},
		{NullFloat64{2.71, true}, floatProto(2.71), floatType()},
		{NullFloat64{1.41, false}, nullProto(), floatType()},
		{[]float64(nil), nullProto(), listType(floatType())},
		{[]float64{}, listProto(), listType(floatType())},
		{[]float64{2.72, math.Inf(1)}, listProto(floatProto(2.72), floatProto(math.Inf(1))), listType(floatType())},
		{[]NullFloat64(nil), nullProto(), listType(floatType())},
		{[]NullFloat64{}, listProto(), listType(floatType())},
		{[]NullFloat64{{2.72, true}, {}}, listProto(floatProto(2.72), nullProto()), listType(floatType())},
		// float32
		{float32(0.0), float32Proto(0.0), float32Type()},
		{float32(math.Inf(1)), float32Proto(float32(math.Inf(1))), float32Type()},
		{float32(math.Inf(-1)), float32Proto(float32(math.Inf(-1))), float32Type()},
		{float32(math.NaN()), float32Proto(float32(math.NaN())), float32Type()},
		{NullFloat32{3.14, true}, float32Proto(3.14), float32Type()},
		{NullFloat32{-99.99, false}, nullProto(), float32Type()},
		{[]float32(nil), nullProto(), listType(float32Type())},
		{[]float32{}, listProto(), listType(float32Type())},
		{[]float32{3.14, float32(math.Inf(1))}, listProto(float32Proto(3.14), float32Proto(float32(math.Inf(1)))), listType(float32Type())},
		{[]NullFloat32(nil), nullProto(), listType(float32Type())},
		{[]NullFloat32{}, listProto(), listType(float32Type())},
		{[]NullFloat32{{3.14, true}, {}}, listProto(float32Proto(3.14), nullProto()), listType(float32Type())},
		// string
		{"", stringProto(""), stringType()},
		{"foo", stringProto("foo"), stringType()},
		{NullString{"bar", true}, stringProto("bar"), stringType()},
		{NullString{"bar", false}, nullProto(), stringType()},
		{[]string(nil), nullProto(), listType(stringType())},
		{[]string{}, listProto(), listType(stringType())},
		{[]string{"foo", "bar"}, listProto(stringProto("foo"), stringProto("bar")), listType(stringType())},
		{[]NullString(nil), nullProto(), listType(stringType())},
		{[]NullString{}, listProto(), listType(stringType())},
		{[]NullString{{"foo", true}, {}}, listProto(stringProto("foo"), nullProto()), listType(stringType())},
		// bytes
		{[]byte{}, bytesProto([]byte{}), bytesType()},
		{[]byte{1, 2, 3}, bytesProto([]byte{1, 2, 3}), bytesType()},
		{[]byte(nil), nullProto(), bytesType()},
		{[][]byte(nil), nullProto(), listType(bytesType())},
		{[][]byte{}, listProto(), listType(bytesType())},
		{[][]byte{{1}, []byte(nil)}, listProto(bytesProto([]byte{1}), nullProto()), listType(bytesType())},
		// date
		{d1, dateProto(d1), dateType()},
		{NullDate{civil.Date{}, false}, nullProto(), dateType()},
		{[]civil.Date(nil), nullProto(), listType(dateType())},
		{[]civil.Date{}, listProto(), listType(dateType())},
		{[]civil.Date{d1, d2, d3}, listProto(dateProto(d1), dateProto(d2), dateProto(d3)), listType(dateType())},
		{[]NullDate{{d2, true}, {}}, listProto(dateProto(d2), nullProto()), listType(dateType())},
		// timestamp
		{t1, timeProto(t1), timeType()},
		{NullTime{}, nullProto(), timeType()},
		{[]time.Time(nil), nullProto(), listType(timeType())},
		{[]time.Time{}, listProto(), listType(timeType())},
		{[]time.Time{t1, t2, t3}, listProto(timeProto(t1), timeProto(t2), timeProto(t3)), listType(timeType())},
		{[]NullTime{{t2, true}, {}}, listProto(timeProto(t2), nullProto()), listType(timeType())},
		// uuid
		{uuid1, uuidProto(uuid1), uuidType()},
		{NullUUID{uuid1, false}, nullProto(), uuidType()},
		{[]uuid.UUID(nil), nullProto(), listType(uuidType())},
		{[]uuid.UUID{}, listProto(), listType(uuidType())},
		{[]uuid.UUID{uuid1, uuid2}, listProto(uuidProto(uuid1), uuidProto(uuid2)), listType(uuidType())},
		{[]NullUUID{{uuid1, true}, {}}, listProto(uuidProto(uuid1), nullProto()), listType(uuidType())},
		// Struct
		{
			s1,
			listProto(intProto(10)),
			structType(mkField("field", intType())),
		},
		{
			s3,
			listProto(intProto(30)),
			structType(mkField("field", intType())),
		},
		{
			(*struct {
				F1 civil.Date `spanner:""`
				F2 bool
			})(nil),
			nullProto(),
			structType(
				mkField("", dateType()),
				mkField("F2", boolType())),
		},
		// Array-of-struct
		{
			[]staticStruct{s1, s2},
			listProto(listProto(intProto(10)), listProto(intProto(20))),
			listType(structType(mkField("field", intType()))),
		},
		// Untyped null
		{
			nil,
			nullProto(),
			nil,
		},
	} {
		st.Params["var"] = test.val
		gotParams, gotParamTypes, gotErr := st.convertParams()
		if gotErr != nil {
			t.Error(gotErr)
			continue
		}
		gotParamField := gotParams.Fields["var"]
		if !proto.Equal(gotParamField, test.wantField) {
			// handle NaN
			gotParamFieldText, err := prototext.Marshal(gotParamField)
			if err != nil {
				t.Fatal(err)
			}
			wantParamFieldText, err := prototext.Marshal(test.wantField)
			if err != nil {
				t.Fatal(err)
			}
			if test.wantType.Code == floatType().Code && bytes.Equal(gotParamFieldText, wantParamFieldText) {
				continue
			}
			if test.wantType.Code == float32Type().Code && bytes.Equal(gotParamFieldText, wantParamFieldText) {
				continue
			}
			t.Errorf("%#v:\n got: %v\nwant: %v\n", test.val, gotParamField, test.wantField)
		}
		gotParamType := gotParamTypes["var"]
		if !proto.Equal(gotParamType, test.wantType) {
			t.Errorf("%#v:\n got: %v\nwant: %v\n", test.val, gotParamType, test.wantField)
		}
	}
}

func TestNewStatement(t *testing.T) {
	s := NewStatement("query")
	if got, want := s.SQL, "query"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
