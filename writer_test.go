/*
Package m3u8. Playlist generation tests.

Copyright 2013-2019, 2023 The Project Developers.
See the AUTHORS and LICENSE files at the top-level directory of this distribution
and at https://github.com/grafov/m3u8/

ॐ तारे तुत्तारे तुरे स्व
*/
package m3u8

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Check how master and media playlists implement common Playlist interface
func TestInterfaceImplemented(t *testing.T) {
	m := NewMasterPlaylist()
	CheckType(t, m)

	p, e := NewMediaPlaylist(1, 2)
	require.NoError(t, e)

	CheckType(t, p)
}

// Create new media playlist with wrong size (must be failed)
func TestCreateMediaPlaylistWithWrongSize(t *testing.T) {
	_, e := NewMediaPlaylist(2, 1) // wrong winsize
	require.Error(t, e)
}

// Tests the last method on media playlist
func TestLastSegmentMediaPlaylist(t *testing.T) {
	p, _ := NewMediaPlaylist(5, 5)
	require.Equal(t, p.last(), uint(4))

	for i := range uint(5) {
		_ = p.Append("uri.ts", 4, "")
		require.Equal(t, p.last(), i)
	}
}

// Create new media playlist
// Add two segments to media playlist
func TestAddSegmentToMediaPlaylist(t *testing.T) {
	p, e := NewMediaPlaylist(1, 2)
	require.NoError(t, e)

	e = p.Append("test01.ts", 10.0, "title")
	require.NoError(t, e)

	require.Equal(t, p.Segments[0].URI, "test01.ts")
	require.Equal(t, p.Segments[0].Duration, 10.0)
	require.Equal(t, p.Segments[0].Title, "title")
	require.Zero(t, p.Segments[0].SeqId)
}

func TestAppendSegmentToMediaPlaylist(t *testing.T) {
	p, _ := NewMediaPlaylist(2, 2)
	e := p.AppendSegment(&MediaSegment{Duration: 10})
	require.NoError(t, e)
	require.Equal(t, p.TargetDuration, float64(10))

	e = p.AppendSegment(&MediaSegment{Duration: 10})
	require.NoError(t, e)

	e = p.AppendSegment(&MediaSegment{Duration: 10})
	require.ErrorIs(t, e, ErrPlaylistFull)
	require.Equal(t, p.Count(), uint(2))
	require.Zero(t, p.SeqNo)
	require.Zero(t, p.Segments[0].SeqId)
	require.Equal(t, p.Segments[1].SeqId, uint64(1))
}

// Create new media playlist
// Add three segments to media playlist
// Set discontinuity tag for the 2nd segment.
func TestDiscontinuityForMediaPlaylist(t *testing.T) {
	var e error
	p, e := NewMediaPlaylist(3, 4)
	require.NoError(t, e)
	p.Close()

	e = p.Append("test01.ts", 5.0, "")
	require.NoError(t, e)

	e = p.Append("test02.ts", 6.0, "")
	require.NoError(t, e)

	e = p.SetDiscontinuity(1.0)
	require.NoError(t, e)

	e = p.Append("test03.ts", 6.0, "")
	require.NoError(t, e)

	// fmt.Println(p.Encode().String())
}

// Create new media playlist
// Add three segments to media playlist
// Set program date and time for 2nd segment.
// Set discontinuity tag for the 2nd segment.
func TestProgramDateTimeForMediaPlaylist(t *testing.T) {
	var e error
	p, e := NewMediaPlaylist(3, 4)
	require.NoError(t, e)
	p.Close()

	e = p.Append("test01.ts", 5.0, "")
	require.NoError(t, e)

	e = p.Append("test02.ts", 6.0, "")
	require.NoError(t, e)

	loc, _ := time.LoadLocation("Europe/Moscow")
	e = p.SetProgramDateTime(time.Date(2010, time.November, 30, 16, 25, 0, 125*1e6, loc))
	require.NoError(t, e)

	e = p.SetDiscontinuity(1.0)
	require.NoError(t, e)

	e = p.Append("test03.ts", 6.0, "")
	require.NoError(t, e)

	e = p.SetDiscontinuity(1.0)
	require.NoError(t, e)

	e = p.Append("test03.ts", 6.0, "")
	require.NoError(t, e)
	// fmt.Println(p.Encode().String())
}

