// Copyright 2021 Matrix Origin
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

package deletion

import (
	"bytes"
	"sync/atomic"

	"github.com/matrixorigin/matrixone/pkg/catalog"
	"github.com/matrixorigin/matrixone/pkg/container/batch"
	"github.com/matrixorigin/matrixone/pkg/container/nulls"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/sql/colexec"
	"github.com/matrixorigin/matrixone/pkg/vm"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/tae/options"
	"github.com/matrixorigin/matrixone/pkg/vm/process"
)

//row id be divided into four types:
// 1. RawBatchOffset : belong to txn's workspace
// 2. CNBlockOffset  : belong to txn's workspace

// 3. RawRowIdBatch  : belong to txn's snapshot data.
// 4. FlushDeltaLoc   : belong to txn's snapshot data, which on S3 and pointed by delta location.
const (
	RawRowIdBatch = iota
	// remember that, for one block,
	// when it sends the info to mergedeletes,
	// either it's Compaction or not.
	Compaction
	CNBlockOffset
	RawBatchOffset
	FlushDeltaLoc
)

func (arg *Argument) String(buf *bytes.Buffer) {
	buf.WriteString("delete rows")
}

func (arg *Argument) Prepare(_ *process.Process) error {
	ap := arg
	if ap.RemoteDelete {
		ap.ctr = new(container)
		ap.ctr.state = vm.Build
		ap.ctr.blockId_type = make(map[string]int8)
		ap.ctr.blockId_bitmap = make(map[string]*nulls.Nulls)
		ap.ctr.pool = &BatchPool{pools: make([]*batch.Batch, 0, options.DefaultBlocksPerSegment)}
		ap.ctr.partitionId_blockId_rowIdBatch = make(map[int]map[string]*batch.Batch)
		ap.ctr.partitionId_blockId_deltaLoc = make(map[int]map[string]*batch.Batch)
	}
	return nil
}

// the bool return value means whether it completed its work or not
func (arg *Argument) Call(proc *process.Process) (vm.CallResult, error) {
	if arg.RemoteDelete {
		return arg.remote_delete(proc)
	}
	return arg.normal_delete(proc)
}

func (arg *Argument) remote_delete(proc *process.Process) (vm.CallResult, error) {
	if arg.ctr.state == vm.Build {
		for {
			result, err := arg.children[0].Call(proc)
			if err != nil {
				return result, err
			}
			if result.Batch == nil {
				arg.ctr.state = vm.Eval
				break
			}
			if result.Batch.IsEmpty() {
				continue
			}

			arg.SplitBatch(proc, result.Batch)
		}
	}

	result := vm.NewCallResult()
	if arg.ctr.state == vm.Eval {
		// ToDo: CNBlock Compaction
		// blkId,delta_metaLoc,type
		if arg.resBat != nil {
			proc.PutBatch(arg.resBat)
			arg.resBat = nil
		}
		arg.resBat = batch.NewWithSize(5)
		arg.resBat.Attrs = []string{
			catalog.BlockMeta_Delete_ID,
			catalog.BlockMeta_DeltaLoc,
			catalog.BlockMeta_Type,
			catalog.BlockMeta_Partition,
			catalog.BlockMeta_Deletes_Length,
		}
		arg.resBat.SetVector(0, proc.GetVector(types.T_text.ToType()))
		arg.resBat.SetVector(1, proc.GetVector(types.T_text.ToType()))
		arg.resBat.SetVector(2, proc.GetVector(types.T_int8.ToType()))
		arg.resBat.SetVector(3, proc.GetVector(types.T_int32.ToType()))

		for pidx, blockId_rowIdBatch := range arg.ctr.partitionId_blockId_rowIdBatch {
			for blkid, bat := range blockId_rowIdBatch {
				vector.AppendBytes(arg.resBat.GetVector(0), []byte(blkid), false, proc.GetMPool())
				bat.SetRowCount(bat.GetVector(0).Length())
				bytes, err := bat.MarshalBinary()
				if err != nil {
					result.Status = vm.ExecStop
					return result, err
				}
				vector.AppendBytes(arg.resBat.GetVector(1), bytes, false, proc.GetMPool())
				vector.AppendFixed(arg.resBat.GetVector(2), arg.ctr.blockId_type[blkid], false, proc.GetMPool())
				vector.AppendFixed(arg.resBat.GetVector(3), int32(pidx), false, proc.GetMPool())
			}
		}

		for pidx, blockId_deltaLoc := range arg.ctr.partitionId_blockId_deltaLoc {
			for blkid, bat := range blockId_deltaLoc {
				vector.AppendBytes(arg.resBat.GetVector(0), []byte(blkid), false, proc.GetMPool())
				//bat.Attrs = {catalog.BlockMeta_DeltaLoc}
				bat.SetRowCount(bat.GetVector(0).Length())
				bytes, err := bat.MarshalBinary()
				if err != nil {
					result.Status = vm.ExecStop
					return result, err
				}
				vector.AppendBytes(arg.resBat.GetVector(1), bytes, false, proc.GetMPool())
				vector.AppendFixed(arg.resBat.GetVector(2), int8(FlushDeltaLoc), false, proc.GetMPool())
				vector.AppendFixed(arg.resBat.GetVector(3), int32(pidx), false, proc.GetMPool())
			}
		}

		arg.resBat.SetRowCount(arg.resBat.Vecs[0].Length())
		arg.resBat.SetVector(4, vector.NewConstFixed(types.T_uint32.ToType(), arg.ctr.deleted_length, arg.resBat.RowCount(), proc.GetMPool()))

		result.Batch = arg.resBat
		arg.ctr.state = vm.End
		return result, nil
	}

	if arg.ctr.state == vm.End {
		return result, nil
	}

	panic("bug")

}

func (arg *Argument) normal_delete(proc *process.Process) (vm.CallResult, error) {
	result, err := arg.children[0].Call(proc)
	if err != nil {
		return result, err
	}
	if result.Batch == nil || result.Batch.IsEmpty() {
		return result, nil
	}
	bat := result.Batch

	var affectedRows uint64
	delCtx := arg.DeleteCtx

	if len(delCtx.PartitionTableIDs) > 0 {
		delBatches, err := colexec.GroupByPartitionForDelete(proc, bat, delCtx.RowIdIdx, delCtx.PartitionIndexInBatch,
			len(delCtx.PartitionTableIDs), delCtx.PrimaryKeyIdx)
		if err != nil {
			return result, err
		}

		for i, delBatch := range delBatches {
			tempRows := uint64(delBatch.RowCount())
			if tempRows > 0 {
				affectedRows += tempRows
				err = delCtx.PartitionSources[i].Delete(proc.Ctx, delBatch, catalog.Row_ID)
				if err != nil {
					delBatch.Clean(proc.Mp())
					return result, err
				}
				delBatch.Clean(proc.Mp())
			}
		}
	} else {
		delBatch, err := colexec.FilterRowIdForDel(proc, bat, delCtx.RowIdIdx,
			delCtx.PrimaryKeyIdx)
		if err != nil {
			return result, err
		}
		affectedRows = uint64(delBatch.RowCount())
		if affectedRows > 0 {
			err = delCtx.Source.Delete(proc.Ctx, delBatch, catalog.Row_ID)
			if err != nil {
				delBatch.Clean(proc.GetMPool())
				return result, err
			}
		}
		delBatch.Clean(proc.GetMPool())
	}
	// result.Batch = batch.EmptyBatch

	if delCtx.AddAffectedRows {
		atomic.AddUint64(&arg.affectedRows, affectedRows)
	}
	return result, nil
}
