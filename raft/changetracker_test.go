package raft

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Change tracker", func() {
	var tracker *ChangeTracker

	BeforeEach(func() {
		tracker = CreateTracker()
		tracker.Update(2)
	})

	AfterEach(func() {
		tracker.Close()
	})

	It("Behind", func() {
		behind := tracker.Wait(1)
		Expect(behind).Should(BeEquivalentTo(2))
	})

	It("Caught up", func() {
		behind := tracker.Wait(2)
		Expect(behind).Should(BeEquivalentTo(2))
	})

	It("Up to date", func() {
		doneChan := make(chan uint64, 1)

		go func() {
			new := tracker.Wait(3)
			doneChan <- new
		}()

		tracker.Update(3)
		gotVal := <-doneChan
		Expect(gotVal).Should(BeEquivalentTo(3))
	})

	It("Up to date with timeout", func() {
		doneChan := make(chan uint64, 1)

		go func() {
			new := tracker.TimedWait(3, 2*time.Second)
			doneChan <- new
		}()

		tracker.Update(3)
		gotVal := <-doneChan
		Expect(gotVal).Should(BeEquivalentTo(3))
	})

	It("Up to date timeout", func() {
		doneChan := make(chan uint64, 1)

		go func() {
			new := tracker.TimedWait(3, 500*time.Millisecond)
			doneChan <- new
		}()

		time.Sleep(1 * time.Second)
		tracker.Update(3)
		gotVal := <-doneChan
		Expect(gotVal).Should(BeEquivalentTo(2))
	})

	It("Update", func() {
		doneChan := make(chan uint64, 1)

		go func() {
			new := tracker.Wait(4)
			doneChan <- new
		}()

		time.Sleep(250 * time.Millisecond)
		tracker.Update(3)
		time.Sleep(250 * time.Millisecond)
		tracker.Update(4)
		gotVal := <-doneChan
		Expect(gotVal).Should(BeEquivalentTo(4))
	})

	It("Update twice", func() {
		doneChan := make(chan uint64, 1)
		doneChan2 := make(chan uint64, 1)

		go func() {
			new := tracker.Wait(4)
			doneChan <- new
		}()

		go func() {
			new2 := tracker.Wait(4)
			doneChan2 <- new2
		}()

		time.Sleep(250 * time.Millisecond)
		tracker.Update(3)
		time.Sleep(250 * time.Millisecond)
		tracker.Update(4)
		gotVal := <-doneChan
		Expect(gotVal).Should(BeEquivalentTo(4))
		gotVal = <-doneChan2
		Expect(gotVal).Should(BeEquivalentTo(4))
	})

	It("Multi Update", func() {
		prematureDoneChan := make(chan uint64, 1)
		doneChan := make(chan uint64, 1)

		go func() {
			oldNew := tracker.Wait(10)
			prematureDoneChan <- oldNew
		}()

		go func() {
			new := tracker.Wait(4)
			doneChan <- new
		}()

		time.Sleep(250 * time.Millisecond)
		tracker.Update(3)
		time.Sleep(250 * time.Millisecond)
		tracker.Update(4)

		// No loop -- we expect that the first case arrive before the second
		select {
		case gotVal := <-doneChan:
			Expect(gotVal).Should(BeEquivalentTo(4))
		case <-prematureDoneChan:
			Expect(true).Should(BeFalse())
		}
	})

	It("Close", func() {
		tracker := CreateTracker()
		tracker.Update(2)
		done := make(chan uint64, 1)

		go func() {
			new := tracker.Wait(3)
			done <- new
		}()

		time.Sleep(250 * time.Millisecond)
		tracker.Close()

		val := <-done
		Expect(val).Should(BeEquivalentTo(2))
	})

	It("Naming", func() {
		name1 := GetNamedTracker("test1")
		name2 := GetNamedTracker("test2")
		name1same := GetNamedTracker("test1")
		Expect(name1).Should(Equal(name1same))
		Expect(name1).ShouldNot(Equal(name2))
	})
})
