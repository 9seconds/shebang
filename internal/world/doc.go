// Package world caches filesystem, identity, and Git queries for a process.
// Cache instances must not be copied after first use. Results, including errors,
// are snapshots; callers must treat returned slices as read-only. Git instances
// and relative filesystem paths assume the working directory remains unchanged.
package world
