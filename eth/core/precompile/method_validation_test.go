// SPDX-License-Identifier: MIT
//
// Copyright (c) 2024 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
// OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
// WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package precompile

import (
	"context"
	"math/big"
	"reflect"

	"github.com/berachain/polaris/contracts/bindings/testing"
	"github.com/berachain/polaris/eth/accounts/abi"

	"github.com/ethereum/go-ethereum/common"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Method", func() {
	var precompileABI map[string]abi.Method
	var m *mockImpl
	BeforeEach(func() {
		precompileABI = abi.MustUnmarshalJSON(testing.MockMethodsABI).Methods
		m = &mockImpl{}
	})
	Context("findMatchingAbiMethod", func() {
		It("should validate args successfully", func() {
			exampleFuncValue, found := reflect.TypeOf(m).MethodByName("ExampleFunc")
			Expect(found).To(BeTrue())

			methodName, err := findMatchingABIMethod(exampleFuncValue, precompileABI)
			Expect(err).ToNot(HaveOccurred())
			Expect(methodName).To(Equal("exampleFunc"))

			sliceA := []uint64{}
			sliceB := []*big.Int{}

			Expect(validateArg(
				reflect.ValueOf(sliceA),
				reflect.ValueOf(sliceB)).Error()).To(Equal(
				"type mismatch: []uint64 != []*big.Int",
			))
		})
	})

	Context("validateArg", func() {
		It("should error when array and scalar mismatch", func() {
			sliceA := []uint64{0}
			sliceB := uint64(0)
			Expect(validateArg(
				reflect.ValueOf(sliceA),
				reflect.ValueOf(sliceB)).Error()).To(Equal(
				"type mismatch: []uint64 != uint64",
			))
		})
		It("should error when struct fields aren't the same", func() {
			sliceA := []mockStruct{}
			sliceB := []mockStructBad{}
			Expect(validateArg(
				reflect.ValueOf(sliceA),
				reflect.ValueOf(sliceB)).Error()).To(Equal(
				"type mismatch: *big.Int != uint64",
			))
		})
		It("should error when we point to a non-struct", func() {
			randomPointer := 69
			implMethodVarType := &randomPointer
			abiMethodVarType := &mockStruct{}
			Expect(validateArg(
				reflect.ValueOf(implMethodVarType).Elem(),
				reflect.ValueOf(abiMethodVarType)).Error()).To(Equal(
				"type mismatch: int != *precompile.mockStruct",
			))
		})
	})

	It("should panic when our ABI method does not return anything", func() {
		zeroReturn := precompileABI["zeroReturn"]
		mockMethod, _ := reflect.TypeOf(m).MethodByName("MockMethod")

		//nolint:errcheck // it's going to panic
		Expect(func() { validateOutputs(mockMethod, &zeroReturn) }).To(Panic())
	})
	It("should error when we have different structs as params", func() {
		m := mockStruct{}
		mb := mockStructBad{}

		Expect(validateArg(
			reflect.New(reflect.TypeOf(m)).Elem(),
			reflect.New(reflect.TypeOf(mb)).Elem())).To(HaveOccurred())

		Expect(validateStruct(reflect.TypeOf(m), reflect.TypeOf(mb))).To(HaveOccurred())
		mbn := mockStructBadNumFields{}

		Expect(validateStruct(reflect.TypeOf(m), reflect.TypeOf(mbn))).To(HaveOccurred())

		notAStruct := 69

		Expect(validateStruct(reflect.TypeOf(m), reflect.TypeOf(notAStruct)).Error()).To(Equal(
			"validateStruct: not a struct"))
	})

	Context("validateOutputs", func() {
		It("should error when our impl and abi outputs aren't correct", func() {
			exampleFunc := precompileABI["exampleFunc"]

			noErrorReturn, found := reflect.TypeOf(m).MethodByName("NoErrorReturn")
			Expect(found).To(BeTrue())
			Expect(validateOutputs(noErrorReturn, &exampleFunc).Error()).To(Equal(
				"last return type must be error, got bool"))

			numReturnMismatch, found := reflect.TypeOf(m).MethodByName("NumReturnMismatch")
			Expect(found).To(BeTrue())
			Expect(validateOutputs(numReturnMismatch, &exampleFunc).Error()).To(Equal(
				"number of return args mismatch: exampleFunc expects 1 return vals, " +
					"NumReturnMismatch returns 0 vals"))

			returnTypeMismatch, found := reflect.TypeOf(m).MethodByName("ReturnTypeMismatch")
			Expect(found).To(BeTrue())
			Expect(validateOutputs(returnTypeMismatch, &exampleFunc).Error()).To(Equal(
				"return type mismatch: exampleFunc expects bool, ReturnTypeMismatch has string"))
		})
	})

	Context("findMatchingABIMethod", func() {

		It("should return ErrNoImplMethodSubstringMatchesABIMethods", func() {
			mockMethod, found := reflect.TypeOf(m).MethodByName("MockMethod")
			Expect(found).To(BeTrue())
			methodName, err := findMatchingABIMethod(mockMethod, precompileABI)
			Expect(methodName).To(Equal(""))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

type mockImpl struct{}

type mockStruct struct {
	_ *big.Int
}

type mockStructBad struct {
	_ uint64
}

type mockStructBadNumFields struct {
	_ *big.Int
	_ *big.Int
}

func (m *mockImpl) MockMethod() error { return nil }

func (m *mockImpl) ExampleFunc(
	_ context.Context,
	_ *big.Int,
	_ common.Address,
	_ []mockStruct,
) (bool, error) {
	return true, nil
}

func (m *mockImpl) ExampleFuncBad(
	_ context.Context,
	_ *big.Int,
	_ common.Address,
	_ []mockStructBad,
) (bool, error) {
	return true, nil
}

func (m *mockImpl) NoErrorReturn(_ context.Context, _ *big.Int) (bool, bool) {
	return true, true
}

func (m *mockImpl) NumReturnMismatch(_ context.Context, _ *big.Int) error {
	return nil
}

func (m *mockImpl) ReturnTypeMismatch(context.Context, *big.Int) (string, error) {
	return "bera", nil
}
