module go_memcache_loader

go 1.24.5

replace (
	github.com/speculzzz/go_memcache_loader => .
	go_memcache_loader/tools => /dev/null
)

require github.com/bradfitz/gomemcache v0.0.0-20250403215159-8d39553ac7cf

require google.golang.org/protobuf v1.36.6
