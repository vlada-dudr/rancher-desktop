package funcqueue

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestFuncQueue(t *testing.T) {
	t.Run("should run all functions if context not cancelled and no errors", func(t *testing.T) {
		ctx := context.Background()
		funcQueue := NewFuncQueue(ctx)
		ranSlice := []bool{false, false, false}
		for i := range ranSlice {
			i := i
			funcQueue.Add(func() error {
				ranSlice[i] = true
				return nil
			})
		}
		if err := funcQueue.Wait(); err != nil {
			t.Fatalf("unexpected error waiting on funcQueue: %s", err)
		}
		for i := range ranSlice {
			if !ranSlice[i] {
				t.Errorf("function %d appears to not have run", i)
			}
		}
	})

	t.Run("should stop execution after current function when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		funcQueue := NewFuncQueue(ctx)

		// Used to delay until func1 has started running
		readyForCancelChan := make(chan struct{})
		// Gets closed when we want func1 to return
		func1Chan := make(chan struct{})
		func1Ran := false
		func1 := func() error {
			func1Ran = true
			t.Log("func1 ran")
			close(readyForCancelChan)
			<-func1Chan
			return nil
		}

		func2Ran := false
		func2 := func() error {
			func2Ran = true
			t.Log("func2 ran")
			return nil
		}

		funcQueue.Add(func1)
		funcQueue.Add(func2)
		<-readyForCancelChan
		cancel()
		close(func1Chan)

		if err := funcQueue.Wait(); !errors.Is(err, ErrContextDone) {
			t.Fatalf("unexpected error waiting on funcQueue: %s", err)
		}
		if !func1Ran {
			t.Errorf("func1 unexpectedly did not run")
		}
		if func2Ran {
			t.Errorf("func2 ran but should not have")
		}
	})

	t.Run("should return error from first function that errors out and not run subsequent functions", func(t *testing.T) {
		ctx := context.Background()
		funcQueue := NewFuncQueue(ctx)

		expectedError := "func1 error"
		ranSlice := make([]bool, 2)
		for i := range ranSlice {
			i := i
			funcQueue.Add(func() error {
				ranSlice[i] = true
				t.Logf("func%d ran", i+1)
				return fmt.Errorf("func%d error", i+1)
			})
		}
		if err := funcQueue.Wait(); err.Error() != expectedError {
			t.Errorf("funcQueue.Wait() returned unexpected error %q (expected %q)", err, expectedError)
		}
		if !ranSlice[0] {
			t.Errorf("func1 unexpectedly did not run")
		}
		if ranSlice[1] {
			t.Errorf("func2 ran but should not have")
		}
	})
}
