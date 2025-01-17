// Copyright 2023 Matrix Origin
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

package logtailreplay

import (
	"github.com/matrixorigin/matrixone/pkg/catalog"
	"github.com/matrixorigin/matrixone/pkg/common/moerr"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/objectio"
	"github.com/tidwall/btree"
)

type ObjectsIter interface {
	Next() bool
	Close() error
	Entry() ObjectEntry
}

type objectsIter struct {
	ts          types.TS
	iter        btree.IterG[ObjectIndexByCreateTSEntry]
	firstCalled bool
}

// not accurate!  only used by stats
func (p *PartitionState) ApproxObjectsNum() int {
	return p.dataObjects.Len()
}

func (p *PartitionState) NewObjectsIter(ts types.TS) (*objectsIter, error) {
	if ts.Less(p.minTS) {
		return nil, moerr.NewTxnStaleNoCtx()
	}
	iter := p.dataObjectsByCreateTS.Copy().Iter()
	ret := &objectsIter{
		ts:   ts,
		iter: iter,
	}
	return ret, nil
}

var _ ObjectsIter = new(objectsIter)

func (b *objectsIter) Next() bool {
	for {

		if !b.firstCalled {
			if !b.iter.Seek(ObjectIndexByCreateTSEntry{
				CreateTime: b.ts.Next(),
			}) {
				if !b.iter.Last() {
					return false
				}
			}
			b.firstCalled = true
		} else {
			if !b.iter.Prev() {
				return false
			}
		}

		entry := b.iter.Item()

		if !entry.Visible(b.ts) {
			// not visible
			continue
		}

		return true
	}
}

func (b *objectsIter) Entry() ObjectEntry {
	return ObjectEntry{
		ShortObjName: b.iter.Item().ShortObjName,
		ObjectInfo:   b.iter.Item().ObjectInfo,
	}
}

func (b *objectsIter) Close() error {
	b.iter.Release()
	return nil
}

type BlocksIter interface {
	Next() bool
	Close() error
	Entry() types.Blockid
}

type dirtyBlocksIter struct {
	iter        btree.IterG[types.Blockid]
	firstCalled bool
}

func (p *PartitionState) NewDirtyBlocksIter() *dirtyBlocksIter {
	iter := p.dirtyBlocks.Copy().Iter()
	ret := &dirtyBlocksIter{
		iter: iter,
	}
	return ret
}

var _ BlocksIter = new(dirtyBlocksIter)

func (b *dirtyBlocksIter) Next() bool {
	if !b.firstCalled {
		if !b.iter.First() {
			return false
		}
		b.firstCalled = true
		return true
	}
	return b.iter.Next()
}

func (b *dirtyBlocksIter) Entry() types.Blockid {
	return b.iter.Item()
}

func (b *dirtyBlocksIter) Close() error {
	b.iter.Release()
	return nil
}

// GetChangedObjsBetween get changed objects between [begin, end]
func (p *PartitionState) GetChangedObjsBetween(
	begin types.TS,
	end types.TS,
) (
	deleted []objectio.ObjectNameShort,
	inserted []objectio.ObjectNameShort,
) {

	iter := p.objectIndexByTS.Copy().Iter()
	defer iter.Release()

	for ok := iter.Seek(ObjectIndexByTSEntry{
		Time: begin,
	}); ok; ok = iter.Next() {
		entry := iter.Item()

		if entry.Time.Greater(end) {
			break
		}

		if entry.IsDelete {
			deleted = append(deleted, entry.ShortObjName)
		} else {
			if !entry.IsAppendable {
				inserted = append(inserted, entry.ShortObjName)
			}
		}

	}

	return
}

func (p *PartitionState) GetBockDeltaLoc(bid types.Blockid) (catalog.ObjectLocation, types.TS, bool) {
	iter := p.blockDeltas.Copy().Iter()
	defer iter.Release()

	if ok := iter.Seek(BlockDeltaEntry{
		BlockID: bid,
	}); ok {
		e := iter.Item()
		return e.DeltaLoc, e.CommitTs, true
	}
	return catalog.ObjectLocation{}, types.TS{}, false
}

func (p *PartitionState) BlockPersisted(blockID types.Blockid) bool {
	iter := p.dataObjects.Copy().Iter()
	defer iter.Release()

	if ok := iter.Seek(ObjectEntry{
		ShortObjName: *objectio.ShortName(&blockID),
	}); ok {
		return true
	}
	return false
}

func (p *PartitionState) GetObject(name objectio.ObjectNameShort) (ObjectInfo, bool) {
	iter := p.dataObjects.Copy().Iter()
	defer iter.Release()

	if ok := iter.Seek(ObjectEntry{
		ShortObjName: name,
	}); ok {
		return iter.Item().ObjectInfo, true
	}
	return ObjectInfo{}, false
}
