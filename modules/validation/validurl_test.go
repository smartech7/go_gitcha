// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"testing"

	"github.com/go-macaron/binding"
)

var urlValidationTestCases = []validationTestCase{
	{
		description: "Empty URL",
		data: TestForm{
			URL: "",
		},
		expectedErrors: binding.Errors{},
	},
	{
		description: "URL without port",
		data: TestForm{
			URL: "http://test.lan/",
		},
		expectedErrors: binding.Errors{},
	},
	{
		description: "URL with port",
		data: TestForm{
			URL: "http://test.lan:3000/",
		},
		expectedErrors: binding.Errors{},
	},
	{
		description: "URL with IPv6 address without port",
		data: TestForm{
			URL: "http://[::1]/",
		},
		expectedErrors: binding.Errors{},
	},
	{
		description: "URL with IPv6 address with port",
		data: TestForm{
			URL: "http://[::1]:3000/",
		},
		expectedErrors: binding.Errors{},
	},
	{
		description: "Invalid URL",
		data: TestForm{
			URL: "http//test.lan/",
		},
		expectedErrors: binding.Errors{
			binding.Error{
				FieldNames:     []string{"URL"},
				Classification: binding.ERR_URL,
				Message:        "Url",
			},
		},
	},
	{
		description: "Invalid schema",
		data: TestForm{
			URL: "ftp://test.lan/",
		},
		expectedErrors: binding.Errors{
			binding.Error{
				FieldNames:     []string{"URL"},
				Classification: binding.ERR_URL,
				Message:        "Url",
			},
		},
	},
	{
		description: "Invalid port",
		data: TestForm{
			URL: "http://test.lan:3x4/",
		},
		expectedErrors: binding.Errors{
			binding.Error{
				FieldNames:     []string{"URL"},
				Classification: binding.ERR_URL,
				Message:        "Url",
			},
		},
	},
	{
		description: "Invalid port with IPv6 address",
		data: TestForm{
			URL: "http://[::1]:3x4/",
		},
		expectedErrors: binding.Errors{
			binding.Error{
				FieldNames:     []string{"URL"},
				Classification: binding.ERR_URL,
				Message:        "Url",
			},
		},
	},
}

func Test_ValidURLValidation(t *testing.T) {
	AddBindingRules()

	for _, testCase := range urlValidationTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			performValidationTest(t, testCase)
		})
	}
}
