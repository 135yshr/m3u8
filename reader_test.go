/*
Playlist parsing tests.

Copyright 2013-2019 The Project Developers.
See the AUTHORS and LICENSE files at the top-level directory of this distribution
and at https://github.com/grafov/m3u8/

ॐ तारे तुत्तारे तुरे स्व
*/
package m3u8

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDecodeMasterPlaylist(t *testing.T) {
	f, err := os.Open("sample-playlists/master.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	// check parsed values
	require.Equal(t, p.ver, uint8(3))
	require.Equal(t, len(p.Variants), 5)
	// TODO check other values
	// fmt.Println(p.Encode().String())
}

func TestDecodeMasterPlaylistWithMultipleCodecs(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-multiple-codecs.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	// check parsed values
	require.Equal(t, p.ver, uint8(3))
	require.Equal(t, len(p.Variants), 5)

	for _, v := range p.Variants {
		require.Equal(t, v.Codecs, "avc1.42c015,mp4a.40.2")
	}
	// TODO check other values
	// fmt.Println(p.Encode().String())
}

func TestDecodeMasterPlaylistWithAlternatives(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-alternatives.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	// check parsed values
	require.Equal(t, p.ver, uint8(3))
	require.Equal(t, len(p.Variants), 4)

	require.Len(t, p.Variants[0].Alternatives, 3)
	require.Len(t, p.Variants[1].Alternatives, 3)
	require.Len(t, p.Variants[2].Alternatives, 3)
	require.Len(t, p.Variants[3].Alternatives, 0)
}

func TestDecodeMasterPlaylistWithAlternativesB(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-alternatives-b.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	// check parsed values
	require.Equal(t, p.ver, uint8(3))
	require.Equal(t, len(p.Variants), 4)
	require.Len(t, p.Variants[0].Alternatives, 3)
	require.Len(t, p.Variants[1].Alternatives, 3)
	require.Len(t, p.Variants[2].Alternatives, 3)
	require.Len(t, p.Variants[3].Alternatives, 0)
}

func TestDecodeMasterPlaylistWithClosedCaptionEqNone(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-closed-captions-eq-none.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	require.Equal(t, len(p.Variants), 3)

	for _, v := range p.Variants {
		require.Equal(t, v.Captions, "NONE")
	}
}

// Decode a master playlist with Name tag in EXT-X-STREAM-INF
func TestDecodeMasterPlaylistWithStreamInfName(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-stream-inf-name.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)
	for _, variant := range p.Variants {
		require.NotEmpty(t, variant.Name)
	}
}

func TestDecodeMediaPlaylistByteRange(t *testing.T) {
	f, err := os.Open("sample-playlists/media-playlist-with-byterange.m3u8")
	require.NoError(t, err)

	p, err := NewMediaPlaylist(3, 3)
	require.NoError(t, err)

	err = p.DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	expected := []*MediaSegment{
		{URI: "video.ts", Duration: 10, Limit: 75232, SeqId: 0},
		{URI: "video.ts", Duration: 10, Limit: 82112, Offset: 752321, SeqId: 1},
		{URI: "video.ts", Duration: 10, Limit: 69864, SeqId: 2},
	}
	for i, seg := range p.Segments {
		require.Equal(t, *seg, *expected[i])
	}
}

// Decode a master playlist with i-frame-stream-inf
func TestDecodeMasterPlaylistWithIFrameStreamInf(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-i-frame-stream-inf.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	expected := map[int]*Variant{
		86000:  {URI: "low/iframe.m3u8", VariantParams: VariantParams{Bandwidth: 86000, ProgramId: 1, Codecs: "c1", Resolution: "1x1", Video: "1", Iframe: true}},
		150000: {URI: "mid/iframe.m3u8", VariantParams: VariantParams{Bandwidth: 150000, ProgramId: 1, Codecs: "c2", Resolution: "2x2", Video: "2", Iframe: true}},
		550000: {URI: "hi/iframe.m3u8", VariantParams: VariantParams{Bandwidth: 550000, ProgramId: 1, Codecs: "c2", Resolution: "2x2", Video: "2", Iframe: true}},
	}
	for _, variant := range p.Variants {
		for k, expect := range expected {
			if reflect.DeepEqual(variant, expect) {
				delete(expected, k)
			}
		}
	}

	// Verify all variants have been deleted
	for _, expect := range expected {
		t.Errorf("not found:%+v", expect)
	}
}