// Create new media playlist
// Add two segments to media playlist with duration 9.0 and 9.1.
// Target duration must be set to nearest greater integer (= 10).
func TestTargetDurationForMediaPlaylist(t *testing.T) {
	p, e := NewMediaPlaylist(1, 2)
	require.NoError(t, e)

	e = p.Append("test01.ts", 9.0, "")
	require.NoError(t, e)

	e = p.Append("test02.ts", 9.1, "")
	require.NoError(t, e)

	require.GreaterOrEqual(t, p.TargetDuration, 10.0)
}

// Create new media playlist with capacity 10 elements
// Try to add 11 segments to media playlist (oversize error)
func TestOverAddSegmentsToMediaPlaylist(t *testing.T) {
	p, e := NewMediaPlaylist(1, 10)
	require.NoError(t, e)
	for i := range 11 {
		e = p.Append(fmt.Sprintf("test%d.ts", i), 5.0, "")
		if i < 10 {
			require.NoError(t, e, "As expected new segment #%d not assigned to a media playlist: %s due oversize\n", i, e)
		} else {
			require.Error(t, e)
		}
	}
}

func TestSetSCTE35(t *testing.T) {
	p, err := NewMediaPlaylist(1, 2)
	require.NoError(t, err)

	scte := &SCTE{Cue: "some cue"}
	err = p.SetSCTE35(scte)
	require.Error(t, err)

	err = p.Append("test01.ts", 10.0, "title")
	require.NoError(t, err)

	err = p.SetSCTE35(scte)
	require.NoError(t, err)

	require.Equal(t, p.Segments[0].SCTE, scte)
}

// Create new media playlist
// Add segment to media playlist
// Set SCTE
func TestSetSCTEForMediaPlaylist(t *testing.T) {
	tests := []struct {
		Cue      string
		ID       string
		Time     float64
		Expected string
	}{
		{"CueData1", "", 0, `#EXT-SCTE35:CUE="CueData1"` + "\n"},
		{"CueData2", "ID2", 0, `#EXT-SCTE35:CUE="CueData2",ID="ID2"` + "\n"},
		{"CueData3", "ID3", 3.141, `#EXT-SCTE35:CUE="CueData3",ID="ID3",TIME=3.141` + "\n"},
		{"CueData4", "", 3.1, `#EXT-SCTE35:CUE="CueData4",TIME=3.1` + "\n"},
		{"CueData5", "", 3.0, `#EXT-SCTE35:CUE="CueData5",TIME=3` + "\n"},
	}

	for _, test := range tests {
		p, e := NewMediaPlaylist(1, 1)
		require.NoError(t, e)

		e = p.Append("test01.ts", 5.0, "")
		require.NoError(t, e)

		e = p.SetSCTE(test.Cue, test.ID, test.Time)
		require.NoError(t, e)

		require.Contains(t, p.String(), test.Expected)
	}
}

// Create new media playlist
// Add segment to media playlist
// Set encryption key
func TestSetKeyForMediaPlaylist(t *testing.T) {
	tests := []struct {
		KeyFormat         string
		KeyFormatVersions string
		ExpectVersion     uint8
	}{
		{"", "", 3},
		{"Format", "", 5},
		{"", "Version", 5},
		{"Format", "Version", 5},
	}

	for _, test := range tests {
		p, e := NewMediaPlaylist(3, 5)
		require.NoError(t, e)

		e = p.Append("test01.ts", 5.0, "")
		require.NoError(t, e)

		e = p.SetKey("AES-128", "https://example.com", "iv", test.KeyFormat, test.KeyFormatVersions)
		require.NoError(t, e)

		require.Equal(t, p.ver, test.ExpectVersion)
	}
}

// Create new media playlist
// Add segment to media playlist
// Set encryption key
func TestSetDefaultKeyForMediaPlaylist(t *testing.T) {
	tests := []struct {
		KeyFormat         string
		KeyFormatVersions string
		ExpectVersion     uint8
	}{
		{"", "", 3},
		{"Format", "", 5},
		{"", "Version", 5},
		{"Format", "Version", 5},
	}

	for _, test := range tests {
		p, e := NewMediaPlaylist(3, 5)
		require.NoError(t, e)

		e = p.SetDefaultKey("AES-128", "https://example.com", "iv", test.KeyFormat, test.KeyFormatVersions)
		require.NoError(t, e)

		require.Equal(t, p.ver, test.ExpectVersion)
	}
}

