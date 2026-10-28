package main

import "testing"

func TestFindArtistPathInIndex(t *testing.T) {
	content := `
		<a href="/cazuza/">Cazuza</a>
		<a href="/caetano-veloso/">Caetano Veloso</a>
	`

	tests := []struct {
		name   string
		artist string
		want   string
	}{
		{"exact match", "cazuza", "cazuza"},
		{"case insensitive", "CAZUZA", "cazuza"},
		{"another artist in the same index", "caetano veloso", "caetano-veloso"},
		{"not found", "legiao urbana", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findArtistPathInIndex(tt.artist, content)
			if got != tt.want {
				t.Errorf("findArtistPathInIndex(%q) = %q, want %q", tt.artist, got, tt.want)
			}
		})
	}
}

func TestFindSongLyricsId(t *testing.T) {
	content := `
		<a href="/cazuza/exagerado/" title="Exagerado"><span>Exagerado</span></a>
		<a href="/cazuza/pro-dia-nascer-feliz/"><span>Pro Dia Nascer Feliz</span></a>
		<a href="/caetano-veloso/exagerado/"><span>Exagerado</span></a>
	`

	tests := []struct {
		name       string
		artistPath string
		songName   string
		want       string
	}{
		{"match with title attribute", "cazuza", "exagerado", "exagerado"},
		{"match without title attribute", "cazuza", "pro dia nascer feliz", "pro-dia-nascer-feliz"},
		{"same song name, different artist", "caetano-veloso", "exagerado", "exagerado"},
		{"song not found for artist", "cazuza", "sonho meu", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSongLyricsId(tt.artistPath, content, tt.songName)
			if got != tt.want {
				t.Errorf("findSongLyricsId(%q, _, %q) = %q, want %q", tt.artistPath, tt.songName, got, tt.want)
			}
		})
	}
}

func TestGetSongNameAndArtistName(t *testing.T) {
	content := `{"track_name":"Exagerado","artist_name":"Cazuza"}`

	song, artist := getSongNameAndArtistName(content)
	if song != "Exagerado" {
		t.Errorf("song name = %q, want %q", song, "Exagerado")
	}
	if artist != "Cazuza" {
		t.Errorf("artist name = %q, want %q", artist, "Cazuza")
	}
}
