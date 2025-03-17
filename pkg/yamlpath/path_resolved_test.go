/*
 * Copyright 2020 VMware, Inc.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package yamlpath_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

func TestResolvedFind(t *testing.T) {
	y := `---
store:
  vars:
    c: &cat fantasy
    information: &info
      price: 3.14
      weight: 0.5
    another: &another
      price: 8
      test: value
    final: &fb
      random: *cat
  book:
    - category: reference
      author: Nigel Rees
      title: Sayings of the Century
      price: 8.95
    - category: fiction
      author: Evelyn Waugh
      title: Sword of Honour
      price: 12.99
    - category: fiction
      author: Herman Melville
      title: Moby Dick
      isbn: 0-553-21311-3
      price: 8.99
    - category: fiction
      author: J. R. R. Tolkien
      title: The Lord of the Rings
      isbn: 0-395-19395-8
      price: 22.99
    - category: *cat
      <<: *info
      author: Something
      title: idk man
    - category: fiction
      <<: [*another, *info]
      some: thing
    - category: non-fiction
      <<: *fb
      a: b
  bicycle:
    color: red
    price: 19.95
  feather duster:
    price: 9.95
x:
  - y:
    - z: 1
      w: 2
  - y:
    - z: 3
      w: 4
test~: hello world
test: this is a test
`
	var n yaml.Node

	err := yaml.Unmarshal([]byte(y), &n)
	require.NoError(t, err)

	cases := []struct {
		name            string
		path            string
		expectedStrings []string
		expectedPathErr string
		focus           bool // if true, run only tests with focus set to true
	}{
		{
			name: "test using the value of an alias",
			path: "$.store.book[?(@.category=='fantasy')]",
			expectedStrings: []string{`category: *cat
!!merge <<: *info
author: Something
title: idk man
`,
			},
			expectedPathErr: "",
		},
		{
			name: "test using a key within a merge",
			path: "$.store.book[?(@.price==3.14)]",
			expectedStrings: []string{`category: *cat
!!merge <<: *info
author: Something
title: idk man
`,
			},
			expectedPathErr: "",
		},
		{
			name: "test using a sequence of aliases in a merge",
			path: "$.store.book[?(@.price==8)]",
			expectedStrings: []string{`category: fiction
!!merge <<: [*another, *info]
some: thing
`,
			},
		},
		{

			name: "test using an alias in a merge",
			path: "$.store.book[?(@.random=='fantasy')]",
			expectedStrings: []string{`category: non-fiction
!!merge <<: *fb
a: b
`,
			},
		},
	}

	focussed := false
	for _, tc := range cases {
		if tc.focus {
			focussed = true
			break
		}
	}

	for _, tc := range cases {
		if focussed && !tc.focus {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			p, err := yamlpath.NewPath(tc.path)
			if err != nil {
				require.Nil(t, p)
			}
			if tc.expectedPathErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedPathErr)
				return
			}

			actual, err := p.Find(&n)
			require.NoError(t, err)

			actualStrings := []string{}
			for _, a := range actual {
				var buf bytes.Buffer
				e := yaml.NewEncoder(&buf)
				e.SetIndent(2)

				err = e.Encode(a)
				require.NoError(t, err)
				e.Close()
				actualStrings = append(actualStrings, buf.String())
			}

			require.Equal(t, tc.expectedStrings, actualStrings)
		})
	}

	if focussed {
		t.Fatalf("testcase(s) still focussed")
	}
}