// Create new media playlist
// Set default map
func TestSetDefaultMapForMediaPlaylist(t *testing.T) {
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)
	p.SetDefaultMap("https://example.com", 1000*1024, 1024*1024)

	expected := `EXT-X-MAP:URI="https://example.com",BYTERANGE=1024000@1048576`
	require.Contains(t, p.String(), expected)
}

// Create new media playlist
// Add segment to media playlist
// Set map on segment
func TestSetMapForMediaPlaylist(t *testing.T) {
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)

	e = p.Append("test01.ts", 5.0, "")
	require.NoError(t, e)

	e = p.SetMap("https://example.com", 1000*1024, 1024*1024)
	require.NoError(t, e)

	expected := `EXT-X-MAP:URI="https://example.com",BYTERANGE=1024000@1048576
#EXTINF:5.000,
test01.ts`
	require.Contains(t, p.String(), expected)
}

// Create new media playlist
// Set default map
// Add segment to media playlist
// Set map on segment (should be ignored when encoding)
func TestEncodeMediaPlaylistWithDefaultMap(t *testing.T) {
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)
	p.SetDefaultMap("https://example.com", 1000*1024, 1024*1024)

	e = p.Append("test01.ts", 5.0, "")
	require.NoError(t, e)

	e = p.SetMap("https://notencoded.com", 1000*1024, 1024*1024)
	require.NoError(t, e)

	encoded := p.String()
	expected := `EXT-X-MAP:URI="https://example.com",BYTERANGE=1024000@1048576`
	require.Contains(t, encoded, expected)

	ignored := `EXT-X-MAP:URI="https://notencoded.com"`
	require.NotContains(t, encoded, ignored)
}

// Create new media playlist
// Add custom playlist tag
// Add segment with custom tag
func TestEncodeMediaPlaylistWithCustomTags(t *testing.T) {
	p, e := NewMediaPlaylist(1, 1)
	require.NoError(t, e)

	customPTag := &MockCustomTag{
		name:          "#CustomPTag",
		encodedString: "#CustomPTag",
	}
	p.SetCustomTag(customPTag)

	customEmptyPTag := &MockCustomTag{
		name:          "#CustomEmptyPTag",
		encodedString: "",
	}
	p.SetCustomTag(customEmptyPTag)

	e = p.Append("test01.ts", 5.0, "")
	require.NoError(t, e)

	customSTag := &MockCustomTag{
		name:          "#CustomSTag",
		encodedString: "#CustomSTag",
	}
	e = p.SetCustomSegmentTag(customSTag)
	require.NoError(t, e)

	customEmptySTag := &MockCustomTag{
		name:          "#CustomEmptySTag",
		encodedString: "",
	}
	e = p.SetCustomSegmentTag(customEmptySTag)
	require.NoError(t, e)

	encoded := p.String()
	expectedStrings := []string{"#CustomPTag", "#CustomSTag"}
	for _, expected := range expectedStrings {
		require.Contains(t, encoded, expected)
	}
	unexpectedStrings := []string{"#CustomEmptyPTag", "#CustomEmptySTag"}
	for _, unexpected := range unexpectedStrings {
		require.NotContains(t, encoded, unexpected)
	}
}

// Create new media playlist
// Add two segments to media playlist
// Encode structures to HLS
func TestEncodeMediaPlaylist(t *testing.T) {
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)

	e = p.Append("test01.ts", 5.0, "")
	require.NoError(t, e)

	p.DurationAsInt(true)
	// fmt.Println(p.Encode().String())
}

// Create new media playlist
// Add 10 segments to media playlist
// Test iterating over segments
func TestLoopSegmentsOfMediaPlaylist(t *testing.T) {
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)
	for i := 0; i < 5; i++ {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.0, ""))
	}
	p.DurationAsInt(true)
	// fmt.Println(p.Encode().String())
}

// Create new media playlist with capacity 5
// Add 5 segments and 5 unique keys
// Test correct keys set on correct segments
func TestEncryptionKeysInMediaPlaylist(t *testing.T) {
	p, _ := NewMediaPlaylist(5, 5)
	// Add 5 segments and set custom encryption key
	for i := range uint(5) {
		uri := fmt.Sprintf("uri-%d", i)
		expected := &Key{
			Method:            "AES-128",
			URI:               uri,
			IV:                fmt.Sprintf("%d", i),
			Keyformat:         "identity",
			Keyformatversions: "1",
		}
		err := p.Append(uri+".ts", 4, "")
		require.NoError(t, err)

		err = p.SetKey(expected.Method, expected.URI, expected.IV, expected.Keyformat, expected.Keyformatversions)
		require.NoError(t, err)

		require.Equal(t, p.Segments[i].Key, expected)
	}
}

