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

package vm

import (
	"bytes"

	"github.com/matrixorigin/matrixone/pkg/container/batch"
	"github.com/matrixorigin/matrixone/pkg/vm/process"
)

type OpType int

const (
	Top OpType = iota
	Limit
	Order
	Group
	Window
	TimeWin
	Fill
	Output
	Offset
	Product
	Restrict
	Dispatch
	Connector
	Projection

	Join
	LoopJoin
	Left
	LoopLeft
	Single
	LoopSingle
	Semi
	RightSemi
	LoopSemi
	Anti
	RightAnti
	LoopAnti
	Mark
	LoopMark

	Merge
	MergeTop
	MergeLimit
	MergeOrder
	MergeGroup
	MergeOffset
	MergeRecursive
	MergeCTE

	Deletion
	Insert
	External
	Stream

	Minus
	Intersect
	IntersectAll

	HashBuild

	TableFunction
	TableScan
	ValueScan
	// MergeBlock is used to recieve S3 block metLoc Info, and write
	// them to S3
	MergeBlock
	// MergeDelete is used to recieve S3 Blcok Delete Info from remote Cn
	MergeDelete
	Right
	OnDuplicateKey
	PreInsert
	PreInsertUnique
	PreInsertSecondaryIndex
	// LastInstructionOp is not a true operator and must set at last.
	// It was used by unit testing to ensure that
	// all functions related to instructions can reach 100% coverage.
	LastInstructionOp

	// LockOp is used to add locks to lockservice for pessimistic transactions.
	// Operator that encounters a write conflict will block until the previous
	// transaction has released the lock
	LockOp

	Shuffle
)

// Instruction contains relational algebra
type Instruction struct {
	// Op specified the operator code of an instruction.
	Op OpType
	// Idx specified the analysis information index.
	Idx int
	// Arg contains the operand of this instruction.
	Arg Operator

	// flag for analyzeInfo record the row information
	IsFirst bool
	IsLast  bool
}

type Operator interface {
	// Free release all the memory allocated from mPool in an operator.
	// pipelineFailed marks the process status of the pipeline when the method is called.
	Free(proc *process.Process, pipelineFailed bool, err error)

	// String returns the string representation of an operator.
	String(buf *bytes.Buffer)

	//Prepare prepares an operator for execution.
	Prepare(proc *process.Process) error

	//Call calls an operator.
	Call(proc *process.Process) (CallResult, error)

	//SetInfo set operator info
	SetInfo(info *OperatorInfo)

	//AppendChild append child to operator
	AppendChild(child Operator)
}

type ExecStatus int

const (
	ExecStop ExecStatus = iota
	ExecNext
	ExecHasMore
)

type CtrState int

const (
	Build CtrState = iota
	Eval
	End
)

type CallResult struct {
	Status ExecStatus
	Batch  *batch.Batch
}

func NewCallResult() CallResult {
	return CallResult{
		Status: ExecNext,
	}
}

type OperatorInfo struct {
	Idx     int
	IsFirst bool
	IsLast  bool
}

type Instructions []Instruction

func (ins *Instruction) IsBrokenNode() bool {
	switch ins.Op {
	case Order, MergeOrder:
		return true
	case Limit, MergeLimit:
		return true
	case Offset, MergeOffset:
		return true
	case Group, MergeGroup:
		return true
	case Top, MergeTop:
		return true
	case Window:
		return true
	case TimeWin, Fill:
		return true
	case MergeRecursive:
		return true
	}
	return false
}

type ModificationArgument interface {
	AffectedRows() uint64
}
