// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Code generated by beats/dev-tools/cmd/asset/asset.go - DO NOT EDIT.

package kubernetes

import (
	"github.com/elastic/beats/v7/libbeat/asset"
)

func init() {
	if err := asset.SetFields("metricbeat", "kubernetes", asset.ModuleFieldsPri, AssetKubernetes); err != nil {
		panic(err)
	}
}

// AssetKubernetes returns asset data.
// This is the base64 encoded zlib format compressed contents of module/kubernetes.
func AssetKubernetes() string {
	return "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"
}