func TestDecodeMasterPlaylistWithStreamInfAverageBandwidth(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-stream-inf-1.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	for _, variant := range p.Variants {
		require.NotZero(t, variant.AverageBandwidth)
	}
}

func TestDecodeMasterPlaylistWithStreamInfFrameRate(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-stream-inf-1.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	for _, variant := range p.Variants {
		require.NotZero(t, variant.FrameRate)
	}
}

func TestDecodeMasterPlaylistWithIndependentSegments(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-independent-segments.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)
	require.True(t, p.IndependentSegments())
}

func TestDecodeMasterWithHLSV7(t *testing.T) {
	f, err := os.Open("sample-playlists/master-with-hlsv7.m3u8")
	require.NoError(t, err)

	p := NewMasterPlaylist()
	err = p.DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)

	var unexpected []*Variant
	expected := map[string]VariantParams{
		"sdr_720/prog_index.m3u8":      {Bandwidth: 3971374, AverageBandwidth: 2778321, Codecs: "hvc1.2.4.L123.B0", Resolution: "1280x720", Captions: "NONE", VideoRange: "SDR", HDCPLevel: "NONE", FrameRate: 23.976},
		"sdr_1080/prog_index.m3u8":     {Bandwidth: 10022043, AverageBandwidth: 6759875, Codecs: "hvc1.2.4.L123.B0", Resolution: "1920x1080", Captions: "NONE", VideoRange: "SDR", HDCPLevel: "TYPE-0", FrameRate: 23.976},
		"sdr_2160/prog_index.m3u8":     {Bandwidth: 28058971, AverageBandwidth: 20985770, Codecs: "hvc1.2.4.L150.B0", Resolution: "3840x2160", Captions: "NONE", VideoRange: "SDR", HDCPLevel: "TYPE-1", FrameRate: 23.976},
		"dolby_720/prog_index.m3u8":    {Bandwidth: 5327059, AverageBandwidth: 3385450, Codecs: "dvh1.05.01", Resolution: "1280x720", Captions: "NONE", VideoRange: "PQ", HDCPLevel: "NONE", FrameRate: 23.976},
		"dolby_1080/prog_index.m3u8":   {Bandwidth: 12876596, AverageBandwidth: 7999361, Codecs: "dvh1.05.03", Resolution: "1920x1080", Captions: "NONE", VideoRange: "PQ", HDCPLevel: "TYPE-0", FrameRate: 23.976},
		"dolby_2160/prog_index.m3u8":   {Bandwidth: 30041698, AverageBandwidth: 24975091, Codecs: "dvh1.05.06", Resolution: "3840x2160", Captions: "NONE", VideoRange: "PQ", HDCPLevel: "TYPE-1", FrameRate: 23.976},
		"hdr10_720/prog_index.m3u8":    {Bandwidth: 5280654, AverageBandwidth: 3320040, Codecs: "hvc1.2.4.L123.B0", Resolution: "1280x720", Captions: "NONE", VideoRange: "PQ", HDCPLevel: "NONE", FrameRate: 23.976},
		"hdr10_1080/prog_index.m3u8":   {Bandwidth: 12886714, AverageBandwidth: 7964551, Codecs: "hvc1.2.4.L123.B0", Resolution: "1920x1080", Captions: "NONE", VideoRange: "PQ", HDCPLevel: "TYPE-0", FrameRate: 23.976},
		"hdr10_2160/prog_index.m3u8":   {Bandwidth: 29983769, AverageBandwidth: 24833402, Codecs: "hvc1.2.4.L150.B0", Resolution: "3840x2160", Captions: "NONE", VideoRange: "PQ", HDCPLevel: "TYPE-1", FrameRate: 23.976},
		"sdr_720/iframe_index.m3u8":    {Bandwidth: 593626, AverageBandwidth: 248586, Codecs: "hvc1.2.4.L123.B0", Resolution: "1280x720", Iframe: true, VideoRange: "SDR", HDCPLevel: "NONE"},
		"sdr_1080/iframe_index.m3u8":   {Bandwidth: 956552, AverageBandwidth: 399790, Codecs: "hvc1.2.4.L123.B0", Resolution: "1920x1080", Iframe: true, VideoRange: "SDR", HDCPLevel: "TYPE-0"},
		"sdr_2160/iframe_index.m3u8":   {Bandwidth: 1941397, AverageBandwidth: 826971, Codecs: "hvc1.2.4.L150.B0", Resolution: "3840x2160", Iframe: true, VideoRange: "SDR", HDCPLevel: "TYPE-1"},
		"dolby_720/iframe_index.m3u8":  {Bandwidth: 573073, AverageBandwidth: 232253, Codecs: "dvh1.05.01", Resolution: "1280x720", Iframe: true, VideoRange: "PQ", HDCPLevel: "NONE"},
		"dolby_1080/iframe_index.m3u8": {Bandwidth: 905037, AverageBandwidth: 365337, Codecs: "dvh1.05.03", Resolution: "1920x1080", Iframe: true, VideoRange: "PQ", HDCPLevel: "TYPE-0"},
		"dolby_2160/iframe_index.m3u8": {Bandwidth: 1893236, AverageBandwidth: 739114, Codecs: "dvh1.05.06", Resolution: "3840x2160", Iframe: true, VideoRange: "PQ", HDCPLevel: "TYPE-1"},
		"hdr10_720/iframe_index.m3u8":  {Bandwidth: 572673, AverageBandwidth: 232511, Codecs: "hvc1.2.4.L123.B0", Resolution: "1280x720", Iframe: true, VideoRange: "PQ", HDCPLevel: "NONE"},
		"hdr10_1080/iframe_index.m3u8": {Bandwidth: 905053, AverageBandwidth: 364552, Codecs: "hvc1.2.4.L123.B0", Resolution: "1920x1080", Iframe: true, VideoRange: "PQ", HDCPLevel: "TYPE-0"},
		"hdr10_2160/iframe_index.m3u8": {Bandwidth: 1895477, AverageBandwidth: 739757, Codecs: "hvc1.2.4.L150.B0", Resolution: "3840x2160", Iframe: true, VideoRange: "PQ", HDCPLevel: "TYPE-1"},
	}
	for _, variant := range p.Variants {
		var found bool
		for uri, vp := range expected {
			if variant == nil || variant.URI != uri {
				continue
			}
			if reflect.DeepEqual(variant.VariantParams, vp) {
				delete(expected, uri)
				found = true
			}
		}
		if !found {
			unexpected = append(unexpected, variant)
		}
	}

	// Verify all variants have been deleted
	for uri, expect := range expected {
		t.Errorf("not found: uri=%q %+v", uri, expect)
	}

	// Verify all unexpected variants have been found
	for _, unexpect := range unexpected {
		t.Errorf("found but not expecting:%+v", unexpect)
	}
}

