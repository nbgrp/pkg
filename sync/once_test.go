package sync_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/nbgrp/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestSuccessfulOnce_Do(t *testing.T) {
	goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// Arrange
	const runLimit = 40                   // Всего запусков
	const successAt = int32(runLimit / 2) // На каком запуске нужно перестать имитировать ошибку
	once := SuccessfulOnce{}
	// Счётчик запусков:
	runCounter := int32(0)
	// Счётчик запусков, при которых функция once.Do(func) была вызвана:
	attemptCounter := int32(0)
	// Канал для ошибок, которые будут сымитированы при первых попытках.
	// Должно быть достаточно и successAt, но только для случая, когда всё работает правильно:
	errs := make(chan error, runLimit)
	wg := sync.WaitGroup{}

	// Act
	wg.Add(runLimit)
	for range runLimit {
		go func() {
			atomic.AddInt32(&runCounter, 1)
			defer wg.Done()

			err := once.Do(func() error {
				cnt := atomic.AddInt32(&attemptCounter, 1)

				if cnt == successAt {
					return nil
				}

				return attemptError{
					Cnt: cnt,
				}
			})

			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	// Assert
	assert.Equal(t, runLimit, int(runCounter))
	assert.Equal(t, successAt, attemptCounter)
	assert.Len(t, errs, int(attemptCounter-1))
	for err := range errs {
		attempt := attemptError{}
		require.True(t, errors.As(err, &attempt))
		assert.LessOrEqual(t, attempt.Cnt, successAt-1)
	}
}

type attemptError struct {
	Cnt int32
}

func (f attemptError) Error() string {
	return fmt.Sprintf("attempt %d failed", f.Cnt)
}
