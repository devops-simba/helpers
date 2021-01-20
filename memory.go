package helpers

import (
	"reflect"
	"sync"
)

func assert(condition bool, message string) {
	// if !condition {
	// 	panic(message)
	// }
}

type MemoryItem interface {
	GetNext() MemoryItem
	SetNext(value MemoryItem)
	Reset()
}
type MemoryItemCollection interface {
	GetSize() int
	GetItem(index int) MemoryItem
}

type MemoryItemArray []MemoryItem

func (this MemoryItemArray) GetSize() int                 { return len(this) }
func (this MemoryItemArray) GetItem(index int) MemoryItem { return this[index] }

type bufferMemoryItemList []buffer_t

func (this bufferMemoryItemList) GetSize() int                 { return len(this) }
func (this bufferMemoryItemList) GetItem(index int) MemoryItem { return &this[index] }

type bucketMemoryItemList []bucket_t

func (this bucketMemoryItemList) GetSize() int                 { return len(this) }
func (this bucketMemoryItemList) GetItem(index int) MemoryItem { return &this[index] }

type reflectionBasedMemoryItemCollection struct {
	array reflect.Value
}

func (this *reflectionBasedMemoryItemCollection) GetSize() int { return this.array.Len() }
func (this *reflectionBasedMemoryItemCollection) GetItem(index int) MemoryItem {
	return this.array.Index(index).Interface().(MemoryItem)
}

func CreateMemoryItemCollection(list interface{}) MemoryItemCollection {
	array := reflect.ValueOf(list)
	switch array.Kind() {
	case reflect.Slice, reflect.Array:
		return &reflectionBasedMemoryItemCollection{array: array}

	default:
		panic("You may only create a MemoryItemCollection from an array or slice")
	}

}

type MemoryItemListFactory = func(count int) MemoryItemCollection

type AllocatorStats struct {
	ReservedItems  int
	AllocatedItems int
}
type Allocator interface {
	Allocate() MemoryItem
	Free(data MemoryItem)
	GetStats() AllocatorStats
}

type memoryAllocator struct {
	avail          MemoryItem
	burstSize      int
	factory        MemoryItemListFactory
	reservedItems  int
	allocatedItems int
}
type synchedMemoryAllocator struct {
	memoryAllocator
	lock sync.Mutex
}

func NewAllocator(burstSize int, factory MemoryItemListFactory) Allocator {
	if burstSize < 0 || factory == nil {
		panic("Invalid argument")
	}

	return &memoryAllocator{avail: nil, burstSize: burstSize, factory: factory}
}
func NewSynchedAllocator(burstSize int, factory MemoryItemListFactory) Allocator {
	if burstSize < 0 || factory == nil {
		panic("Invalid argument")
	}

	return &synchedMemoryAllocator{
		memoryAllocator: memoryAllocator{avail: nil, burstSize: burstSize, factory: factory},
		lock:            sync.Mutex{},
	}
}

func allocate_memory_items(factory MemoryItemListFactory, count int) (MemoryItem, int) {
	items := factory(count)
	if items == nil {
		return nil, 0
	}

	size := items.GetSize()
	first := items.GetItem(0)

	prev := first
	for i := 1; i < size; i++ {
		item := items.GetItem(i)
		if item == nil {
			return nil, 0
		}
		prev.SetNext(item)
		prev = item
	}
	prev.SetNext(nil)

	return first, size
}
func (this *memoryAllocator) Allocate() MemoryItem {
	if this.avail == nil {
		assert(this.reservedItems == this.allocatedItems, "There is no available item, but available counter is not 0")

		avail, size := allocate_memory_items(this.factory, this.burstSize)
		if avail == nil {
			panic("MemoryItem factory should return an slice or array of MemoryItems")
		}

		this.avail = avail
		this.reservedItems += size
	}

	assert(this.allocatedItems < this.reservedItems, "There is available items but available counter is 0")
	result := this.avail
	this.allocatedItems += 1
	this.avail = this.avail.GetNext()
	result.Reset()
	return result
}
func (this *memoryAllocator) Free(item MemoryItem) {
	if item == nil {
		return
	}

	item.SetNext(this.avail)
	this.avail = item
	this.allocatedItems -= 1
}
func (this *memoryAllocator) GetStats() AllocatorStats {
	return AllocatorStats{
		ReservedItems:  this.reservedItems,
		AllocatedItems: this.allocatedItems,
	}
}

func (this *synchedMemoryAllocator) Allocate() MemoryItem {
	this.lock.Lock()
	defer this.lock.Unlock()

	return this.memoryAllocator.Allocate()
}
func (this *synchedMemoryAllocator) Free(item MemoryItem) {
	if item == nil {
		return
	}

	this.lock.Lock()
	defer this.lock.Unlock()
	this.memoryAllocator.Free(item)
}
func (this *synchedMemoryAllocator) GetStats() AllocatorStats {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.memoryAllocator.GetStats()
}

//region Buffer
type Buffer interface {
	GetSize() int
	GetData() []byte
}

type buffer_t struct {
	Data   []byte
	Bucket *bucket_t
	Start  int
	Size   int
	Next   *buffer_t
}

