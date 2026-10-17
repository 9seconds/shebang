package world

// World holds the process-wide filesystem, identity, and Git caches.
var World = struct {
	Users Users
	Files Files
	Git   Git
}{}