func TestEncryptionKeyMethodNoneInMediaPlaylist(t *testing.T) {
	expected := `#EXT-X-KEY:METHOD=NONE
#EXTINF:4.000,
segment-2.ts`
	p, e := NewMediaPlaylist(5, 5)

	require.NoError(t, e)
	require.NoError(t, p.Append("segment-1.ts", 4, ""))
	require.NoError(t, p.SetKey("AES-128", "key-uri", "iv", "identity", "1"))
	require.NoError(t, p.Append("segment-2.ts", 4, ""))
	require.NoError(t, p.SetKey("NONE", "", "", "", ""))
	require.Contains(t, p.String(), expected)
}

// Create new media playlist
// Add 10 segments to media playlist
// Encode structure to HLS with integer target durations
func TestMediaPlaylistWithIntegerDurations(t *testing.T) {
	p, e := NewMediaPlaylist(3, 10)
	require.NoError(t, e)
	for i := 0; i < 9; i++ {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.6, ""))
	}
	p.DurationAsInt(false)
	//	fmt.Println(p.Encode().String())
}

// Create new media playlist
// Add 9 segments to media playlist
// 11 times encode structure to HLS with integer target durations
// Last playlist must be empty
func TestMediaPlaylistWithEmptyMedia(t *testing.T) {
	p, e := NewMediaPlaylist(3, 10)
	require.NoError(t, e)

	for i := range 10 {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.6, ""))
	}
	for range 11 {
		// fmt.Println(p.Encode().String())
		p.Remove()
	} // TODO add check for buffers equality
}

// Create new media playlist with winsize == capacity
func TestMediaPlaylistWinsize(t *testing.T) {
	p, e := NewMediaPlaylist(6, 6)
	require.NoError(t, e)

	for i := range 10 {
		p.Slide(fmt.Sprintf("test%d.ts", i), 5.6, "")
		// fmt.Println(p.Encode().String()) // TODO check playlist sizes and mediasequence values
	}
}

// Create new media playlist as sliding playlist.
// Close it.
func TestClosedMediaPlaylist(t *testing.T) {
	p, e := NewMediaPlaylist(1, 10)
	require.NoError(t, e)
	defer p.Close()

	for i := range 10 {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.0, ""))
	}
}

// Create new media playlist as sliding playlist.
func TestLargeMediaPlaylistWithParallel(t *testing.T) {
	t.Skip()
	testCount := 10
	expect, err := os.ReadFile("sample-playlists/media-playlist-large.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var errChan = make(chan error, 1)
	for i := 0; i < testCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, err := os.Open("sample-playlists/media-playlist-large.m3u8")
			if err != nil {
				errChan <- err
				return
			}
			p, err := NewMediaPlaylist(50000, 50000)
			if err != nil {
				errChan <- err
				return
			}
			if err = p.DecodeFrom(bufio.NewReader(f), true); err != nil {
				errChan <- err
				return
			}

			actual := p.Encode().Bytes() // disregard output
			if !bytes.Equal(expect, actual) {
				errChan <- fmt.Errorf("not matched")
			}
		}()
	}
	wg.Wait()
	if err := <-errChan; err != nil {
		t.Fatal(err)
	}
}

func TestMediaVersion(t *testing.T) {
	m, e := NewMediaPlaylist(3, 3)
	require.NoError(t, e)

	m.ver = 5
	require.Equal(t, m.Version(), m.ver)
}

func TestMediaSetVersion(t *testing.T) {
	m, e := NewMediaPlaylist(3, 3)
	require.NoError(t, e)

	m.ver = 3
	m.SetVersion(5)
	require.Equal(t, m.ver, uint8(5))
}

func TestMediaWinSize(t *testing.T) {
	m, e := NewMediaPlaylist(3, 3)
	require.NoError(t, e)
	require.Equal(t, m.WinSize(), m.winsize)
}

