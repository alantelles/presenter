module presenter/images

go 1.23.4

require (
	golang.org/x/image v0.32.0
	presenter/fsutil v0.0.0-00010101000000-000000000000
	presenter/storage v0.0.0-00010101000000-000000000000
)

replace presenter/fsutil => ../fsutil

replace presenter/storage => ../storage
