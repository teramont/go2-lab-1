package gomodule

import (
	"fmt"
	"path"
	"regexp"
	"github.com/google/blueprint"
	"github.com/roman-mazur/bood"
)

var (
	// Package context for Ninja build rules
	pctx = blueprint.NewPackageContext("github.com/teramont/go2-lab-1/build/gomodule/testbin")
	// Ninja rule to build.
	goBuild = pctx.StaticRule("binaryBuild", blueprint.RuleParams{
		Command:     "cd $workDir && go build -o $outputPath $pkg",
		Description: "build go command $pkg",
	}, "workDir", "outputPath", "pkg")
	// Ninja rule to run go mod vendor
	goVendor = pctx.StaticRule("vendor", blueprint.RuleParams{
		Command:     "cd $workDir && go mod vendor",
		Description: "vendor dependencies of $name",
	}, "workDir", "name")
	// Ninja rule to test
	goTest = pctx.StaticRule("test", blueprint.RuleParams{
		Command:     "cd ${workDir} && go test -v ${testPkg} > ${outPath}",
		Description: "test ${testPkg}",
	}, "workDir", "outPath", "testPkg")
)

type testedBinaryModule struct {
	blueprint.SimpleName

	properties struct {
		// Builded package
		Pkg string
		// Tested package
		TestPkg string
		// Source files
		Srcs []string
		// Source files to exclude
		SrcsExclude []string
		// Call vendor command first
		VendorFirst bool
	}
}

func (tb *testedBinaryModule) GenerateBuildActions(ctx blueprint.ModuleContext) {
	name := ctx.ModuleName()
	config := bood.ExtractConfig(ctx)
	config.Debug.Printf("Adding build actions for module '%s'", name)

	outputPath := path.Join(config.BaseOutputDir, "bin", name)

	testOutPath := path.Join(config.BaseOutputDir, "out.txt")

	var srcInputs []string
	var testInputs []string

	inputErrors := false

	for _, src := range tb.properties.Srcs {
		if matches, err := ctx.GlobWithDeps(src, tb.properties.SrcsExclude); err == nil {
			for _, srcPath := range matches {
				isTestFile, err := regexp.MatchString("^.*_test.go$", srcPath)
				if err != nil {
					ctx.PropertyErrorf("srcs", "Problem with %s", srcPath)
					inputErrors = true
					break
				}

				if isTestFile {
					testInputs = append(testInputs, srcPath)
				} else {
					srcInputs = append(srcInputs, srcPath)
				}
			}
		} else {
			ctx.PropertyErrorf("srcs", "Can not resolve %s", src)
			inputErrors = true
		}
	}

	if inputErrors {
		return
	}

	if tb.properties.VendorFirst {
		vendorDirPath := path.Join(ctx.ModuleDir(), "vendor")
		ctx.Build(pctx, blueprint.BuildParams{
			Description: fmt.Sprintf("Vendor dependencies of %s", name),
			Rule:        goVendor,
			Outputs:     []string{vendorDirPath},
			Implicits:   []string{path.Join(ctx.ModuleDir(), "go.mod")},
			Optional:    true,
			Args: map[string]string{
				"workDir": ctx.ModuleDir(),
				"name":    name,
			},
		})
		srcInputs = append(srcInputs, vendorDirPath)
	}

	ctx.Build(pctx, blueprint.BuildParams{
		Description: fmt.Sprintf("Test %s", tb.properties.TestPkg),
		Rule:        goTest,
		Outputs:     []string{testOutPath},
		Implicits:   append(srcInputs, testInputs...),
		Args: map[string]string{
			"outPath": testOutPath,
			"workDir": ctx.ModuleDir(),
			"testPkg": tb.properties.TestPkg,
		},
	})

	ctx.Build(pctx, blueprint.BuildParams{
		Description: fmt.Sprintf("Build %s", name),
		Rule:        goBuild,
		Outputs:     []string{outputPath},
		Implicits:   srcInputs,
		Args: map[string]string{
			"outputPath": outputPath,
			"workDir":    ctx.ModuleDir(),
			"pkg":        tb.properties.Pkg,
		},
	})
}

func TestedBinaryFactory() (blueprint.Module, []interface{}) {
	mType := &testedBinaryModule{}
	return mType, []interface{}{&mType.SimpleName.Properties, &mType.properties}
}
