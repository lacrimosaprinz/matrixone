// Copyright 2021 - 2022 Matrix Origin
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

package service

import (
	"strconv"
	"strings"
	"testing"

	"github.com/RoaringBitmap/roaring"
	"github.com/matrixorigin/matrixone/pkg/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceAddresses contains addresses of all services.
type serviceAddresses struct {
	t *testing.T

	// Construct service addresses according to service number
	logServiceNum int
	tnServiceNum  int
	cnServiceNum  int

	logAddresses []logServiceAddress
	tnAddresses  []tnServiceAddress
	cnAddresses  []cnServiceAddress
}

// newServiceAddresses constructs addresses for all services.
func newServiceAddresses(t *testing.T, logServiceNum, tnServiceNum, cnServiceNum int, hostAddr string) serviceAddresses {
	address := serviceAddresses{
		t:             t,
		logServiceNum: logServiceNum,
		tnServiceNum:  tnServiceNum,
		cnServiceNum:  cnServiceNum,
	}

	// build log service addresses
	logBatch := address.logServiceNum
	logAddrs := make([]logServiceAddress, logBatch)
	for i := 0; i < logBatch; i++ {
		logAddr, err := newLogServiceAddress(hostAddr)
		require.NoError(t, err)
		logAddrs[i] = logAddr
	}
	address.logAddresses = logAddrs

	// build tn service addresses
	tnBatch := address.tnServiceNum
	tnAddrs := make([]tnServiceAddress, tnBatch)
	for i := 0; i < tnBatch; i++ {
		tnAddr, err := newTNServiceAddress(hostAddr)
		require.NoError(t, err)
		tnAddrs[i] = tnAddr
	}
	address.tnAddresses = tnAddrs

	cnBatch := address.cnServiceNum
	cnAddrs := make([]cnServiceAddress, cnBatch)
	for i := 0; i < cnBatch; i++ {
		cnAddr, err := newCNServiceAddress(hostAddr)
		require.NoError(t, err)
		cnAddrs[i] = cnAddr
	}
	address.cnAddresses = cnAddrs

	return address
}

// assertTNService asserts constructed address for tn service.
func (a serviceAddresses) assertTNService() {
	assert.Equal(a.t, a.tnServiceNum, len(a.tnAddresses))
}

// assertLogService asserts constructed address for log service.
func (a serviceAddresses) assertLogService() {
	assert.Equal(a.t, a.logServiceNum, len(a.logAddresses))
}

// assertCnService asserts constructed address for cn service.
func (a serviceAddresses) assertCnService() {
	assert.Equal(a.t, a.cnServiceNum, len(a.cnAddresses))
}

// getTnListenAddress gets tn listen address by its index.
func (a serviceAddresses) getTnListenAddress(index int) string {
	a.assertTNService()

	if index >= len(a.tnAddresses) || index < 0 {
		return ""
	}
	return a.tnAddresses[index].listenAddr
}

// getTnLogtailAddress gets logtail server address by its index.
func (a serviceAddresses) getTnLogtailAddress(index int) string {
	a.assertTNService()

	if index >= len(a.tnAddresses) || index < 0 {
		return ""
	}
	return a.tnAddresses[index].logtailAddr
}

// getLogListenAddress gets log service address by its index.
func (a serviceAddresses) getLogListenAddress(index int) string {
	a.assertLogService()

	if index >= len(a.logAddresses) || index < 0 {
		return ""
	}
	return a.logAddresses[index].listenAddr
}

func (a serviceAddresses) getCNListenAddress(index int) string {
	a.assertCnService()

	if index >= len(a.cnAddresses) || index < 0 {
		return ""
	}
	return a.cnAddresses[index].listenAddr
}

// getLogRaftAddress gets log raft address by its index.
func (a serviceAddresses) getLogRaftAddress(index int) string {
	a.assertLogService()

	if index >= len(a.logAddresses) || index < 0 {
		return ""
	}
	return a.logAddresses[index].raftAddr
}

// getLogGossipAddress gets log gossip address by its index.
func (a serviceAddresses) getLogGossipAddress(index int) string {
	a.assertLogService()

	if index >= len(a.logAddresses) || index < 0 {
		return ""
	}
	return a.logAddresses[index].gossipAddr
}

// getLogGossipSeedAddresses gets all gossip seed addresses.
//
// Select gossip addresses of the first 3 log services.
// If the number of log services was less than 3,
// then select all of them.
func (a serviceAddresses) getLogGossipSeedAddresses() []string {
	a.assertLogService()

	n := gossipSeedNum(len(a.logAddresses))
	seedAddrs := make([]string, n)
	for i := 0; i < n; i++ {
		seedAddrs[i] = a.logAddresses[i].gossipAddr
	}
	return seedAddrs
}

// listHAKeeperListenAddresses gets addresses of all hakeeper servers.
//
// Select the first 3 log services to start hakeeper replica.
// If the number of log services was less than 3,
// then select the first of them.
func (a serviceAddresses) listHAKeeperListenAddresses() []string {
	a.assertLogService()

	n := haKeeperNum(len(a.logAddresses))
	listenAddrs := make([]string, n)
	for i := 0; i < n; i++ {
		listenAddrs[i] = a.logAddresses[i].listenAddr
	}
	return listenAddrs
}

// buildPartitionAddressSets returns service addresses by every partition.
func (a serviceAddresses) buildPartitionAddressSets(partitions ...NetworkPartition) []addressSet {
	sets := make([]addressSet, 0, len(partitions))
	for _, part := range partitions {
		sets = append(sets, a.listPartitionAddresses(part))
	}
	return sets
}

// listPartitionAddresses returns all service addresses within the same partition.
func (a serviceAddresses) listPartitionAddresses(partition NetworkPartition) addressSet {
	addrSet := newAddressSet()
	for _, tnIndex := range partition.ListTNServiceIndex() {
		addrs := a.listTnServiceAddresses(int(tnIndex))
		addrSet.addAddresses(addrs...)
	}
	for _, logIndex := range partition.ListLogServiceIndex() {
		addrs := a.listLogServiceAddresses(int(logIndex))
		addrSet.addAddresses(addrs...)
	}
	for _, cnIndex := range partition.ListCNServiceIndex() {
		addrs := a.listCnServiceAddresses(int(cnIndex))
		addrSet.addAddresses(addrs...)
	}
	return addrSet
}

// listTnServiceAddresses lists all addresses of tn service by its index.
func (a serviceAddresses) listTnServiceAddresses(index int) []string {
	a.assertTNService()

	if index >= len(a.tnAddresses) || index < 0 {
		return nil
	}
	return a.tnAddresses[index].listAddresses()
}

// listLogServiceAddresses lists all addresses of log service by its index.
func (a serviceAddresses) listLogServiceAddresses(index int) []string {
	a.assertLogService()

	if index >= len(a.logAddresses) || index < 0 {
		return nil
	}
	return a.logAddresses[index].listAddresses()
}

// listCnServiceAddresses lists all addresses of log service by its index.
func (a serviceAddresses) listCnServiceAddresses(index int) []string {
	a.assertCnService()

	if index >= len(a.cnAddresses) || index < 0 {
		return nil
	}
	return a.cnAddresses[index].listAddresses()
}

// logServiceAddress contains addresses for log service.
type logServiceAddress struct {
	listenAddr string
	raftAddr   string
	gossipAddr string
}

func newLogServiceAddress(host string) (logServiceAddress, error) {
	addrs, err := tests.GetAddressBatch(host, 3)
	if err != nil {
		return logServiceAddress{}, err
	}

	return logServiceAddress{
		listenAddr: addrs[0],
		raftAddr:   addrs[1],
		gossipAddr: addrs[2],
	}, nil
}

// listAddresses returns all addresses for single log service.
func (la logServiceAddress) listAddresses() []string {
	return []string{la.listenAddr, la.raftAddr, la.gossipAddr}
}

// tnServiceAddress contains address for tn service.
type tnServiceAddress struct {
	listenAddr  string
	logtailAddr string
}

func newTNServiceAddress(host string) (tnServiceAddress, error) {
	addrs, err := tests.GetAddressBatch(host, 2)
	if err != nil {
		return tnServiceAddress{}, err
	}
	return tnServiceAddress{
		listenAddr:  addrs[0],
		logtailAddr: addrs[1],
	}, nil
}

// listAddresses returns all addresses for single tn service.
func (da tnServiceAddress) listAddresses() []string {
	return []string{da.listenAddr, da.logtailAddr}
}

type cnServiceAddress struct {
	listenAddr string
}

func newCNServiceAddress(host string) (cnServiceAddress, error) {
	addrs, err := tests.GetAddressBatch(host, 1)
	if err != nil {
		return cnServiceAddress{}, err
	}
	return cnServiceAddress{listenAddr: addrs[0]}, nil
}

func (ca cnServiceAddress) listAddresses() []string {
	return []string{ca.listenAddr}
}

// addressSet records addresses for services within the same partition.
type addressSet map[string]struct{}

func newAddressSet() addressSet {
	return make(map[string]struct{})
}

// addAddresses registers a list of addresses.
func (s addressSet) addAddresses(addrs ...string) {
	for _, addr := range addrs {
		s[addr] = struct{}{}
	}
}

// contains checks address exist or not.
func (s addressSet) contains(addr string) bool {
	_, ok := s[addr]
	return ok
}

// NetworkPartition records index of services from the same network partition.
type NetworkPartition struct {
	logIndexSet *roaring.Bitmap
	tnIndexSet  *roaring.Bitmap
	cnIndexSet  *roaring.Bitmap
}

// newNetworkPartition returns an instance of NetworkPartition.
//
// The returned instance only contains valid index according to service number.
func newNetworkPartition(
	logServiceNum int, logIndexes []uint32,
	tnServiceNum int, tnIndexes []uint32,
	cnServiceNum int, cnIndexes []uint32,
) NetworkPartition {
	logTotal := roaring.FlipInt(roaring.NewBitmap(), 0, logServiceNum)
	tnTotal := roaring.FlipInt(roaring.NewBitmap(), 0, tnServiceNum)
	cnTotal := roaring.FlipInt(roaring.NewBitmap(), 0, cnServiceNum)

	rawLogSet := roaring.BitmapOf(logIndexes...)
	rawTnSet := roaring.BitmapOf(tnIndexes...)
	rawCnSet := roaring.BitmapOf(cnIndexes...)

	return NetworkPartition{
		logIndexSet: roaring.And(logTotal, rawLogSet),
		tnIndexSet:  roaring.And(tnTotal, rawTnSet),
		cnIndexSet:  roaring.And(cnTotal, rawCnSet),
	}
}

// remainingNetworkPartition returns partition for the remaining services.
func remainingNetworkPartition(logServiceNum, tnServiceNum, cnServiceNum int,
	partitions ...NetworkPartition) NetworkPartition {
	logTotal := roaring.FlipInt(roaring.NewBitmap(), 0, logServiceNum)
	tnTotal := roaring.FlipInt(roaring.NewBitmap(), 0, tnServiceNum)
	cnTotal := roaring.FlipInt(roaring.NewBitmap(), 0, cnServiceNum)

	logUsed := roaring.NewBitmap()
	tnUsed := roaring.NewBitmap()
	cnUsed := roaring.NewBitmap()
	for _, p := range partitions {
		tnUsed.Or(p.tnIndexSet)
		logUsed.Or(p.logIndexSet)
		cnUsed.Or(p.cnIndexSet)
	}

	return NetworkPartition{
		logIndexSet: roaring.AndNot(logTotal, logUsed),
		tnIndexSet:  roaring.AndNot(tnTotal, tnUsed),
		cnIndexSet:  roaring.AndNot(cnTotal, cnUsed),
	}
}

// ListTNServiceIndex lists index of all tn services in the partition.
func (p NetworkPartition) ListTNServiceIndex() []uint32 {
	set := p.tnIndexSet

	if set.GetCardinality() == 0 {
		return nil
	}

	indexes := make([]uint32, 0, set.GetCardinality())
	iter := set.Iterator()
	for iter.HasNext() {
		indexes = append(indexes, iter.Next())
	}
	return indexes
}

// ListLogServiceIndex lists index of all log services in the partition.
func (p NetworkPartition) ListLogServiceIndex() []uint32 {
	set := p.logIndexSet

	if set.GetCardinality() == 0 {
		return nil
	}

	indexes := make([]uint32, 0, set.GetCardinality())
	iter := set.Iterator()
	for iter.HasNext() {
		indexes = append(indexes, iter.Next())
	}
	return indexes
}

func (p NetworkPartition) ListCNServiceIndex() []uint32 {
	set := p.cnIndexSet

	if set.GetCardinality() == 0 {
		return nil
	}

	indexes := make([]uint32, 0, set.GetCardinality())
	iter := set.Iterator()
	for iter.HasNext() {
		indexes = append(indexes, iter.Next())
	}
	return indexes
}

func getPort(addr string) int {
	ss := strings.Split(addr, ":")
	if len(ss) != 2 {
		panic("bad address: " + addr)
	}
	p, err := strconv.Atoi(ss[1])
	if err != nil {
		panic("bad address: " + addr)
	}
	return p
}
