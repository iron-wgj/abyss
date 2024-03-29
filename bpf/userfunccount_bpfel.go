// Code generated by bpf2go; DO NOT EDIT.
//go:build 386 || amd64 || amd64p32 || arm || arm64 || mips64le || mips64p32le || mipsle || ppc64le || riscv64
// +build 386 amd64 amd64p32 arm arm64 mips64le mips64p32le mipsle ppc64le riscv64

package bpf

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/cilium/ebpf"
)

// loadUserFuncCount returns the embedded CollectionSpec for userFuncCount.
func loadUserFuncCount() (*ebpf.CollectionSpec, error) {
	reader := bytes.NewReader(_UserFuncCountBytes)
	spec, err := ebpf.LoadCollectionSpecFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("can't load userFuncCount: %w", err)
	}

	return spec, err
}

// loadUserFuncCountObjects loads userFuncCount and converts it into a struct.
//
// The following types are suitable as obj argument:
//
//	*userFuncCountObjects
//	*userFuncCountPrograms
//	*userFuncCountMaps
//
// See ebpf.CollectionSpec.LoadAndAssign documentation for details.
func loadUserFuncCountObjects(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := loadUserFuncCount()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

// userFuncCountSpecs contains maps and programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type userFuncCountSpecs struct {
	userFuncCountProgramSpecs
	userFuncCountMapSpecs
}

// userFuncCountSpecs contains programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type userFuncCountProgramSpecs struct {
	UprobeFuncCall *ebpf.ProgramSpec `ebpf:"uprobe__func_call"`
}

// userFuncCountMapSpecs contains maps before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type userFuncCountMapSpecs struct {
	Events *ebpf.MapSpec `ebpf:"events"`
}

// userFuncCountObjects contains all objects after they have been loaded into the kernel.
//
// It can be passed to loadUserFuncCountObjects or ebpf.CollectionSpec.LoadAndAssign.
type userFuncCountObjects struct {
	userFuncCountPrograms
	userFuncCountMaps
}

func (o *userFuncCountObjects) Close() error {
	return _UserFuncCountClose(
		&o.userFuncCountPrograms,
		&o.userFuncCountMaps,
	)
}

// userFuncCountMaps contains all maps after they have been loaded into the kernel.
//
// It can be passed to loadUserFuncCountObjects or ebpf.CollectionSpec.LoadAndAssign.
type userFuncCountMaps struct {
	Events *ebpf.Map `ebpf:"events"`
}

func (m *userFuncCountMaps) Close() error {
	return _UserFuncCountClose(
		m.Events,
	)
}

// userFuncCountPrograms contains all programs after they have been loaded into the kernel.
//
// It can be passed to loadUserFuncCountObjects or ebpf.CollectionSpec.LoadAndAssign.
type userFuncCountPrograms struct {
	UprobeFuncCall *ebpf.Program `ebpf:"uprobe__func_call"`
}

func (p *userFuncCountPrograms) Close() error {
	return _UserFuncCountClose(
		p.UprobeFuncCall,
	)
}

func _UserFuncCountClose(closers ...io.Closer) error {
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Do not access this directly.
//
//go:embed userfunccount_bpfel.o
var _UserFuncCountBytes []byte