func TestMediaSetWinSize(t *testing.T) {
	m, err := NewMediaPlaylist(3, 5)
	require.NoError(t, err)

	err = m.SetWinSize(5)
	require.NoError(t, err)

	require.Equal(t, m.winsize, uint(5))

	// Check winsize cannot exceed capacity
	err = m.SetWinSize(99999)
	require.Error(t, err)

	// Ensure winsize didn't change
	require.Equal(t, m.winsize, uint(5))
}

func TestIndependentSegments(t *testing.T) {
	m := NewMasterPlaylist()
	require.False(t, m.IndependentSegments())

	m.SetIndependentSegments(true)
	require.True(t, m.IndependentSegments())
	require.Contains(t, m.Encode().String(), "#EXT-X-INDEPENDENT-SEGMENTS")
}

// Create new media playlist
// Set default map
func TestStartTimeOffset(t *testing.T) {
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)
	p.StartTime = 3.4

	expected := `#EXT-X-START:TIME-OFFSET=3.4`
	require.Contains(t, p.String(), expected)
}

func TestMediaPlaylist_Slide(t *testing.T) {
	m, e := NewMediaPlaylist(3, 4)
	require.NoError(t, e)

	require.NoError(t, m.Append("t00.ts", 10, ""))
	require.NoError(t, m.Append("t01.ts", 10, ""))
	require.NoError(t, m.Append("t02.ts", 10, ""))
	require.NoError(t, m.Append("t03.ts", 10, ""))
	require.Equal(t, m.Count(), uint(4))
	require.Zero(t, m.SeqNo)

	for i := range uint(3) {
		segIdx := (m.head + i) % m.capacity
		segUri := fmt.Sprintf("t%02d.ts", i)
		seg := m.Segments[segIdx]
		require.Equal(t, seg.URI, segUri)
		require.Equal(t, seg.SeqId, uint64(i))
	}

	m.Slide("t04.ts", 10, "")
	require.Equal(t, m.Count(), uint(4))
	require.Equal(t, m.SeqNo, uint64(1))

	for idx, seqId := uint(0), uint(1); idx < 3; idx, seqId = idx+1, seqId+1 {
		segIdx := (m.head + idx) % m.capacity
		segUri := fmt.Sprintf("t%02d.ts", seqId)
		seg := m.Segments[segIdx]
		require.Equal(t, seg.URI, segUri)
		require.Equal(t, seg.SeqId, uint64(seqId))
	}

	m.Slide("t05.ts", 10, "")
	m.Slide("t06.ts", 10, "")
	require.Equal(t, m.Count(), uint(4))
	require.Equal(t, m.SeqNo, uint64(3))

	for idx, seqId := uint(0), uint(3); idx < 3; idx, seqId = idx+1, seqId+1 {
		segIdx := (m.head + idx) % m.capacity
		segUri := fmt.Sprintf("t%02d.ts", seqId)
		seg := m.Segments[segIdx]
		require.Equal(t, seg.URI, segUri)
		require.Equal(t, seg.SeqId, uint64(seqId))
	}
}

// Create new master playlist without params
// Add media playlist
func TestNewMasterPlaylist(t *testing.T) {
	m := NewMasterPlaylist()
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)

	for i := range 5 {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.0, ""))
	}
	m.Append("chunklist1.m3u8", p, VariantParams{})
}

// Create new master playlist without params
// Add media playlist with Alternatives
func TestNewMasterPlaylistWithAlternatives(t *testing.T) {
	m := NewMasterPlaylist()
	audioUri := fmt.Sprintf("%s/rendition.m3u8", "800")
	audioAlt := &Alternative{
		GroupId:    "audio",
		URI:        audioUri,
		Type:       "AUDIO",
		Name:       "main",
		Default:    true,
		Autoselect: "YES",
		Language:   "english",
	}
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)

	for i := range 5 {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.0, ""))
	}
	m.Append("chunklist1.m3u8", p, VariantParams{Alternatives: []*Alternative{audioAlt}})

	require.Equal(t, m.ver, uint8(4))

	expected := `#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio",NAME="main",DEFAULT=YES,AUTOSELECT=YES,LANGUAGE="english",URI="800/rendition.m3u8"`
	require.Contains(t, m.String(), expected)
}