func (this *buffer_t) GetNext() MemoryItem {
	if this.Next == nil {
		return nil
	}
	return this.Next
}
func (this *buffer_t) SetNext(value MemoryItem) {
	if value == nil {
		this.Next = nil
	} else {
		this.Next = value.(*buffer_t)
		if this.Next == nil {
			panic("Invalid next for buffer")
		}
	}
}
func (this *buffer_t) Reset() {
	this.Next = nil
	this.Size = 0
	this.Start = 0
	this.Data = nil
	this.Bucket = nil
}

func (this *buffer_t) GetSize() int    { return this.Size }
func (this *buffer_t) GetData() []byte { return this.Data }

// end Return end location of this buffer in its bucket buffer
func (this *buffer_t) End() int { return this.Start + this.Size }

// cut Cut a buffer from this buffer, if requested buffer is equal to this buffer, return this
func (this *buffer_t) Cut(size int, bufferAllocator Allocator) *buffer_t {
	if size == this.GetSize() {
		return this
	}

	result := bufferAllocator.Allocate().(*buffer_t)
	result.Data = this.Data[:size]
	result.Start = this.Start
	result.Size = size
	result.Bucket = this.Bucket
	assert(result.End() <= cap(this.Bucket.Buffer), "Buffer overrun after allocation")
	this.Data = this.Data[size:]
	this.Start += size
	this.Size -= size
	assert(this.End() <= cap(this.Bucket.Buffer), "Buffer overrun after cut")
	return result
}

// mergableWith Check if this buffer can be merged with another buffer
func (this *buffer_t) MergableWith(other *buffer_t) bool {
	return this.Start == other.End() || this.End() == other.Start
}

// merge Merge content of this buffer with another buffer(either at start or end)
func (this *buffer_t) Merge(other *buffer_t) {
	assert(this.MergableWith(other), "Merging buffers that are not mergable")
	newStart := this.Start
	newSize := this.Size + other.Size
	if other.Start < this.Start {
		newStart = other.Start
	}
	assert(newStart+newSize <= cap(this.Bucket.Buffer), "Overrun after merge")
	this.Data = this.Bucket.Buffer[newStart : newStart+newSize]
	this.Start = newStart
	this.Size = newSize
}

//endregion

//region bucket_t
type bucket_t struct {
	Buffer      []byte
	FreeBuffers *buffer_t
	Next        *bucket_t
}

func (this *bucket_t) GetNext() MemoryItem {
	if this.Next == nil {
		return nil
	}
	return this.Next
}
func (this *bucket_t) SetNext(value MemoryItem) {
	if value == nil {
		this.Next = nil
	} else {
		this.Next = value.(*bucket_t)
		if this.Next == nil {
			panic("Invalid next for bucket_t")
		}
	}
}
func (this *bucket_t) Reset() {
	this.Next = nil
	this.FreeBuffers = nil
	this.Buffer = nil
}

func (this *bucket_t) Allocate(size int, bufferAllocator Allocator) *buffer_t {
	p := &this.FreeBuffers
	for *p != nil {
		bufferSize := (*p).Size
		if bufferSize == size {
			result := *p
			*p = (*p).Next
			return result
		}
		if bufferSize > size {
			return (*p).Cut(size, bufferAllocator)
		}

		p = &(*p).Next
	}

	return nil
}
func (this *bucket_t) Release(buffer *buffer_t, bufferAllocator Allocator) {
	if this.FreeBuffers == nil {
		this.FreeBuffers = buffer
		buffer.Next = nil
		return
	}

	p := &this.FreeBuffers
	for *p != nil {
		buf := *p
		if buffer.MergableWith(*p) {
			buffer.Merge(buf)
			*p = buf.Next
			bufferAllocator.Free(buf) // free merged buffer
			p = &this.FreeBuffers     // Restart merge checking
		} else {
			p = &buf.Next
		}
	}

	buffer.Next = this.FreeBuffers
	this.FreeBuffers = buffer
}

//endregion

type BufferManagerStats struct {
	ReservedBuckets       int
	ReservedBytes         int
	AvailableBuckets      int
	AllocatedBuffers      int
	AllocatedBytes        int
	TotalAllocatedBuffers int
	TotalAllocatedBytes   int
	BufferAllocatorStats  AllocatorStats
	BucketAllocatorStats  AllocatorStats
}
type BufferManager interface {
	GetBucketSize() int
	Allocate(size int) Buffer
	Free(buffer Buffer)
	GetStats() BufferManagerStats
}

var sentry_bucket = &bucket_t{}

type bufferManager struct {
	BufferAllocator Allocator
	BucketAllocator Allocator
	Buckets         *bucket_t
	BucketSize      int

	ReservedBuckets       int
	ReservedBytes         int
	AvailableBuckets      int
	AllocatedBuffers      int
	AllocatedBytes        int
	TotalAllocatedBuffers int
	TotalAllocatedBytes   int
}
type syncBufferManager struct {
	bufferManager
	Lock sync.Mutex
}

