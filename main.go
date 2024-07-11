package main

import (
	"debug/dwarf"
	"debug/elf"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

func main() {
	filename := os.Args[1]
	info, err := RuntimeInfo(filename)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	pretty, err := json.MarshalIndent(info, "", "    ")
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(2)
	}

	fmt.Println(string(pretty))
}

type Info struct {
	MOffset             uint32
	VdsoSp              uint32
	VdsoPc              uint32
	Curg                uint32
	Labels              uint32
	HmapCount           uint32
	HmapLog2BucketCount uint32
	HmapBuckets         uint32
}

func RuntimeInfo(path string) (*Info, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, err
	}

	d, err := f.DWARF()
	if err != nil {
		return nil, err
	}

	r := d.Reader()
	g, err := ReadEntry(r, "runtime.g", dwarf.TagStructType)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, errors.New("type runtime.g not found")
	}
	mPType, mOffset, err := ReadChildTypeAndOffset(r, "m")
	if err != nil {
		return nil, err
	}
	if mPType.Tag != dwarf.TagPointerType {
		return nil, errors.New("type of m in runtime.g is not a pointer")
	}

	mType, err := ReadType(r, mPType)
	if err != nil {
		return nil, err
	}

	_, spOffset, err := ReadChildTypeAndOffset(r, "vdsoSP")
	if err != nil {
		return nil, err
	}
	r.Seek(mType.Offset)
	_, err = r.Next()
	if err != nil {
		return nil, err
	}
	_, pcOffset, err := ReadChildTypeAndOffset(r, "vdsoPC")
	if err != nil {
		return nil, err
	}

	r.Seek(mType.Offset)
	_, err = r.Next()
	if err != nil {
		return nil, err
	}
	curgPType, curgOffset, err := ReadChildTypeAndOffset(r, "curg")
	if err != nil {
		return nil, err
	}
	if curgPType.Tag != dwarf.TagPointerType {
		return nil, errors.New("type of curg in m is not a pointer")
	}
	_, err = ReadType(r, curgPType)
	if err != nil {
		return nil, err
	}

	_, labelsOffset, err := ReadChildTypeAndOffset(r, "labels")
	if err != nil {
		return nil, err
	}

	hmap, err := ReadEntry(r, "runtime.hmap", dwarf.TagStructType)
	if err != nil {
		return nil, err
	}

	_, countOffset, err := ReadChildTypeAndOffset(r, "count")
	if err != nil {
		return nil, err
	}
	r.Seek(hmap.Offset)
	_, err = r.Next()
	if err != nil {
		return nil, err
	}
	_, bOffset, err := ReadChildTypeAndOffset(r, "B")
	if err != nil {
		return nil, err
	}
	r.Seek(hmap.Offset)
	_, err = r.Next()
	if err != nil {
		return nil, err
	}
	_, bucketsOffset, err := ReadChildTypeAndOffset(r, "buckets")
	if err != nil {
		return nil, err
	}

	return &Info{
		MOffset:             uint32(mOffset),
		VdsoSp:              uint32(spOffset),
		VdsoPc:              uint32(pcOffset),
		Curg:                uint32(curgOffset),
		Labels:              uint32(labelsOffset),
		HmapCount:           uint32(countOffset),
		HmapLog2BucketCount: uint32(bOffset),
		HmapBuckets:         uint32(bucketsOffset),
	}, nil
}
