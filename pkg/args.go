package pkg

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var ErrInvalidArguments = errors.New("invalid arguments")

// Args contains all the recognized commandline arguments
type Args struct {
	// Datafile is the dat-o-matic datafile for multi-game processing mode. Mutually exclusive with CRC.
	Datafile string
	// CRC is the crc to use when generating an image in single file mode. Mutually exclusive with Datafile.
	CRC string
	// ImagePath is a file if CRC is specified & a directory if Datafile is.
	ImagePath string
	// OutPath will be created during processing. Will default to the current working dir otherwise.
	OutPath string
	// Upscale specifies whether to resize images less than MaxImgSize pixels.
	Upscale bool
	// Verbose prints a few extra logging messages.
	Verbose bool
	// Duo specifies whether to output in Duo-format. Doesn't currently work
	// TODO: What is different between Duo & Pocket images?
	Duo bool
}

// ParseArgs parses the commandline args as well as verifies that the correct set of them is provided.
// flag is admittedly a bit crap, as you technically can specify the same arg on the command line more than once &
// there's nothing we can do to detect that
func ParseArgs() (Args, error) {
	args := Args{}
	flag.Usage = printUsage
	flag.BoolVar(&args.Upscale, "upscale", false, "Resizes images less than 170 pixels high")
	flag.BoolVar(&args.Upscale, "u", false, "Resizes images less than 170 pixels high")

	flag.BoolVar(&args.Verbose, "verbose", false, "Turns on verbose logging")
	flag.BoolVar(&args.Verbose, "v", false, "Turns on verbose logging")

	flag.StringVar(&args.CRC, "crc", "", "CRC value for single image processing")
	flag.StringVar(&args.CRC, "c", "", "CRC value for single image processing")

	flag.StringVar(&args.Datafile, "datafile", "", "no-intro dat-o-matic datafile")
	flag.StringVar(&args.Datafile, "d", "", "no-intro dat-o-matic datafile")

	flag.StringVar(&args.ImagePath, "in", "", "Image input src")
	flag.StringVar(&args.ImagePath, "i", "", "Image input src")

	// Use the current working directory as the default output dir if not specified
	// Probably should warn against this if you're using a datafile though
	out, err := os.Getwd()
	if err != nil {
		return Args{}, fmt.Errorf("os.Getwd: %w", err)
	}
	flag.StringVar(&args.OutPath, "out", out, "Output directory")
	flag.StringVar(&args.OutPath, "o", out, "Output directory")

	flag.Parse()

	args, err = verifyArgs(args)
	if errors.Is(err, ErrInvalidArguments) {
		flag.Usage()
		os.Exit(2)
	}

	return args, err
}

func verifyArgs(args Args) (Args, error) {
	if len(args.Datafile) == 0 && len(args.CRC) == 0 {
		fmt.Println("ERROR: one of --datafile or --crc must be specified")
		return args, ErrInvalidArguments
	}
	if len(args.Datafile) != 0 && len(args.CRC) != 0 {
		fmt.Println("ERROR: --datafile and --crc are mutually exclusive")
		return args, ErrInvalidArguments
	}
	if len(args.ImagePath) == 0 {
		fmt.Println("ERROR: --in must be specified")
		return args, ErrInvalidArguments
	}
	if len(args.OutPath) == 0 {
		fmt.Println("ERROR: --out must be specified")
		return args, ErrInvalidArguments
	}

	outpath, err := filepath.Abs(args.OutPath)
	if err != nil {
		return Args{}, fmt.Errorf("error finding absolute path: %w", err)
	}
	args.OutPath = outpath

	imgPath, err := filepath.Abs(args.ImagePath)
	if err != nil {
		return Args{}, fmt.Errorf("error finding absolute path: %w", err)
	}
	args.ImagePath = imgPath
	// Image source is the most complex: it can be a file or a directory depending on the mode we're in
	if imgSrc, err := os.Stat(args.ImagePath); errors.Is(err, os.ErrNotExist) {
		return Args{}, fmt.Errorf("input source %s does not exist", args.ImagePath)
	} else if err != nil {
		return Args{}, fmt.Errorf("error opening input src %s: %w", args.ImagePath, err)
	} else if len(args.Datafile) != 0 && !imgSrc.IsDir() {
		fmt.Println("ERROR: --in must be a dir if --datafile is specified")
		return args, ErrInvalidArguments
	} else if len(args.CRC) != 0 && imgSrc.IsDir() {
		fmt.Println("ERROR: --in must be a file if --crc is specified")
		return args, ErrInvalidArguments
	}

	if len(args.CRC) != 0 {
		// convert to lowercase for filename consistency & remove the `0x` prefix in case someone copied directly from a dat-o-matic file
		args.CRC = strings.TrimPrefix(strings.ToLower(args.CRC), "0x")
		if match, err := regexp.MatchString(`^[[:xdigit:]]{8}$`, args.CRC); !match {
			return Args{}, fmt.Errorf("%s is not a valid crc32 hash", args.CRC)
		} else if err != nil {
			return Args{}, fmt.Errorf("regex error: %w", err)
		}
	}

	if len(args.Datafile) > 0 {
		// Only do this after checking length as filepath.Abs("") gives you the working directory.
		datafile, err := filepath.Abs(args.Datafile)
		if err != nil {
			return Args{}, fmt.Errorf("error finding absolute path: %w", err)
		}
		args.Datafile = datafile

		if datafile, err := os.Stat(args.Datafile); errors.Is(err, os.ErrNotExist) {
			return Args{}, fmt.Errorf("datafile %s does not exist", args.Datafile)
		} else if err != nil {
			return Args{}, fmt.Errorf("error opening input src %s: %w", args.Datafile, err)
		} else if datafile.IsDir() {
			return Args{}, fmt.Errorf("datafile %s is a directory", args.Datafile)
		}
	}

	return args, nil
}

// printUsage exists to override the default flag function, which is ugly as sin
func printUsage() {
	fmt.Println(`
Usage of thumbnailizer:
-c, --crc      CRC32 of game for single image processing mode
-d, --datafile Path to no-intro dat-o-matic datafile for multi image processing mode
-h, --help     Prints this message
-i, --in       Path to image file (for crc mode) or image directory (for datafile mode)
-o, --out      Output directory (defaults to current directory if unspecified)
-u, --upscale  Resizes images less than 170 pixels high
-v, --verbose  Turns on verbose logging`)
}