func NewBufferManager(bucketSize, bucketAllocatorBurst, bufferAllocatorBurst int) BufferManager {
	result := &bufferManager{}
	result.initialize(bucketSize, bucketAllocatorBurst, bufferAllocatorBurst)
	return result
}
func NewSynchedBufferManager(bucketSize, bucketAllocatorBurst, bufferAllocatorBurst int) BufferManager {
	result := &syncBufferManager{Lock: sync.Mutex{}}
	result.bufferManager.initialize(bucketSize, bucketAllocatorBurst, bufferAllocatorBurst)
	return result
}

func (this *bufferManager) initialize(bucketSize, bucketAllocatorBurst, bufferAllocatorBurst int) {
	this.BucketSize = bucketSize
	this.BucketAllocator = NewAllocator(bucketAllocatorBurst, func(count int) MemoryItemCollection {
		return make(bucketMemoryItemList, count)
	})
	this.BufferAllocator = NewAllocator(bufferAllocatorBurst, func(count int) MemoryItemCollection {
		return make(bufferMemoryItemList, count)
	})
}
func (this *bufferManager) createBucket() *bucket_t {
	newBucket := this.BucketAllocator.Allocate().(*bucket_t)
	newBucket.Buffer = make([]byte, this.BucketSize)
	newBucket.FreeBuffers = this.BufferAllocator.Allocate().(*buffer_t)
	newBucket.FreeBuffers.Bucket = newBucket
	newBucket.FreeBuffers.Data = newBucket.Buffer
	newBucket.FreeBuffers.Size = this.BucketSize
	newBucket.FreeBuffers.Start = 0
	this.ReservedBuckets += 1
	this.ReservedBytes += this.BucketSize
	return newBucket
}
func (this *bufferManager) try_remove_bucket(pbucket **bucket_t) {
	bucket := *pbucket
	if bucket.FreeBuffers == nil {
		this.AvailableBuckets -= 1
		*pbucket = bucket.Next
		bucket.Next = sentry_bucket
	}
}
func (this *bufferManager) try_insert_bucket(bucket *bucket_t) {
	if bucket.FreeBuffers != nil {
		this.AvailableBuckets += 1
		bucket.Next = this.Buckets
		this.Buckets = bucket
	} else {
		bucket.Next = sentry_bucket
	}
}
func (this *bufferManager) do_allocate(size int) *buffer_t {
	var buffer *buffer_t
	pbucket := &this.Buckets
	for *pbucket != nil {
		bucket := *pbucket
		buffer = bucket.Allocate(size, this.BufferAllocator)
		if buffer != nil {
			this.try_remove_bucket(pbucket)
			return buffer
		}

		pbucket = &bucket.Next
	}

	// there was no buffer that have enough space to allocate the buffer
	newBucket := this.createBucket()
	buffer = newBucket.Allocate(size, this.BufferAllocator)
	this.try_insert_bucket(newBucket)
	return buffer
}

func (this *bufferManager) GetBucketSize() int { return this.BucketSize }
func (this *bufferManager) Allocate(size int) Buffer {
	if size > this.BucketSize {
		return nil
	}

	buffer := this.do_allocate(size)
	this.AllocatedBuffers += 1
	this.TotalAllocatedBuffers += 1
	this.AllocatedBytes += size
	this.TotalAllocatedBytes += size
	buffer.Next = nil
	return buffer
}
func (this *bufferManager) Free(buffer Buffer) {
	if buffer == nil {
		return
	}

	buf, ok := buffer.(*buffer_t)
	if !ok {
		panic("Invalid buffer")
	}

	this.AllocatedBuffers -= 1
	this.AllocatedBytes -= buf.Size
	buf.Bucket.Release(buf, this.BufferAllocator)
	if buf.Bucket.Next == sentry_bucket {
		this.AvailableBuckets += 1
		buf.Bucket.Next = this.Buckets
		this.Buckets = buf.Bucket
	}
}
func (this *bufferManager) GetStats() BufferManagerStats {
	return BufferManagerStats{
		ReservedBuckets:       this.ReservedBuckets,
		ReservedBytes:         this.ReservedBytes,
		AvailableBuckets:      this.AvailableBuckets,
		AllocatedBuffers:      this.AllocatedBuffers,
		AllocatedBytes:        this.AllocatedBytes,
		TotalAllocatedBuffers: this.TotalAllocatedBuffers,
		TotalAllocatedBytes:   this.TotalAllocatedBytes,
		BufferAllocatorStats:  this.BufferAllocator.GetStats(),
		BucketAllocatorStats:  this.BucketAllocator.GetStats(),
	}
}

func (this *syncBufferManager) GetBucketSize() int { return this.bufferManager.BucketSize }
func (this *syncBufferManager) Allocate(size int) Buffer {
	this.Lock.Lock()
	defer this.Lock.Unlock()

	return this.bufferManager.Allocate(size)
}
func (this *syncBufferManager) Free(buffer Buffer) {
	this.Lock.Lock()
	defer this.Lock.Unlock()

	this.bufferManager.Free(buffer)
}
func (this *syncBufferManager) GetStats() BufferManagerStats {
	this.Lock.Lock()
	defer this.Lock.Unlock()

	return this.bufferManager.GetStats()
}
