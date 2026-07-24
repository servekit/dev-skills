module example.com/myorg/userservice

go 1.22

require (
	github.com/google/uuid v1.6.0
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.25.12
)

// gorm.io/cli is a code-generation tool installed separately via:
//   go install gorm.io/cli/gorm@latest
// It is NOT listed as a dependency here — running `gorm gen` only needs
// the binary on PATH, not a library import.