// Create new master playlist supporting CLOSED-CAPTIONS=NONE
func TestNewMasterPlaylistWithClosedCaptionEqNone(t *testing.T) {
	m := NewMasterPlaylist()

	vp := &VariantParams{
		ProgramId:  0,
		Bandwidth:  8000,
		Codecs:     "avc1",
		Resolution: "1280x720",
		Audio:      "audio0",
		Captions:   "NONE",
	}

	p, err := NewMediaPlaylist(1, 1)
	require.NoError(t, err)

	m.Append("eng_rendition_rendition.m3u8", p, *vp)

	expected := "CLOSED-CAPTIONS=NONE"
	require.Contains(t, m.String(), expected)

	// quotes need to be include if not eq NONE
	vp.Captions = "CC1"
	m2 := NewMasterPlaylist()
	m2.Append("eng_rendition_rendition.m3u8", p, *vp)
	expected = `CLOSED-CAPTIONS="CC1"`
	require.Contains(t, m2.String(), expected)
}

// Create new master playlist with params
// Add media playlist
func TestNewMasterPlaylistWithParams(t *testing.T) {
	m := NewMasterPlaylist()
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)
	for i := range 5 {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.0, ""))
	}
	m.Append("chunklist1.m3u8", p, VariantParams{ProgramId: 123, Bandwidth: 1500000, Resolution: "576x480"})
}

// Create new master playlist
// Add media playlist with existing query params in URI
// Append more query params and ensure it encodes correctly
func TestEncodeMasterPlaylistWithExistingQuery(t *testing.T) {
	m := NewMasterPlaylist()
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)

	for i := range 5 {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.0, ""))
	}
	m.Append("chunklist1.m3u8?k1=v1&k2=v2", p, VariantParams{ProgramId: 123, Bandwidth: 1500000, Resolution: "576x480"})
	m.Args = "k3=v3"
	require.Contains(t, m.String(), `chunklist1.m3u8?k1=v1&k2=v2&k3=v3`)
}

// Create new master playlist
// Add media playlist
// Encode structures to HLS
func TestEncodeMasterPlaylist(t *testing.T) {
	m := NewMasterPlaylist()
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)

	for i := range 5 {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.0, ""))
	}
	m.Append("chunklist1.m3u8", p, VariantParams{ProgramId: 123, Bandwidth: 1500000, Resolution: "576x480"})
	m.Append("chunklist2.m3u8", p, VariantParams{ProgramId: 123, Bandwidth: 1500000, Resolution: "576x480"})
}

// Create new master playlist with Name tag in EXT-X-STREAM-INF
func TestEncodeMasterPlaylistWithStreamInfName(t *testing.T) {
	m := NewMasterPlaylist()
	p, e := NewMediaPlaylist(3, 5)
	require.NoError(t, e)

	for i := range 5 {
		require.NoError(t, p.Append(fmt.Sprintf("test%d.ts", i), 5.0, ""))
	}
	m.Append("chunklist1.m3u8", p, VariantParams{ProgramId: 123, Bandwidth: 3000000, Resolution: "1152x960", Name: "HD 960p"})

	require.Equal(t, m.Variants[0].Name, "HD 960p")
	require.Contains(t, m.String(), `NAME="HD 960p"`)
}

func TestEncodeMasterPlaylistWithCustomTags(t *testing.T) {
	m := NewMasterPlaylist()
	customMTag := &MockCustomTag{
		name:          "#CustomMTag",
		encodedString: "#CustomMTag",
	}
	m.SetCustomTag(customMTag)

	encoded := m.String()
	expected := "#CustomMTag"

	require.Contains(t, encoded, expected)
}

func TestMasterVersion(t *testing.T) {
	m := NewMasterPlaylist()
	m.ver = 5
	require.Equal(t, m.Version(), m.ver)
}

func TestMasterSetVersion(t *testing.T) {
	m := NewMasterPlaylist()
	m.ver = 3
	m.SetVersion(5)
	require.Equal(t, m.ver, uint8(5))
}

/******************************
 *  Code generation examples  *
 ******************************/

// Create new media playlist
// Add two segments to media playlist
// Print it
func ExampleMediaPlaylist_String() {
	p, _ := NewMediaPlaylist(1, 2)
	p.Append("test01.ts", 5.0, "")
	p.Append("test02.ts", 6.0, "")
	fmt.Printf("%s\n", p)
	// Output:
	// #EXTM3U
	// #EXT-X-VERSION:3
	// #EXT-X-MEDIA-SEQUENCE:0
	// #EXT-X-TARGETDURATION:6
	// #EXTINF:5.000,
	// test01.ts
}

