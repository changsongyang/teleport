// Copyright 2024 Gravitational, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: teleport/decision/v1alpha1/enforcement_feature.proto

package decisionpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// EnforcementFeature represents PEP (Policy Enforcement Point) features.
type EnforcementFeature int32

const (
	// ENFORCEMENT_FEATURE_UNSPECIFIED is the default/unspecified value for
	// EnforcementFeature. Asserting this feature has no effect.
	EnforcementFeature_ENFORCEMENT_FEATURE_UNSPECIFIED EnforcementFeature = 0
)

// Enum value maps for EnforcementFeature.
var (
	EnforcementFeature_name = map[int32]string{
		0: "ENFORCEMENT_FEATURE_UNSPECIFIED",
	}
	EnforcementFeature_value = map[string]int32{
		"ENFORCEMENT_FEATURE_UNSPECIFIED": 0,
	}
)

func (x EnforcementFeature) Enum() *EnforcementFeature {
	p := new(EnforcementFeature)
	*p = x
	return p
}

func (x EnforcementFeature) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EnforcementFeature) Descriptor() protoreflect.EnumDescriptor {
	return file_teleport_decision_v1alpha1_enforcement_feature_proto_enumTypes[0].Descriptor()
}

func (EnforcementFeature) Type() protoreflect.EnumType {
	return &file_teleport_decision_v1alpha1_enforcement_feature_proto_enumTypes[0]
}

func (x EnforcementFeature) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EnforcementFeature.Descriptor instead.
func (EnforcementFeature) EnumDescriptor() ([]byte, []int) {
	return file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDescGZIP(), []int{0}
}

var File_teleport_decision_v1alpha1_enforcement_feature_proto protoreflect.FileDescriptor

const file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDesc = "" +
	"\n" +
	"4teleport/decision/v1alpha1/enforcement_feature.proto\x12\x1ateleport.decision.v1alpha1*9\n" +
	"\x12EnforcementFeature\x12#\n" +
	"\x1fENFORCEMENT_FEATURE_UNSPECIFIED\x10\x00BZZXgithub.com/gravitational/teleport/api/gen/proto/go/teleport/decision/v1alpha1;decisionpbb\x06proto3"

var (
	file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDescOnce sync.Once
	file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDescData []byte
)

func file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDescGZIP() []byte {
	file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDescOnce.Do(func() {
		file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDesc), len(file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDesc)))
	})
	return file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDescData
}

var file_teleport_decision_v1alpha1_enforcement_feature_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_teleport_decision_v1alpha1_enforcement_feature_proto_goTypes = []any{
	(EnforcementFeature)(0), // 0: teleport.decision.v1alpha1.EnforcementFeature
}
var file_teleport_decision_v1alpha1_enforcement_feature_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_teleport_decision_v1alpha1_enforcement_feature_proto_init() }
func file_teleport_decision_v1alpha1_enforcement_feature_proto_init() {
	if File_teleport_decision_v1alpha1_enforcement_feature_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDesc), len(file_teleport_decision_v1alpha1_enforcement_feature_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_teleport_decision_v1alpha1_enforcement_feature_proto_goTypes,
		DependencyIndexes: file_teleport_decision_v1alpha1_enforcement_feature_proto_depIdxs,
		EnumInfos:         file_teleport_decision_v1alpha1_enforcement_feature_proto_enumTypes,
	}.Build()
	File_teleport_decision_v1alpha1_enforcement_feature_proto = out.File
	file_teleport_decision_v1alpha1_enforcement_feature_proto_goTypes = nil
	file_teleport_decision_v1alpha1_enforcement_feature_proto_depIdxs = nil
}
