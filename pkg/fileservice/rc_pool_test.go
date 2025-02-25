// Copyright 2022 Matrix Origin
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

package fileservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRCPool(t *testing.T) {
	pool := NewRCPool(
		func() []byte {
			return make([]byte, 1024)
		},
	)

	i := pool.Get()
	assert.Equal(t, 1024, len(i.Value))

	i.Retain()
	i.Release()
	i.Release()
}

func BenchmarkRCPool(b *testing.B) {
	pool := NewRCPool(
		func() []byte {
			return make([]byte, 128)
		},
	)
	for i := 0; i < b.N; i++ {
		s := pool.Get()
		s.Release()
	}
}
