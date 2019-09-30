# Pandoras Box

`pandorasbox` is a go package that allows for simple use of the host's filesystem, and a virtual filesystem.

The design goal of Pandora's Box is to easily facilitate the use of a transparently-encrypted VFS (virtual filesystem), and the host's filesystem.
Switching between using the VFS or host fs is as easy as prepending `vfs://` to your paths. If a path starts with `vfs://`, the VFS will be used. Otherwise, the host's filesystem will be used.

All files in the VFS are encrypted when not in use. When files from the VFS are opened, they are opened into a protected buffer that is very difficult for other processes to read. VFS files are then re-encrypted when reading or writing from them is finished. For more information about the exact cryptographic code and algorithms used, refer to this repo: https://github.com/awnumar/memguard.

Pandora's Box implements all the file-based functions from `os` and `io/ioutil`, and the returned files have all the same methods as `os.File`. Using Pandora's Box is as simple as just creating a new `Box`, and then using it like you would `os` and/or `io/ioutil`.

Example: (error handling omitted)

```go
box, _ := pandorasbox.NewBox()
defer box.Close()

box.WriteFile("vfs://file.txt", []byte("Testing testing 1 2 3"), 0644)
data, _ := box.ReadFile("vfs://file.txt")
fmt.Println(string(data))
```

### Acknowledgements

Thanks to AbsFs contributors for the amazing repos, 90% of the code is from repos from [this organization](https://github.com/absfs).

Thanks to [awnumar](https://github.com/awnumar) for [memguard](https://github.com/awnumar/memguard), he created a great repo that is very easy to use safely.