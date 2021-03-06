package godm

import (
	"io/ioutil"
	"testing"

	"os"
	"path"

	"github.com/hectorj/godm/git"
	"github.com/stretchr/testify/assert"
)

func newProjectNoVCLProject(t *testing.T) (project *ProjectNoVCL) {
	tmpDirPath, err := ioutil.TempDir("", "godm-project_no_vcl_test")
	assert.Nil(t, err, "Failed creating a temp dir for ProjectNoVCL tests.")

	project = NewProjectNoVCL(tmpDirPath)
	assert.Equal(t, tmpDirPath, project.GetBaseDir(), "Fresh ProjectNoVCL from temp dir has a wrong base dir.")

	return
}

var testCases = map[string]struct {
	prepare             func(t *testing.T, project *ProjectNoVCL, failMessageAndArgs ...interface{})
	expectedImportPaths map[string]LocalProject
}{
	"none": {
		prepare:             func(_ *testing.T, _ *ProjectNoVCL, _ ...interface{}) {},
		expectedImportPaths: nil,
	},
	"1 NoVCL vendor": {
		prepare: func(t *testing.T, project *ProjectNoVCL, failMessageAndArgs ...interface{}) {
			targetDirPath := path.Join(project.GetBaseDir(), "vendor", "test1") + "/"
			err := os.MkdirAll(targetDirPath, os.ModeDir|os.ModePerm)
			assert.Nil(t, err, failMessageAndArgs...)

			targetFilePath := path.Join(targetDirPath, "whatever.go")
			file, err := os.Create(targetFilePath)
			assert.Nil(t, err, failMessageAndArgs...)

			_, err = file.WriteString("package test1\n")
			assert.Nil(t, err, failMessageAndArgs...)

			git.Service = git.NewGitStub()
		},
		expectedImportPaths: map[string]LocalProject{
			"test1": &ProjectNoVCL{},
		},
	},
	"1 Git vendor": {
		prepare: func(t *testing.T, project *ProjectNoVCL, failMessageAndArgs ...interface{}) {
			targetDirPath := path.Join(project.GetBaseDir(), "vendor", "test1") + "/"
			err := os.MkdirAll(targetDirPath, os.ModeDir|os.ModePerm)
			assert.Nil(t, err, failMessageAndArgs...)

			targetFilePath := path.Join(targetDirPath, "whatever.go")
			file, err := os.Create(targetFilePath)
			assert.Nil(t, err, failMessageAndArgs...)

			_, err = file.WriteString("package test1\n")
			assert.Nil(t, err, failMessageAndArgs...)

			gitStub := git.NewGitStub()

			gitStub.Repos[targetDirPath] = git.NewGitRepoStub()

			git.Service = gitStub
		},
		expectedImportPaths: map[string]LocalProject{
			"test1": &localGitProject{},
		},
	},
}

func TestProjectNoVCL_GetVendors(t *testing.T) {
testCasesLoop:
	for caseName, testCase := range testCases {
		failMessageAndArgs := []interface{}{"Test case %q failed.", caseName}
		project := newProjectNoVCLProject(t)
		defer os.RemoveAll(project.GetBaseDir())

		testCase.prepare(t, project, failMessageAndArgs...)

		vendors, err := project.GetVendors()
		if !assert.Nil(t, err, failMessageAndArgs...) {
			continue testCasesLoop
		}

		if !assert.Len(t, vendors, len(testCase.expectedImportPaths), failMessageAndArgs...) {
			continue testCasesLoop
		}

	vendorsLoop:
		for importPath, projectType := range testCase.expectedImportPaths {
			vendor, exists := vendors[importPath]

			specificFailMessageAndArgs := make([]interface{}, len(failMessageAndArgs)+1)
			copy(specificFailMessageAndArgs, failMessageAndArgs)
			specificFailMessageAndArgs[0] = specificFailMessageAndArgs[0].(string) + " (vendor %q)"
			specificFailMessageAndArgs[len(failMessageAndArgs)] = importPath

			if !assert.True(t, exists, specificFailMessageAndArgs...) {
				continue vendorsLoop
			}

			if !assert.NotNil(t, vendor, specificFailMessageAndArgs...) {
				continue vendorsLoop
			}

			if !assert.IsType(t, projectType, vendor.GetProject(), specificFailMessageAndArgs...) {
				continue vendorsLoop
			}
		}
	}

}

func TestProjectNoVCL_RemoveVendor(t *testing.T) {
	//testCasesLoop:
	for caseName, testCase := range testCases {
		failMessageAndArgs := []interface{}{"Test case %q failed.", caseName}
		project := newProjectNoVCLProject(t)
		defer os.RemoveAll(project.GetBaseDir())

		testCase.prepare(t, project, failMessageAndArgs...)

	vendorsLoop:
		for importPath, _ := range testCase.expectedImportPaths {
			specificFailMessageAndArgs := make([]interface{}, len(failMessageAndArgs)+1)
			copy(specificFailMessageAndArgs, failMessageAndArgs)
			specificFailMessageAndArgs[0] = specificFailMessageAndArgs[0].(string) + " (vendor %q)"
			specificFailMessageAndArgs[len(failMessageAndArgs)] = importPath

			vendorPath := path.Join(project.GetBaseDir(), "vendor", importPath)

			// Check that vendor's dir exists
			_, err := os.Stat(vendorPath)

			if !assert.Nil(t, err, specificFailMessageAndArgs...) {
				continue vendorsLoop
			}

			// Remove vendor
			err = project.RemoveVendor(importPath)

			if !assert.Nil(t, err, specificFailMessageAndArgs...) {
				continue vendorsLoop
			}

			// Check that vendor's dir doesn't exist anymore
			_, err = os.Stat(vendorPath)

			if !assert.True(t, os.IsNotExist(err), specificFailMessageAndArgs...) {
				continue vendorsLoop
			}
		}

		// Check that PROJECTPATH/vendor is empty or doesn't exists
		fileInfos, err := ioutil.ReadDir(path.Join(project.GetBaseDir(), "vendor"))

		if os.IsNotExist(err) || assert.Nil(t, err, failMessageAndArgs...) {
			assert.Len(t, fileInfos, 0)
		}
	}

}

func TestProjectNoVCL_AddVendor(t *testing.T) {
	project := newProjectNoVCLProject(t)
	defer os.RemoveAll(project.GetBaseDir())

	futureVendor := newProjectNoVCLProject(t)
	defer os.RemoveAll(futureVendor.GetBaseDir())

	importPath := "completely/madeup/importpath"
	actualVendor, err := project.AddVendor(importPath, futureVendor)

	assert.Nil(t, err)

	assert.Equal(t, path.Join(project.GetBaseDir(), "vendor", importPath), actualVendor.GetBaseDir())

	fileInfo, err := os.Stat(actualVendor.GetBaseDir())

	assert.Nil(t, err)
	assert.True(t, fileInfo.IsDir())
}
