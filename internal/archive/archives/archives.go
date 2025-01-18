package archives

import (
	"github.com/alist-org/alist/v3/internal/archive/tool"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/alist-org/alist/v3/pkg/utils"
	"io"
	"io/fs"
	"os"
	stdpath "path"
	"strings"
)

type Archives struct {
}

func (_ *Archives) AcceptedExtensions() []string {
	return []string{
		".br", ".bz2", ".gz", ".lz4", ".lz", ".sz", ".s2", ".xz", ".zz", ".zst", ".tar", ".rar", ".7z",
	}
}

func (_ *Archives) GetMeta(ss *stream.SeekableStream, args model.ArchiveArgs) (model.ArchiveMeta, error) {
	fsys, err := getFs(ss, args)
	if err != nil {
		return nil, err
	}
	_, err = fsys.ReadDir(".")
	if err != nil {
		return nil, filterPassword(err)
	}
	return &model.ArchiveMetaInfo{
		Comment:   "",
		Encrypted: false,
	}, nil
}

func (_ *Archives) List(ss *stream.SeekableStream, args model.ArchiveInnerArgs) ([]model.Obj, error) {
	fsys, err := getFs(ss, args.ArchiveArgs)
	if err != nil {
		return nil, err
	}
	innerPath := strings.TrimPrefix(args.InnerPath, "/")
	if innerPath == "" {
		innerPath = "."
	}
	obj, err := fsys.ReadDir(innerPath)
	if err != nil {
		return nil, filterPassword(err)
	}
	return utils.SliceConvert(obj, func(src os.DirEntry) (model.Obj, error) {
		info, err := src.Info()
		if err != nil {
			return nil, err
		}
		return toModelObj(info), nil
	})
}

func (_ *Archives) Extract(ss *stream.SeekableStream, args model.ArchiveInnerArgs) (io.ReadCloser, int64, error) {
	fsys, err := getFs(ss, args.ArchiveArgs)
	if err != nil {
		return nil, 0, err
	}
	file, err := fsys.Open(strings.TrimPrefix(args.InnerPath, "/"))
	if err != nil {
		return nil, 0, filterPassword(err)
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, 0, filterPassword(err)
	}
	return file, stat.Size(), nil
}

func (_ *Archives) Decompress(ss *stream.SeekableStream, outputPath string, args model.ArchiveInnerArgs, up model.UpdateProgress) error {
	fsys, err := getFs(ss, args.ArchiveArgs)
	if err != nil {
		return err
	}
	isDir := false
	path := strings.TrimPrefix(args.InnerPath, "/")
	if path == "" {
		isDir = true
		path = "."
	} else {
		stat, err := fsys.Stat(path)
		if err != nil {
			return filterPassword(err)
		}
		if stat.IsDir() {
			isDir = true
			outputPath = stdpath.Join(outputPath, stat.Name())
			err = os.Mkdir(outputPath, 0700)
			if err != nil {
				return filterPassword(err)
			}
		}
	}
	if isDir {
		err = fs.WalkDir(fsys, path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			relPath := strings.TrimPrefix(p, path+"/")
			dstPath := stdpath.Join(outputPath, relPath)
			if d.IsDir() {
				err = os.MkdirAll(dstPath, 0700)
			} else {
				dir := stdpath.Dir(dstPath)
				err = decompress(fsys, p, dir, func(_ float64) {})
			}
			return err
		})
	} else {
		err = decompress(fsys, path, outputPath, up)
	}
	return filterPassword(err)
}

var _ tool.Tool = (*Archives)(nil)

func init() {
	tool.RegisterTool(&Archives{})
}