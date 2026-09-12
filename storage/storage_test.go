package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCategoryByName(t *testing.T) {
	t.Run("known category", func(t *testing.T) {
		category, err := FindCategoryByName("songs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if category == nil || *category != Songs {
			t.Errorf("category = %+v, want %+v", category, Songs)
		}
	})

	t.Run("unknown category", func(t *testing.T) {
		category, err := FindCategoryByName("nao-existe")
		if err == nil {
			t.Fatal("expected an error for an unknown category")
		}
		if category != nil {
			t.Errorf("expected nil category, got %+v", *category)
		}
	})
}

func TestSetBasePath(t *testing.T) {
	t.Cleanup(func() { SetBasePath("") })

	t.Run("custom path gets a trailing slash", func(t *testing.T) {
		SetBasePath("/var/dados/media")
		if got := BasePath(); got != "/var/dados/media/" {
			t.Errorf("BasePath() = %q, want %q", got, "/var/dados/media/")
		}
	})

	t.Run("custom path with trailing slash is kept as-is", func(t *testing.T) {
		SetBasePath("/var/dados/media/")
		if got := BasePath(); got != "/var/dados/media/" {
			t.Errorf("BasePath() = %q, want %q", got, "/var/dados/media/")
		}
	})

	t.Run("empty path resets to the default", func(t *testing.T) {
		SetBasePath("/var/dados/media")
		SetBasePath("")
		if got := BasePath(); got != defaultBasePath {
			t.Errorf("BasePath() = %q, want %q", got, defaultBasePath)
		}
	})
}

func TestListRespectsCustomBasePath(t *testing.T) {
	dir := withTempCwd(t)
	t.Cleanup(func() { SetBasePath("") })

	SetBasePath(filepath.Join(dir, "dados-do-culto"))

	songsDir := filepath.Join(dir, "dados-do-culto", "songs")
	if err := os.MkdirAll(songsDir, 0755); err != nil {
		t.Fatalf("failed to create songs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(songsDir, "musica.txt"), []byte("letra"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	names, err := List(Songs.Name)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "musica.txt" {
		t.Errorf("List() = %v, want [\"musica.txt\"]", names)
	}
}

func withTempCwd(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into temp dir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(cwd) })
	return dir
}

func TestSaveTextFileAndList(t *testing.T) {
	dir := withTempCwd(t)
	if err := os.MkdirAll(filepath.Join(dir, "media", "songs"), 0755); err != nil {
		t.Fatalf("failed to create songs dir: %v", err)
	}

	SaveTextFile(Songs, "Minha Musica - Autor", "Linha1\nLinha2")

	names, err := List(Songs.Name)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 1 || names[0] != "Minha Musica - Autor.txt" {
		t.Errorf("List() = %v, want [\"Minha Musica - Autor.txt\"]", names)
	}
}

func TestListFromFolderAndListFolders(t *testing.T) {
	dir := withTempCwd(t)

	folderPath := filepath.Join(dir, "media", "songs", "musicas", "louvores")
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		t.Fatalf("failed to create test folder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(folderPath, "musica.txt"), []byte("letra"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	folders, err := ListFolders(Songs.Name, "musicas")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(folders) != 1 || folders[0] != "louvores" {
		t.Errorf("ListFolders() = %v, want [\"louvores\"]", folders)
	}

	files, err := ListFromFolder(Songs.Name, "musicas", "louvores")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "musica.txt" {
		t.Errorf("ListFromFolder() = %v, want [\"musica.txt\"]", files)
	}
}

func TestLoadSongFile(t *testing.T) {
	dir := withTempCwd(t)

	songsDir := filepath.Join(dir, "media", "songs")
	if err := os.MkdirAll(songsDir, 0755); err != nil {
		t.Fatalf("failed to create songs dir: %v", err)
	}
	const body = "conteúdo da música"
	if err := os.WriteFile(filepath.Join(songsDir, "cancao.txt"), []byte(body), 0644); err != nil {
		t.Fatalf("failed to write song file: %v", err)
	}

	got := string(LoadSongFile("cancao.txt"))
	if got != body {
		t.Errorf("LoadSongFile() = %q, want %q", got, body)
	}
}
