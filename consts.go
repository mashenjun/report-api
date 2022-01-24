package main

// 0x0001 -> 0xFFFF
var EdgeMatrix = map[int64][]int64{
	0x0000: []int64{0x0100, 0x0101, 0x0102},
	0x0001: []int64{0x0102, 0x0103, 0x0104},
	0x0100: []int64{0x0200, 0x0205},
	0x0101: []int64{0x0201, 0x0206},
	0x0102: []int64{0x0202, 0x0207},
	0x0103: []int64{0x0203, 0x0208},
	0x0104: []int64{0x0204, 0x0209},
	0x0200: []int64{},
	0x0201: []int64{},
	0x0202: []int64{},
	0x0203: []int64{},
	0x0204: []int64{},
	0x0205: []int64{},
	0x0206: []int64{},
	0x0207: []int64{},
	0x0208: []int64{},
	0x0209: []int64{},
}
