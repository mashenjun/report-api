package main

// 0x0001 -> 0xFFFF
var EdgeMatrix = map[int64][]int64{
	0x0000: {0x0100, 0x0101, 0x0102},
	0x0001: {0x0102, 0x0103, 0x0104},
	0x0100: {0x0200, 0x0205},
	0x0101: {0x0201, 0x0206},
	0x0102: {0x0202, 0x0207},
	0x0103: {0x0203, 0x0208},
	0x0104: {0x0204, 0x0209},
	0x0200: {},
	0x0201: {},
	0x0202: {},
	0x0203: {},
	0x0204: {},
	0x0205: {},
	0x0206: {},
	0x0207: {},
	0x0208: {},
	0x0209: {},
}

var EdgeMatrixV2 = map[int64][]int64{
	// Write to slow -> check wirte stall
	8637: {9875},
	// check wirte stall -> check RocksDB compaction flow
	9875: {8945, 8946, 11281},
	// Check RocksDB compaction flow -> Check RocksDB write latentcy
	8945:  {10486, 11258},
	8946:  {10486, 11258},
	11281: {10486, 11258},
	// Check RocksDB write latentcy -> Check RocksDB WAL latentcy
	// Check RocksDB write latentcy -> Check the Disk Write latency skip
	// Check the Disk Write latency -> Check write batch size
	// Check RocksDB write latentcy -> Check the RocksDB CPU usage
	// Check RocksDB write latentcy -> Check the Frontend flow
	10486: {9102, 11285, 11261, 11262, 11270, 9099, 9101},
	11258: {9102, 11285, 11261, 11262, 11270, 9099, 9101},
	//  Check RocksDB WAL latentcy -> Check out Async Write
	9102:  {11284},
	11285: {11284},
	// Check out Async Write -> Check out RaftStore Threads
	// Check out Async Write -> Check out latch
	11284: {9407, 11008},
	// Check out latch -> Check out Scheduler Threads
	11008: {9255, 8947},
	// Check out Scheduler Threads -> Check Perf Context Mutex
	// Check Perf Context Mutex -> Check Perf Context Thread wait
	9255: {11263, 9571},
	8947: {11263, 9571},
	// Check Perf Context Mutex / Check Perf Context Thread wait -> Check PD Scheduling
	11263: {11276},
	9571:  {11276},
	// Check write batch size -> Check Perf Context Mutex
	11261: {11263, 9571},
	11262: {11263, 9571},

	// Read too slow -> Get too slow
	// Read too slow -> Coprocessor too slow
	9254: {10638, 11260},
	// Coprocessor too slow -> Check coprocessor threads
	11260: {9563, 10790},
	// Check coprocessor threads -> Check scanned data count
	9563:  {9561},
	10790: {9561},
	// Get too slow -> Check scanned data count
	10638: {9561},
	// Check scanned data count -> Check RPC count
	// Check scanned data count -> Check scanned Rocksed tombstone count
	9561: {10942, 10182},
	// Check scanned Rocksed tombstone count -> Check KVDB Seek and Get latency
	10182: {9567, 10030},
	// Check KVDB Seek and Get latency -> Check in-lease-read rate
	// Check KVDB Seek and Get latency -> Check memtable hit count and block-cache hit rate
	9567:  {11259, 11287, 9570},
	10030: {11259, 11287, 9570},
	// Check in-lease-read rate -> Check async-snap
	11259: {11286},
	// Check async-snap -> Check PD leader scheduling
	11286: {11276},
	// Check memtable hit count and block-cache hit rate -> Check SST read count
	11287: {9566},
	9570:  {9566},
	// Check SST read count -> Check SST read latency
	9566: {9569},
	// Check SST read latency -> Check Disk read latency

}