/****************************
 * Begin Test MediaPlaylist *
 ****************************/

func TestDecodeMediaPlaylist(t *testing.T) {
	f, err := os.Open("sample-playlists/wowza-vod-chunklist.m3u8")
	require.NoError(t, err)

	p, err := NewMediaPlaylist(5, 798)
	require.NoError(t, err)

	err = p.DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	//fmt.Printf("Playlist object: %+v\n", p)
	// check parsed values
	require.Equal(t, p.ver, uint8(3))
	require.Equal(t, p.TargetDuration, 12.0)
	require.True(t, p.Closed)

	titles := []string{"Title 1", "Title 2", ""}
	for i, s := range p.Segments {
		if i > len(titles)-1 {
			break
		}
		require.Equal(t, s.Title, titles[i])
	}
	require.Equal(t, p.Count(), uint(522))

	var seqId, idx uint
	for seqId, idx = 1, 0; idx < p.Count(); seqId, idx = seqId+1, idx+1 {
		require.Equal(t, p.Segments[idx].SeqId, uint64(seqId))
	}
	// TODO check other values…
	//fmt.Println(p.Encode().String()), stream.Name}
}

func TestDecodeMediaPlaylistExtInfNonStrict2(t *testing.T) {
	header := `#EXTM3U
#EXT-X-TARGETDURATION:10
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:0
%s
`

	tests := []struct {
		strict      bool
		extInf      string
		wantError   bool
		wantSegment *MediaSegment
	}{
		// strict mode on
		{true, "#EXTINF:10.000,", false, &MediaSegment{Duration: 10.0, Title: ""}},
		{true, "#EXTINF:10.000,Title", false, &MediaSegment{Duration: 10.0, Title: "Title"}},
		{true, "#EXTINF:10.000,Title,Track", false, &MediaSegment{Duration: 10.0, Title: "Title,Track"}},
		{true, "#EXTINF:invalid,", true, nil},
		{true, "#EXTINF:10.000", true, nil},

		// strict mode off
		{false, "#EXTINF:10.000,", false, &MediaSegment{Duration: 10.0, Title: ""}},
		{false, "#EXTINF:10.000,Title", false, &MediaSegment{Duration: 10.0, Title: "Title"}},
		{false, "#EXTINF:10.000,Title,Track", false, &MediaSegment{Duration: 10.0, Title: "Title,Track"}},
		{false, "#EXTINF:invalid,", false, &MediaSegment{Duration: 0.0, Title: ""}},
		{false, "#EXTINF:10.000", false, &MediaSegment{Duration: 10.0, Title: ""}},
	}

	for _, test := range tests {
		p, err := NewMediaPlaylist(1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		reader := bytes.NewBufferString(fmt.Sprintf(header, test.extInf))
		err = p.DecodeFrom(reader, test.strict)
		if test.wantError {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, p.Segments[0], test.wantSegment)
	}
}

func TestDecodeMediaPlaylistWithWidevine(t *testing.T) {
	f, err := os.Open("sample-playlists/widevine-bitrate.m3u8")
	require.NoError(t, err)

	p, err := NewMediaPlaylist(5, 798)
	require.NoError(t, err)

	err = p.DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	require.Equal(t, p.ver, uint8(2))
	require.Equal(t, p.TargetDuration, 9.0)
	// TODO check other values…
	//fmt.Printf("%+v\n", p.Key)
	//fmt.Println(p.Encode().String())
}

func TestDecodeMasterPlaylistWithAutodetection(t *testing.T) {
	f, err := os.Open("sample-playlists/master.m3u8")
	require.NoError(t, err)

	m, listType, err := DecodeFrom(bufio.NewReader(f), false)
	require.NoError(t, err)
	require.Equal(t, listType, MASTER)

	mp := m.(*MasterPlaylist)
	// fmt.Printf(">%+v\n", mp)
	// for _, v := range mp.Variants {
	//	fmt.Printf(">>%+v +v\n", v)
	// }
	//fmt.Println("Type below must be MasterPlaylist:")
	CheckType(t, mp)
}

func TestDecodeMediaPlaylistWithAutodetection(t *testing.T) {
	f, err := os.Open("sample-playlists/wowza-vod-chunklist.m3u8")
	require.NoError(t, err)

	p, listType, err := DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	pp := p.(*MediaPlaylist)
	CheckType(t, pp)
	require.Equal(t, listType, MEDIA)

	// check parsed values
	require.Equal(t, pp.TargetDuration, 12.0)
	require.True(t, pp.Closed)
	require.Zero(t, pp.winsize)
	// TODO check other values…
	// fmt.Println(pp.Encode().String())
}

// TestDecodeMediaPlaylistAutoDetectExtend tests a very large playlist auto
// extends to the appropriate size.
func TestDecodeMediaPlaylistAutoDetectExtend(t *testing.T) {
	f, err := os.Open("sample-playlists/media-playlist-large.m3u8")
	require.NoError(t, err)

	p, listType, err := DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	pp := p.(*MediaPlaylist)
	CheckType(t, pp)
	require.Equal(t, listType, MEDIA)

	require.Equal(t, pp.Count(), uint(40001))
}

// Test for FullTimeParse of EXT-X-PROGRAM-DATE-TIME
// We testing ISO/IEC 8601:2004 where we can get time in UTC, UTC with Nanoseconds
// timeZone in formats '±00:00', '±0000', '±00'
// m3u8.FullTimeParse()
func TestFullTimeParse(t *testing.T) {
	var timestamps = []struct {
		name  string
		value string
	}{
		{"time_in_utc", "2006-01-02T15:04:05Z"},
		{"time_in_utc_nano", "2006-01-02T15:04:05.123456789Z"},
		{"time_with_positive_zone_and_colon", "2006-01-02T15:04:05+01:00"},
		{"time_with_positive_zone_no_colon", "2006-01-02T15:04:05+0100"},
		{"time_with_positive_zone_2digits", "2006-01-02T15:04:05+01"},
		{"time_with_negative_zone_and_colon", "2006-01-02T15:04:05-01:00"},
		{"time_with_negative_zone_no_colon", "2006-01-02T15:04:05-0100"},
		{"time_with_negative_zone_2digits", "2006-01-02T15:04:05-01"},
	}

	var err error
	for _, tstamp := range timestamps {
		_, err = FullTimeParse(tstamp.value)
		require.NoError(t, err)
	}
}

// Test for StrictTimeParse of EXT-X-PROGRAM-DATE-TIME
// We testing Strict format of RFC3339 where we can get time in UTC, UTC with Nanoseconds
// timeZone in formats '±00:00', '±0000', '±00'
// m3u8.StrictTimeParse()
func TestStrictTimeParse(t *testing.T) {
	var timestamps = []struct {
		name  string
		value string
	}{
		{"time_in_utc", "2006-01-02T15:04:05Z"},
		{"time_in_utc_nano", "2006-01-02T15:04:05.123456789Z"},
		{"time_with_positive_zone_and_colon", "2006-01-02T15:04:05+01:00"},
		{"time_with_negative_zone_and_colon", "2006-01-02T15:04:05-01:00"},
	}

	var err error
	for _, tstamp := range timestamps {
		_, err = StrictTimeParse(tstamp.value)
		require.NoError(t, err)
	}
}

func TestMediaPlaylistWithOATCLSSCTE35Tag(t *testing.T) {
	f, err := os.Open("sample-playlists/media-playlist-with-oatcls-scte35.m3u8")
	require.NoError(t, err)

	p, _, err := DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	pp := p.(*MediaPlaylist)

	expect := map[int]*SCTE{
		0: {Syntax: SCTE35_OATCLS, CueType: SCTE35Cue_Start, Cue: "/DAlAAAAAAAAAP/wFAUAAAABf+/+ANgNkv4AFJlwAAEBAQAA5xULLA==", Time: 15},
		1: {Syntax: SCTE35_OATCLS, CueType: SCTE35Cue_Mid, Cue: "/DAlAAAAAAAAAP/wFAUAAAABf+/+ANgNkv4AFJlwAAEBAQAA5xULLA==", Time: 15, Elapsed: 8.844},
		2: {Syntax: SCTE35_OATCLS, CueType: SCTE35Cue_End},
	}
	for i := range int(pp.Count()) {
		require.Equal(t, pp.Segments[i].SCTE, expect[i])
	}
}

func TestDecodeMediaPlaylistWithDiscontinuitySeq(t *testing.T) {
	f, err := os.Open("sample-playlists/media-playlist-with-discontinuity-seq.m3u8")
	require.NoError(t, err)

	p, listType, err := DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	pp := p.(*MediaPlaylist)
	CheckType(t, pp)
	require.Equal(t, listType, MEDIA)
	require.NotZero(t, pp.DiscontinuitySeq)
	require.Equal(t, pp.Count(), uint(4))
	require.Zero(t, pp.SeqNo)

	for idx := range pp.Count() {
		require.Equal(t, pp.Segments[idx].SeqId, uint64(idx))
	}
}

func TestDecodeMasterPlaylistWithCustomTags(t *testing.T) {
	t.Parallel()

	decodingTagError := errors.New("Error decoding tag")
	cases := []struct {
		src                  string
		customDecoders       []CustomDecoder
		expectedError        error
		expectedPlaylistTags []string
	}{
		{
			src:                  "sample-playlists/master-playlist-with-custom-tags.m3u8",
			customDecoders:       nil,
			expectedError:        nil,
			expectedPlaylistTags: nil,
		},
		{
			src: "sample-playlists/master-playlist-with-custom-tags.m3u8",
			customDecoders: []CustomDecoder{
				&MockCustomTag{
					name:          "#CUSTOM-PLAYLIST-TAG:",
					err:           decodingTagError,
					segment:       false,
					encodedString: "#CUSTOM-PLAYLIST-TAG:42",
				},
			},
			expectedError:        decodingTagError,
			expectedPlaylistTags: nil,
		},
		{
			src: "sample-playlists/master-playlist-with-custom-tags.m3u8",
			customDecoders: []CustomDecoder{
				&MockCustomTag{
					name:          "#CUSTOM-PLAYLIST-TAG:",
					err:           nil,
					segment:       false,
					encodedString: "#CUSTOM-PLAYLIST-TAG:42",
				},
			},
			expectedError: nil,
			expectedPlaylistTags: []string{
				"#CUSTOM-PLAYLIST-TAG:",
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.src, func(t *testing.T) {
			f, err := os.Open(testCase.src)
			require.NoError(t, err)

			p, listType, err := DecodeWith(bufio.NewReader(f), true, testCase.customDecoders)
			require.ErrorIs(t, err, testCase.expectedError, "expected error %v, got %v", testCase.expectedError, err)

			if testCase.expectedError != nil {
				return
			}

			pp := p.(*MasterPlaylist)

			CheckType(t, pp)

			require.Equal(t, listType, MASTER, "expected list type %v, got %v", MASTER, listType)
			require.Equal(t, len(pp.Custom), len(testCase.expectedPlaylistTags), "expected %d custom tags, got %d", len(testCase.expectedPlaylistTags), len(pp.Custom))
			// we have the same count, lets confirm its the right tags
			for _, expectedTag := range testCase.expectedPlaylistTags {
				require.Contains(t, pp.Custom, expectedTag)
			}
		})
	}
}

func TestDecodeMediaPlaylistWithCustomTags(t *testing.T) {
	decodingTagError := errors.New("Error decoding tag")
	cases := []struct {
		src                  string
		customDecoders       []CustomDecoder
		expectedError        error
		expectedPlaylistTags []string
		expectedSegmentTags  []*struct {
			index int
			names []string
		}
	}{
		{
			src:                  "sample-playlists/media-playlist-with-custom-tags.m3u8",
			customDecoders:       nil,
			expectedError:        nil,
			expectedPlaylistTags: nil,
			expectedSegmentTags:  nil,
		},
		{
			src: "sample-playlists/media-playlist-with-custom-tags.m3u8",
			customDecoders: []CustomDecoder{
				&MockCustomTag{
					name:          "#CUSTOM-PLAYLIST-TAG:",
					err:           decodingTagError,
					segment:       false,
					encodedString: "#CUSTOM-PLAYLIST-TAG:42",
				},
			},
			expectedError:        decodingTagError,
			expectedPlaylistTags: nil,
			expectedSegmentTags:  nil,
		},
		{
			src: "sample-playlists/media-playlist-with-custom-tags.m3u8",
			customDecoders: []CustomDecoder{
				&MockCustomTag{
					name:          "#CUSTOM-PLAYLIST-TAG:",
					err:           nil,
					segment:       false,
					encodedString: "#CUSTOM-PLAYLIST-TAG:42",
				},
				&MockCustomTag{
					name:          "#CUSTOM-SEGMENT-TAG:",
					err:           nil,
					segment:       true,
					encodedString: "#CUSTOM-SEGMENT-TAG:NAME=\"Yoda\",JEDI=YES",
				},
				&MockCustomTag{
					name:          "#CUSTOM-SEGMENT-TAG-B",
					err:           nil,
					segment:       true,
					encodedString: "#CUSTOM-SEGMENT-TAG-B",
				},
			},
			expectedError: nil,
			expectedPlaylistTags: []string{
				"#CUSTOM-PLAYLIST-TAG:",
			},
			expectedSegmentTags: []*struct {
				index int
				names []string
			}{
				{1, []string{"#CUSTOM-SEGMENT-TAG:"}},
				{2, []string{"#CUSTOM-SEGMENT-TAG:", "#CUSTOM-SEGMENT-TAG-B"}},
			},
		},
	}

	for _, testCase := range cases {
		f, err := os.Open(testCase.src)
		require.NoError(t, err)

		p, listType, err := DecodeWith(bufio.NewReader(f), true, testCase.customDecoders)
		require.ErrorIs(t, err, testCase.expectedError)

		if testCase.expectedError != nil {
			// No need to make other assertions if we were expecting an error
			continue
		}

		pp := p.(*MediaPlaylist)

		CheckType(t, pp)

		require.Equal(t, listType, MEDIA)
		require.Equal(t, len(pp.Custom), len(testCase.expectedPlaylistTags))

		// we have the same count, lets confirm its the right tags
		for _, expectedTag := range testCase.expectedPlaylistTags {
			require.Contains(t, pp.Custom, expectedTag)
		}

		var expectedSegmentTag *struct {
			index int
			names []string
		}

		expectedIndex := 0

		for i := range int(pp.Count()) {
			seg := pp.Segments[i]
			if expectedIndex != len(testCase.expectedSegmentTags) {
				expectedSegmentTag = testCase.expectedSegmentTags[expectedIndex]
			} else {
				// we are at the end of the expectedSegmentTags list, the rest of the segments
				// should have no custom tags
				expectedSegmentTag = nil
			}

			if expectedSegmentTag == nil || expectedSegmentTag.index != i {
				require.Zero(t, len(seg.Custom))
				continue
			}

			// We are now checking the segment corresponding to exepectedSegmentTag
			// increase our expectedIndex for next iteration
			expectedIndex++

			require.Equal(t, len(expectedSegmentTag.names), len(seg.Custom))
			// we have the same count, lets confirm its the right tags
			for _, expectedTag := range expectedSegmentTag.names {
				require.Contains(t, seg.Custom, expectedTag)
			}
		}

		require.Equal(t, expectedIndex, len(testCase.expectedSegmentTags))
	}
}

/***************************
 *  Code parsing examples  *
 ***************************/

// Example of parsing a playlist with EXT-X-DISCONTINIUTY tag
// and output it with integer segment durations.
func ExampleMediaPlaylist_DurationAsInt() {
	f, _ := os.Open("sample-playlists/media-playlist-with-discontinuity.m3u8")
	p, _, _ := DecodeFrom(bufio.NewReader(f), true)
	pp := p.(*MediaPlaylist)
	pp.DurationAsInt(true)
	fmt.Printf("%s", pp)
	// Output:
	// #EXTM3U
	// #EXT-X-VERSION:3
	// #EXT-X-MEDIA-SEQUENCE:0
	// #EXT-X-TARGETDURATION:10
	// #EXTINF:10,
	// ad0.ts
	// #EXTINF:8,
	// ad1.ts
	// #EXT-X-DISCONTINUITY
	// #EXTINF:10,
	// movieA.ts
	// #EXTINF:10,
	// movieB.ts
}

func TestMediaPlaylistWithDiscontinuityTag(t *testing.T) {
	f, _ := os.Open("sample-playlists/media-playlist-with-discontinuity.m3u8")
	p, _, err := DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	pp := p.(*MediaPlaylist)
	pp.DurationAsInt(true)
	fmt.Printf("%s", pp)
}

func TestMediaPlaylistWithSCTE35Tag(t *testing.T) {
	cases := []struct {
		playlistLocation  string
		expectedSCTEIndex int
		expectedSCTECue   string
		expectedSCTEID    string
		expectedSCTETime  float64
	}{
		{
			"sample-playlists/media-playlist-with-scte35.m3u8",
			2,
			"/DAIAAAAAAAAAAAQAAZ/I0VniQAQAgBDVUVJQAAAAH+cAAAAAA==",
			"123",
			123.12,
		},
		{
			"sample-playlists/media-playlist-with-scte35-1.m3u8",
			1,
			"/DAIAAAAAAAAAAAQAAZ/I0VniQAQAgBDVUVJQAA",
			"",
			0,
		},
	}
	for _, c := range cases {
		f, err := os.Open(c.playlistLocation)
		require.NoError(t, err)

		playlist, _, err := DecodeFrom(bufio.NewReader(f), true)
		require.NoError(t, err)

		mediaPlaylist := playlist.(*MediaPlaylist)
		for index, item := range mediaPlaylist.Segments {
			if item == nil {
				break
			}
			require.False(t, index != c.expectedSCTEIndex && item.SCTE != nil,
				"Not expecting SCTE information on this segment")
			require.False(t, index == c.expectedSCTEIndex && item.SCTE == nil,
				"Expecting SCTE information on this segment")
			if index == c.expectedSCTEIndex && item.SCTE != nil {
				require.Equal(t, (*item.SCTE).Cue, c.expectedSCTECue)
				require.Equal(t, (*item.SCTE).ID, c.expectedSCTEID)
				require.Equal(t, (*item.SCTE).Time, c.expectedSCTETime)
			}
		}
	}
}

func TestDecodeMediaPlaylistWithProgramDateTime(t *testing.T) {
	f, err := os.Open("sample-playlists/media-playlist-with-program-date-time.m3u8")
	require.NoError(t, err)

	p, listType, err := DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	pp := p.(*MediaPlaylist)
	CheckType(t, pp)
	require.Equal(t, listType, MEDIA)

	// check parsed values
	require.Equal(t, pp.TargetDuration, 15.0)
	require.True(t, pp.Closed)
	require.Zero(t, pp.SeqNo)

	segNames := []string{"20181231/0555e0c371ea801726b92512c331399d_00000000.ts",
		"20181231/0555e0c371ea801726b92512c331399d_00000001.ts",
		"20181231/0555e0c371ea801726b92512c331399d_00000002.ts",
		"20181231/0555e0c371ea801726b92512c331399d_00000003.ts"}
	if pp.Count() != uint(len(segNames)) {
		t.Errorf("Segments in playlist %d != %d", pp.Count(), len(segNames))
	}

	for idx, name := range segNames {
		require.Equal(t, pp.Segments[idx].URI, name)
	}

	// The ProgramDateTime of the 1st segment should be: 2018-12-31T09:47:22+08:00
	st, err := time.Parse(time.RFC3339, "2018-12-31T09:47:22+08:00")
	require.NoError(t, err)
	require.True(t, pp.Segments[0].ProgramDateTime.Equal(st))
}

func TestDecodeMediaPlaylistStartTime(t *testing.T) {
	f, err := os.Open("sample-playlists/media-playlist-with-start-time.m3u8")
	require.NoError(t, err)

	p, listType, err := DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	pp := p.(*MediaPlaylist)
	CheckType(t, pp)
	require.Equal(t, listType, MEDIA)
	require.Equal(t, pp.StartTime, float64(8.0))
}

func TestDecodeMediaPlaylistWithCueOutCueIn(t *testing.T) {
	f, err := os.Open("sample-playlists/media-playlist-with-cue-out-in-without-oatcls.m3u8")
	require.NoError(t, err)

	p, listType, err := DecodeFrom(bufio.NewReader(f), true)
	require.NoError(t, err)

	pp := p.(*MediaPlaylist)
	CheckType(t, pp)
	require.Equal(t, listType, MEDIA)

	require.Equal(t, pp.Segments[5].SCTE.CueType, SCTE35Cue_Start)
	require.Zero(t, pp.Segments[5].SCTE.Time)
	require.Equal(t, pp.Segments[9].SCTE.CueType, SCTE35Cue_End)
	require.Equal(t, pp.Segments[30].SCTE.CueType, SCTE35Cue_Start)
	require.Equal(t, pp.Segments[30].SCTE.Time, float64(180))
	require.Equal(t, pp.Segments[60].SCTE.CueType, SCTE35Cue_End)
}

/****************
 *  Benchmarks  *
 ****************/

func BenchmarkDecodeMasterPlaylist(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f, err := os.Open("sample-playlists/master.m3u8")
		require.NoError(b, err)

		p := NewMasterPlaylist()
		err = p.DecodeFrom(bufio.NewReader(f), false)
		require.NoError(b, err)
	}
}

func BenchmarkDecodeMediaPlaylist(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f, err := os.Open("sample-playlists/media-playlist-large.m3u8")
		require.NoError(b, err)

		p, err := NewMediaPlaylist(50000, 50000)
		require.NoError(b, err)

		err = p.DecodeFrom(bufio.NewReader(f), true)
		require.NoError(b, err)
	}
}
