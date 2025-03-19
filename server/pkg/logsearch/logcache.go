package logsearch

import (
	"context"
	"fmt"
	"sync"

	"github.com/outrigdev/outrig/pkg/ds"
	"github.com/outrigdev/outrig/pkg/utilfn"
	"github.com/outrigdev/outrig/server/pkg/apppeer"
)

const ChunkPending = -1
const DefaultBackendChunkSize = 500

type RawLogSource interface {
	TotalSize() (int, error)
	InitSource(searcher LogSearcher, startOffset int, chunkSize int)
	GetPageSize() int
	GetSearchedSize() int

	// returns (results, window, eof, error)
	SearchNextChunk() ([]ds.LogLine, bool, error)
	Reset()
}

type LogCache struct {
	Lock         *sync.Mutex
	TotalCount   int
	FilteredSize int
	PageSize     int
	Done         bool
	ChunkSizes   []int
	Cache        [][]ds.LogLine
	LogSource    RawLogSource
}

func MakeLogCache(source RawLogSource) (*LogCache, error) {
	totalSize, err := source.TotalSize()
	if err != nil {
		return nil, err
	}
	pageSize := source.GetPageSize()
	numChunks := totalSize / pageSize
	if totalSize%pageSize != 0 {
		numChunks++
	}
	chunkSizes := make([]int, numChunks)
	for i := 0; i < numChunks; i++ {
		chunkSizes[i] = ChunkPending
	}
	return &LogCache{
		Lock:         &sync.Mutex{},
		TotalCount:   totalSize,
		FilteredSize: 0,
		PageSize:     pageSize,
		Done:         false,
		ChunkSizes:   chunkSizes,
		Cache:        make([][]ds.LogLine, numChunks),
		LogSource:    source,
	}, nil
}

func (lc *LogCache) searchChunk(chunkNum int) {
	if chunkNum > len(lc.ChunkSizes) {
		return
	}
	lines, eof, err := lc.LogSource.SearchNextChunk()
	if err != nil {
		return
	}
	lc.Lock.Lock()
	defer lc.Lock.Unlock()
	lc.FilteredSize += len(lines)
	lc.ChunkSizes[chunkNum] = len(lines)
	lc.Cache[chunkNum] = lines
	if eof {
		lc.Done = true
	}
}

func (lc *LogCache) GetRange(startIndex int, endIndex int) []ds.LogLine {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()

	// Ensure indices are within valid bounds
	startIndex = utilfn.BoundValue(startIndex, 0, lc.FilteredSize)
	endIndex = utilfn.BoundValue(endIndex, startIndex, lc.FilteredSize)

	// Create a slice with capacity for the requested range
	allLines := make([]ds.LogLine, 0, endIndex-startIndex)

	// If after bounds checking we have nothing to return, exit early
	if startIndex == endIndex {
		return allLines
	}

	// Find the chunks containing our target range
	currentIndex := 0
	for i := 0; i < len(lc.Cache); i++ {
		// Stop at pending chunks
		if lc.ChunkSizes[i] == ChunkPending {
			break
		}
		if currentIndex > endIndex {
			break
		}

		// Get filtered lines count in this chunk
		chunkSize := lc.ChunkSizes[i]
		nextIndex := currentIndex + chunkSize

		// Skip chunks completely before the range
		if nextIndex <= startIndex {
			currentIndex = nextIndex
			continue
		}

		// Calculate relative indices within this chunk
		chunkStartIdx := utilfn.BoundValue(startIndex-currentIndex, 0, chunkSize)
		chunkEndIdx := utilfn.BoundValue(endIndex-currentIndex, chunkStartIdx, chunkSize)

		// Add the needed lines from this chunk, update currentIndex
		allLines = append(allLines, lc.Cache[i][chunkStartIdx:chunkEndIdx]...)
		currentIndex = nextIndex
	}

	return allLines
}

func (lc *LogCache) GetFilteredSize() int {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()
	return lc.FilteredSize
}

func (lc *LogCache) GetTotalSize() int {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()
	return lc.TotalCount
}

func (lc *LogCache) GetSearchedSize() int {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()
	// Return the size of the circular buffer, which is the number of entries
	// that were actually searched (may be less than TotalCount if the buffer is limited)
	return lc.LogSource.GetSearchedSize()
}

func (lc *LogCache) IsDone() bool {
	lc.Lock.Lock()
	defer lc.Lock.Unlock()
	return lc.Done
}

func (lc *LogCache) RunSearch(updateFn func()) context.CancelFunc {
	ctx, cancelFn := context.WithCancel(context.Background())
	go func() {
		defer cancelFn()
		chunkNum := 0
		for {
			if lc.IsDone() || ctx.Err() != nil || chunkNum >= len(lc.ChunkSizes) {
				break
			}
			lc.searchChunk(chunkNum)
			if updateFn != nil {
				updateFn()
			}
			chunkNum++
		}
	}()
	return cancelFn
}

type AppPeerLogSource struct {
	AppPeer   *apppeer.AppRunPeer
	IsActive  bool
	Searcher  LogSearcher
	Offset    int
	ChunkSize int
}

func MakeAppPeerLogSource(appPeer *apppeer.AppRunPeer) RawLogSource {
	return &AppPeerLogSource{
		AppPeer: appPeer,
	}
}

func (ls *AppPeerLogSource) InitSource(searcher LogSearcher, startOffset int, chunkSize int) {
	ls.Searcher = searcher
	ls.Offset = startOffset
	ls.ChunkSize = chunkSize
	ls.IsActive = true
}

func (ls *AppPeerLogSource) Reset() {
	ls.Searcher = nil
	ls.Offset = 0
	ls.IsActive = false
}

func (ls *AppPeerLogSource) TotalSize() (int, error) {
	totalCount, _ := ls.AppPeer.Logs.GetTotalCountAndHeadOffset()
	return totalCount, nil
}

func (ls *AppPeerLogSource) GetPageSize() int {
	return ls.ChunkSize
}

func (ls *AppPeerLogSource) GetSearchedSize() int {
	// Return the size of the circular buffer, which is the number of entries
	// that were actually searched (may be less than TotalCount if the buffer is limited)
	size := ls.AppPeer.Logs.Size()
	return size
}

func (ls *AppPeerLogSource) SearchNextChunk() ([]ds.LogLine, bool, error) {
	if !ls.IsActive {
		return nil, false, fmt.Errorf("log source is not active")
	}
	
	// Ensure ChunkSize is positive
	if ls.ChunkSize <= 0 {
		return nil, true, fmt.Errorf("invalid chunk size: %d", ls.ChunkSize)
	}
	
	// Check if logs are available
	totalCount, _ := ls.AppPeer.Logs.GetTotalCountAndHeadOffset()
	if totalCount == 0 || ls.Offset >= totalCount {
		return nil, true, nil // EOF, no logs available
	}
	
	lines, trueOffset, eof := ls.AppPeer.Logs.GetRange(ls.Offset, ls.Offset+ls.ChunkSize)
	ls.Offset = trueOffset + len(lines)
	var filteredLines []ds.LogLine

	for _, line := range lines {
		if ls.Searcher == nil || ls.Searcher.Match(line) {
			filteredLines = append(filteredLines, line)
		}
	}

	if eof {
		ls.Reset()
	}
	return filteredLines, eof, nil
}
