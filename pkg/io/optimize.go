package io

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/g026r/pocket-library-toolkit/pkg/models"
	"github.com/g026r/pocket-library-toolkit/pkg/root"
)

const backupDir = "bak"
const modeDir = 0o777 // Because using os.ModeDir causes errors with os.Root.Mkdir

// Optimize moves images for games that do not exist in the library from the images folders & places them in a backup
// directory, as well as moving any images from the backup directory into the images folders.
// This can improve cart recognition times slightly on slower SD cards, as FAT filesystems are not efficient for large
// directories like gb or gba, where a full library set is roughly 3,000 images each.
// r is the root directory for images (i.e. /System/Library/Images)
// ctr is a pointer to the counter to update for the progress bar
// entries is the game entries in the library
func Optimize(r *root.Root, ctr *float64, entries []models.Entry) error {
	// TODO: Use the ctr properly
	fileSys := r.FS()
	// fileSys, err := fs.Sub(r.FS(), "System/Library/Images")
	// if err != nil {
	// 	return fmt.Errorf("error opening images dir: %w", err)
	// }

	// Remove any entries that are no longer in the library from the dir
	for _, sys := range models.ValidThumbsFiles {
		if err := initialize(r, sys); err != nil {
			return fmt.Errorf("initialization error: %w", err)
		}

		sub, err := fs.ReadDir(fileSys, strings.ToLower(sys.String()))
		if err != nil {
			return err
		}
		slices.SortFunc(entries, func(a, b models.Entry) int {
			return cmp.Compare(a.Crc32, b.Crc32)
		})

		for _, file := range sub {
			if file.IsDir() {
				continue // Skip directories
			}
			_, found := slices.BinarySearchFunc(entries, file.Name(), func(entry models.Entry, s string) int {
				return cmp.Compare(fmt.Sprintf("%08x.bin", entry.Crc32), strings.ToLower(s))
			})
			if !found {
				if err := copyToBackups(r, sys, file.Name()); err != nil {
					return fmt.Errorf("error copying to backups: %w", err)
				}
			}
		}
	}

	// Copy any missing entries into the dir
	for _, entry := range entries {
		sys := entry.System.ThumbFile()
		sysStr := strings.ToLower(sys.String())
		bin := fmt.Sprintf("%08x.bin", entry.Crc32)

		imgDir, err := fs.Sub(fileSys, sysStr)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		_, err = fs.Stat(imgDir, bin)
		if os.IsNotExist(err) {
			if err := copyToImages(r, sys, bin); err != nil {
				return fmt.Errorf("error copying to images: %w", err)
			}
		}
	}

	return nil
}

func copyToBackups(r *root.Root, sys models.System, binFile string) error {
	fileSys := r.FS()
	sysDir := strings.ToLower(sys.String())
	backupDir := filepath.Join(sysDir, backupDir)

	if fi, err := fs.Stat(fileSys, strings.ToLower(sys.String())+backupDir); os.IsNotExist(err) {
		// TODO: Something with this error
		r.Mkdir(backupDir, modeDir)
	} else if err != nil {
		return err
	} else if !fi.IsDir() {
		return errors.New("not a dir")
	}

	// TODO: Check to confirm that the file we want to copy actually exists here
	return r.Rename(filepath.Join(sysDir, binFile), filepath.Join(backupDir, binFile))
}

func copyToImages(r *root.Root, sys models.System, binFile string) error {
	fileSys := r.FS()
	sysDir := strings.ToLower(sys.String())
	backupDir := filepath.Join(sysDir, backupDir)

	if fi, err := fs.Stat(fileSys, strings.ToLower(sys.String())); os.IsNotExist(err) {
		// TODO: Something with this error
		r.Mkdir(sysDir, modeDir)
	} else if err != nil {
		return err
	} else if !fi.IsDir() {
		return errors.New("not a dir")
	}

	// TODO: Check to make certain the file we're copying actually exists in the full library set
	// TODO: Or will that just error out here?
	return r.Rename(filepath.Join(backupDir, binFile), filepath.Join(sysDir, binFile))
}

func initialize(r *root.Root, sys models.System) error {
	fi, err := fs.Stat(r.FS(), strings.ToLower(sys.String()))
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking image dir: %w", err)
	}

	if !fi.IsDir() {
		log.Printf("%s not a dir", fi.Name())
		return errors.New("not a dir")
	}

	if fi2, err := fs.Stat(r.FS(), filepath.Join(strings.ToLower(sys.String()), backupDir)); err == nil {
		log.Printf("%s already exists", fi2.Name())
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking backup dir: %w", err)
	}

	// if err := r.Rename(strings.ToLower(sys.String()), filepath.Join(strings.ToLower(sys.String()), backupDir)); err != nil {
	// 	return fmt.Errorf("rename error: %w", err)
	// }

	if err := r.Mkdir(strings.ToLower(filepath.Join(strings.ToLower(sys.String()), backupDir)), modeDir); err != nil {
		return fmt.Errorf("mkdir error: %w", err)
	}

	return nil
}