// Create new media playlist
// Add two segments to media playlist
// Print it
func ExampleMediaPlaylist_String_winsize0() {
	p, _ := NewMediaPlaylist(0, 2)
	p.Append("test01.ts", 5.0, "")
	p.Append("test02.ts", 6.0, "")
	fmt.Printf("%s\n", p)
	// Output:
	// #EXTM3U
	// #EXT-X-VERSION:3
	// #EXT-X-MEDIA-SEQUENCE:0
	// #EXT-X-TARGETDURATION:6
	// #EXTINF:5.000,
	// test01.ts
	// #EXTINF:6.000,
	// test02.ts
}

// Create new media playlist
// Add two segments to media playlist
// Print it
func ExampleMediaPlaylist_String_winsize0_vod() {
	p, _ := NewMediaPlaylist(0, 2)
	p.Append("test01.ts", 5.0, "")
	p.Append("test02.ts", 6.0, "")
	p.Close()
	fmt.Printf("%s\n", p)
	// Output:
	// #EXTM3U
	// #EXT-X-VERSION:3
	// #EXT-X-MEDIA-SEQUENCE:0
	// #EXT-X-TARGETDURATION:6
	// #EXTINF:5.000,
	// test01.ts
	// #EXTINF:6.000,
	// test02.ts
	// #EXT-X-ENDLIST
}

// Create new master playlist
// Add media playlist
// Encode structures to HLS
func ExampleMasterPlaylist_String() {
	m := NewMasterPlaylist()
	p, _ := NewMediaPlaylist(3, 5)
	for i := 0; i < 5; i++ {
		p.Append(fmt.Sprintf("test%d.ts", i), 5.0, "")
	}
	m.Append("chunklist1.m3u8", p, VariantParams{ProgramId: 123, Bandwidth: 1500000, AverageBandwidth: 1500000, Resolution: "576x480", FrameRate: 25.000})
	m.Append("chunklist2.m3u8", p, VariantParams{ProgramId: 123, Bandwidth: 1500000, AverageBandwidth: 1500000, Resolution: "576x480", FrameRate: 25.000})
	fmt.Printf("%s", m)
	// Output:
	// #EXTM3U
	// #EXT-X-VERSION:3
	// #EXT-X-STREAM-INF:PROGRAM-ID=123,BANDWIDTH=1500000,AVERAGE-BANDWIDTH=1500000,RESOLUTION=576x480,FRAME-RATE=25.000
	// chunklist1.m3u8
	// #EXT-X-STREAM-INF:PROGRAM-ID=123,BANDWIDTH=1500000,AVERAGE-BANDWIDTH=1500000,RESOLUTION=576x480,FRAME-RATE=25.000
	// chunklist2.m3u8
}

func ExampleMasterPlaylist_String_with_hlsv7() {
	m := NewMasterPlaylist()
	m.SetVersion(7)
	m.SetIndependentSegments(true)
	p, _ := NewMediaPlaylist(3, 5)
	m.Append("hdr10_1080/prog_index.m3u8", p, VariantParams{AverageBandwidth: 7964551, Bandwidth: 12886714, VideoRange: "PQ", Codecs: "hvc1.2.4.L123.B0", Resolution: "1920x1080", FrameRate: 23.976, Captions: "NONE", HDCPLevel: "TYPE-0"})
	m.Append("hdr10_1080/iframe_index.m3u8", p, VariantParams{Iframe: true, AverageBandwidth: 364552, Bandwidth: 905053, VideoRange: "PQ", Codecs: "hvc1.2.4.L123.B0", Resolution: "1920x1080", HDCPLevel: "TYPE-0"})
	fmt.Printf("%s", m)
	// Output:
	// #EXTM3U
	// #EXT-X-VERSION:7
	// #EXT-X-INDEPENDENT-SEGMENTS
	// #EXT-X-STREAM-INF:PROGRAM-ID=0,BANDWIDTH=12886714,AVERAGE-BANDWIDTH=7964551,CODECS="hvc1.2.4.L123.B0",RESOLUTION=1920x1080,CLOSED-CAPTIONS=NONE,FRAME-RATE=23.976,VIDEO-RANGE=PQ,HDCP-LEVEL=TYPE-0
	// hdr10_1080/prog_index.m3u8
	// #EXT-X-I-FRAME-STREAM-INF:PROGRAM-ID=0,BANDWIDTH=905053,AVERAGE-BANDWIDTH=364552,CODECS="hvc1.2.4.L123.B0",RESOLUTION=1920x1080,VIDEO-RANGE=PQ,HDCP-LEVEL=TYPE-0,URI="hdr10_1080/iframe_index.m3u8"
}

