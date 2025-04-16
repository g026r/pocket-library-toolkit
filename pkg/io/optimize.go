package io

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/g026r/pocket-library-toolkit/pkg/models"
	"github.com/g026r/pocket-library-toolkit/pkg/root"
)

const backupSuffix = "_bak"

func optimize(r root.Root, entries []models.Entry) error {
	fileSys := r.FS()

	// Remove any entries that are no longer in the library from the dir
	for _, sys := range models.ValidThumbsFiles {
		initialize(r, sys)

		sub, err := fs.ReadDir(fileSys, strings.ToLower(sys.String()))
		if err != nil {
			return err
		}
		slices.SortFunc(entries, func(a, b models.Entry) int {
			return cmp.Compare(a.Crc32, b.Crc32)
		})

		for _, file := range sub {
			_, found := slices.BinarySearchFunc(entries, file.Name(), func(entry models.Entry, s string) int {
				return cmp.Compare(fmt.Sprintf("%08x.bin", entry.Crc32), strings.ToLower(s))
			})
			if !found {
				copyToBackups(r, sys, file.Name())
			}
		}
	}

	// Copy any missing entries into the dir
	for _, entry := range entries {
		sys := strings.ToLower(entry.System.String())
		bin := fmt.Sprintf("%08x.bin", entry.Crc32)

		imgDir, err := fs.Sub(fileSys, sys)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		_, err = fs.Stat(imgDir, bin)
		if os.IsNotExist(err) {
			copyToImages(r, entry.System, bin)
		}
	}

	return nil
}

func copyToBackups(r root.Root, sys models.System, binFile string) error {
	fileSys := r.FS()
	sysDir := strings.ToLower(sys.String())
	backupDir := fmt.Sprintf("%s%s", sysDir, backupSuffix)

	if fi, err := fs.Stat(fileSys, strings.ToLower(sys.String())+backupDir); os.IsNotExist(err) {
		// TODO: Don't think I need to do this? Since I did it during the initialization
		r.Mkdir(backupDir, fs.ModeDir)
	} else if err != nil {
		return err
	} else if !fi.IsDir() {
		return errors.New("not a dir")
	}

	// TODO: Check to confirm that the file we want to copy actually exists here
	return r.Rename(fmt.Sprintf("%s/%s", sysDir, binFile), fmt.Sprintf("%s/%s", backupDir, binFile))
}

func copyToImages(r root.Root, sys models.System, binFile string) error {
	fileSys := r.FS()
	sysDir := strings.ToLower(sys.String())
	backupDir := fmt.Sprintf("%s%s", sysDir, backupSuffix)

	if fi, err := fs.Stat(fileSys, strings.ToLower(sys.String())); os.IsNotExist(err) {
		// TODO: Don't think I need to do this? Since I did it during the initialization
		r.Mkdir(sysDir, fs.ModeDir)
	} else if err != nil {
		return err
	} else if !fi.IsDir() {
		return errors.New("not a dir")
	}

	// TODO: Check to make certain the file we're copying actually exists in the full library set
	// TODO: Or will that just error out here?
	return r.Rename(fmt.Sprintf("%s/%s", backupDir, binFile), fmt.Sprintf("%s/%s", sysDir, binFile))
}

func initialize(r root.Root, sys models.System) error {
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

	if fi2, err := fs.Stat(r.FS(), strings.ToLower(sys.String())+backupSuffix); err == nil {
		log.Printf("%s already exists", fi2.Name())
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking backup dir: %w", err)
	}

	if err := os.Rename(strings.ToLower(sys.String()), strings.ToLower(sys.String())+backupSuffix); err != nil {
		return fmt.Errorf("rename error: %w", err)
	}

	if err := r.Mkdir(strings.ToLower(sys.String()), fs.ModeDir); err != nil {
		return fmt.Errorf("mkdir error: %w", err)
	}

	return nil
}
