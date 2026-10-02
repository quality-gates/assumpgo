package assumpgo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
)

// packageConstLookup answers whether an identifier names a package-level
// constant in the package containing a source file.
type packageConstLookup interface {
	IsPackageConst(filePath, pkgName, identName string) bool
}

// PackageResolver indexes package-level constants by physical package
// directory and package name. Parsed analysis targets can be registered so
// the directory scan reuses their ASTs instead of parsing them again.
type PackageResolver struct {
	constants   map[string]map[string]map[string]struct{}
	scannedDirs map[string]bool
	parsedFiles map[string]*ast.File
	filesByDir  map[string]map[string]*ast.File
}

// NewPackageResolver returns a resolver with empty package symbol caches.
func NewPackageResolver() *PackageResolver {
	return &PackageResolver{
		constants:   make(map[string]map[string]map[string]struct{}),
		scannedDirs: make(map[string]bool),
		parsedFiles: make(map[string]*ast.File),
		filesByDir:  make(map[string]map[string]*ast.File),
	}
}

// IsPackageConst reports whether identName is a package-level constant in the
// package pkgName containing filePath. Sibling files are limited to files
// included by the current Go build context.
func (r *PackageResolver) IsPackageConst(filePath, pkgName, identName string) bool {
	if r == nil {
		return false
	}

	dir := packageDirectory(filePath)
	byPackage := r.constantsForDirectory(dir)
	_, found := byPackage[pkgName][identName]
	return found
}

// addParsedFile registers a parsed source file for reuse during directory
// indexing. Files excluded by Go's current build context cannot contribute
// package declarations.
func (r *PackageResolver) addParsedFile(filePath string, file *ast.File) {
	if r == nil || file == nil {
		return
	}

	canonicalPath := physicalPath(filePath)
	if !contributesConstants(canonicalPath) {
		return
	}

	if r.parsedFiles == nil {
		r.parsedFiles = make(map[string]*ast.File)
	}
	r.parsedFiles[canonicalPath] = file

	dir := filepath.Dir(canonicalPath)
	if r.filesByDir == nil {
		r.filesByDir = make(map[string]map[string]*ast.File)
	}
	if r.filesByDir[dir] == nil {
		r.filesByDir[dir] = make(map[string]*ast.File)
	}
	r.filesByDir[dir][canonicalPath] = file
	if r.scannedDirs[dir] {
		addFileConstants(r.constants[dir], file)
	}
}

func (r *PackageResolver) constantsForDirectory(dir string) map[string]map[string]struct{} {
	if r.scannedDirs[dir] {
		return r.constants[dir]
	}

	byPackage := make(map[string]map[string]struct{})
	r.addRegisteredFiles(dir, byPackage)
	r.addDirectoryFiles(dir, byPackage)

	if r.constants == nil {
		r.constants = make(map[string]map[string]map[string]struct{})
	}
	if r.scannedDirs == nil {
		r.scannedDirs = make(map[string]bool)
	}
	r.constants[dir] = byPackage
	r.scannedDirs[dir] = true
	return byPackage
}

func (r *PackageResolver) addRegisteredFiles(dir string, byPackage map[string]map[string]struct{}) {
	for _, file := range r.filesByDir[dir] {
		addFileConstants(byPackage, file)
	}
}

func (r *PackageResolver) addDirectoryFiles(dir string, byPackage map[string]map[string]struct{}) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	fset := token.NewFileSet()
	for _, entry := range entries {
		file := r.fileForEntry(dir, entry, fset)
		if file != nil {
			addFileConstants(byPackage, file)
		}
	}
}

func (r *PackageResolver) fileForEntry(dir string, entry os.DirEntry, fset *token.FileSet) *ast.File {
	path := filepath.Join(dir, entry.Name())
	if !contributesConstants(path) {
		return nil
	}

	file := r.parsedFiles[physicalPath(path)]
	if file != nil {
		return file
	}

	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	return file
}

// contributesConstants reports whether the file at path can contribute package
// constants. Constant indexing is context for a target and does not select
// targets, so a file that cannot be read or that has an invalid build
// directive is skipped. It does not stop the analysis.
func contributesConstants(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	included, err := buildGoFile(filepath.Dir(path), filepath.Base(path), info)
	return err == nil && included
}

func addFileConstants(byPackage map[string]map[string]struct{}, file *ast.File) {
	constants, ok := byPackage[file.Name.Name]
	if !ok {
		constants = make(map[string]struct{})
		byPackage[file.Name.Name] = constants
	}

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}

		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range value.Names {
				constants[name.Name] = struct{}{}
			}
		}
	}
}

func packageDirectory(filePath string) string {
	return filepath.Dir(physicalPath(filePath))
}

func physicalPath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(absolute); err == nil {
		return filepath.Clean(resolved)
	}

	dir, name := filepath.Split(absolute)
	if resolvedDir, err := filepath.EvalSymlinks(dir); err == nil {
		return filepath.Join(resolvedDir, name)
	}
	return filepath.Clean(absolute)
}