func ExampleMediaPlaylist_Segments_scte35_oatcls() {
	f, _ := os.Open("sample-playlists/media-playlist-with-oatcls-scte35.m3u8")
	p, _, _ := DecodeFrom(bufio.NewReader(f), true)
	pp := p.(*MediaPlaylist)
	fmt.Print(pp)
	// Output:
	// #EXTM3U
	// #EXT-X-VERSION:3
	// #EXT-X-MEDIA-SEQUENCE:0
	// #EXT-X-TARGETDURATION:10
	// #EXT-OATCLS-SCTE35:/DAlAAAAAAAAAP/wFAUAAAABf+/+ANgNkv4AFJlwAAEBAQAA5xULLA==
	// #EXT-X-CUE-OUT:15
	// #EXTINF:8.844,
	// media0.ts
	// #EXT-X-CUE-OUT-CONT:ElapsedTime=8.844,Duration=15,SCTE35=/DAlAAAAAAAAAP/wFAUAAAABf+/+ANgNkv4AFJlwAAEBAQAA5xULLA==
	// #EXTINF:6.156,
	// media1.ts
	// #EXT-X-CUE-IN
	// #EXTINF:3.844,
	// media2.ts
}

func ExampleMediaPlaylist_Segments_scte35_67_2014() {
	f, _ := os.Open("sample-playlists/media-playlist-with-scte35.m3u8")
	p, _, _ := DecodeFrom(bufio.NewReader(f), true)
	pp := p.(*MediaPlaylist)
	fmt.Print(pp)
	// Output:
	// #EXTM3U
	// #EXT-X-VERSION:3
	// #EXT-X-MEDIA-SEQUENCE:0
	// #EXT-X-TARGETDURATION:10
	// #EXTINF:10.000,
	// media0.ts
	// #EXTINF:10.000,
	// media1.ts
	// #EXT-SCTE35:CUE="/DAIAAAAAAAAAAAQAAZ/I0VniQAQAgBDVUVJQAAAAH+cAAAAAA==",ID="123",TIME=123.12
	// #EXTINF:10.000,
	// media2.ts
}

// Range over segments of media playlist. Check for ring buffer corner
// cases.
func ExampleMediaPlaylist_GetAllSegments() {
	m, _ := NewMediaPlaylist(3, 3)
	_ = m.Append("t00.ts", 10, "")
	_ = m.Append("t01.ts", 10, "")
	_ = m.Append("t02.ts", 10, "")
	for _, v := range m.GetAllSegments() {
		fmt.Printf("%s\n", v.URI)
	}
	m.Remove()
	m.Remove()
	m.Remove()
	_ = m.Append("t03.ts", 10, "")
	_ = m.Append("t04.ts", 10, "")
	for _, v := range m.GetAllSegments() {
		fmt.Printf("%s\n", v.URI)
	}
	m.Remove()
	m.Remove()
	_ = m.Append("t05.ts", 10, "")
	_ = m.Append("t06.ts", 10, "")
	m.Remove()
	m.Remove()
	// empty because removed two elements
	for _, v := range m.GetAllSegments() {
		fmt.Printf("%s\n", v.URI)
	}
	// Output:
	// t00.ts
	// t01.ts
	// t02.ts
	// t03.ts
	// t04.ts
}

/****************
 *  Benchmarks  *
 ****************/

func BenchmarkEncodeMasterPlaylist(b *testing.B) {
	f, err := os.Open("sample-playlists/master.m3u8")
	require.NoError(b, err)

	p := NewMasterPlaylist()
	require.NoError(b, p.DecodeFrom(bufio.NewReader(f), true))

	for range b.N {
		p.ResetCache()
		_ = p.Encode() // disregard output
	}
}

func BenchmarkEncodeMediaPlaylist(b *testing.B) {
	f, err := os.Open("sample-playlists/media-playlist-large.m3u8")
	require.NoError(b, err)

	p, err := NewMediaPlaylist(50000, 50000)
	require.NoError(b, err)

	require.NoError(b, p.DecodeFrom(bufio.NewReader(f), true))

	for range b.N {
		p.ResetCache()
		_ = p.Encode() // disregard output
	}
}
