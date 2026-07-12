package sherpa

import (
	"fmt"
	"reflect"

	"github.com/panndabea/GoSherpa/internal/semantics"
)

type SemanticContextOptions struct {
	BuildTags []string
}

type SemanticContext struct {
	root      string
	buildTags []string

	repositoryLoaded bool
	repository       semantics.Repository
	repositoryErr    error

	testRepositoryLoaded bool
	testRepository       semantics.Repository
	testRepositoryErr    error

	referenceCacheLoaded bool
	referenceCache       *referenceAnalysisCache

	callFunctionsLoaded bool
	callFunctions       []functionInfo
	callWarnings        []string
	callOK              bool

	testCallFunctionsLoaded bool
	testCallFunctions       []functionInfo
	testCallWarnings        []string
	testCallOK              bool
}

var loadSemanticContextRepository = semantics.LoadRepository

func NewSemanticContext(root string, options SemanticContextOptions) (*SemanticContext, error) {
	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	return &SemanticContext{
		root:      rootPath,
		buildTags: semantics.NormalizeBuildTags(options.BuildTags),
	}, nil
}

func (context *SemanticContext) Root() string {
	if context == nil {
		return ""
	}

	return context.root
}

func (context *SemanticContext) BuildTags() []string {
	if context == nil {
		return nil
	}

	return append([]string{}, context.buildTags...)
}

func (context *SemanticContext) TypecheckedRepository() (semantics.Repository, bool, error) {
	if context == nil {
		return semantics.Repository{}, false, fmt.Errorf("semantic context is nil")
	}
	if !referenceShouldAttemptTypechecked(context.root) {
		return semantics.Repository{}, false, nil
	}

	if !context.repositoryLoaded {
		context.repositoryLoaded = true
		context.repository, context.repositoryErr = loadSemanticContextRepository(context.root, semantics.LoadOptions{
			BuildTags: context.buildTags,
		})
	}

	return context.repository, true, context.repositoryErr
}

func (context *SemanticContext) TypecheckedTestRepository() (semantics.Repository, bool, error) {
	if context == nil {
		return semantics.Repository{}, false, fmt.Errorf("semantic context is nil")
	}
	if !referenceShouldAttemptTypechecked(context.root) {
		return semantics.Repository{}, false, nil
	}

	if !context.testRepositoryLoaded {
		context.testRepositoryLoaded = true
		context.testRepository, context.testRepositoryErr = loadSemanticContextRepository(context.root, semantics.LoadOptions{
			IncludeTests: true,
			BuildTags:    context.buildTags,
		})
	}

	return context.testRepository, true, context.testRepositoryErr
}

func (context *SemanticContext) supportsBuildTags(buildTags []string) bool {
	if context == nil {
		return false
	}

	return reflect.DeepEqual(context.buildTags, semantics.NormalizeBuildTags(buildTags))
}

func (context *SemanticContext) referenceAnalysisCache(options ReferenceOptions) *referenceAnalysisCache {
	if context == nil || !context.supportsBuildTags(options.BuildTags) {
		return nil
	}
	if context.referenceCacheLoaded {
		return context.referenceCache
	}

	context.referenceCacheLoaded = true
	context.referenceCache = &referenceAnalysisCache{}
	repo, attempted, err := context.TypecheckedRepository()
	if !attempted {
		return context.referenceCache
	}
	if err != nil {
		context.referenceCache = unavailableReferenceAnalysisCache(fmt.Sprintf("typechecked reference analysis unavailable: %v", err))
		return context.referenceCache
	}

	context.referenceCache = newReferenceAnalysisCacheFromRepository(repo)
	return context.referenceCache
}

func (context *SemanticContext) typecheckedCallFunctionInfos(options CallOptions) ([]functionInfo, []string, bool) {
	if context == nil || !context.supportsBuildTags(options.BuildTags) {
		return nil, nil, false
	}
	if context.callFunctionsLoaded {
		return context.callFunctions, append([]string{}, context.callWarnings...), context.callOK
	}

	context.callFunctionsLoaded = true
	repo, attempted, err := context.TypecheckedRepository()
	if !attempted {
		return nil, nil, false
	}
	if err != nil {
		context.callWarnings = []string{fmt.Sprintf("typechecked call analysis unavailable: %v", err)}
		return nil, append([]string{}, context.callWarnings...), false
	}

	context.callFunctions = semanticCallFunctionInfos(repo)
	context.callWarnings = nonNilStrings(repo.Warnings)
	if len(context.callFunctions) == 0 {
		context.callWarnings = append([]string{"typechecked call analysis unavailable: no typechecked packages loaded"}, repo.Warnings...)
		return nil, append([]string{}, context.callWarnings...), false
	}

	context.callOK = true
	return context.callFunctions, append([]string{}, context.callWarnings...), true
}

func (context *SemanticContext) typecheckedTestCallFunctionInfos(options CallOptions) ([]functionInfo, []string, bool) {
	if context == nil || !context.supportsBuildTags(options.BuildTags) {
		return nil, nil, false
	}
	if context.testCallFunctionsLoaded {
		return context.testCallFunctions, append([]string{}, context.testCallWarnings...), context.testCallOK
	}

	context.testCallFunctionsLoaded = true
	repo, attempted, err := context.TypecheckedTestRepository()
	if !attempted {
		return nil, nil, false
	}
	if err != nil {
		context.testCallWarnings = []string{fmt.Sprintf("typechecked test caller analysis unavailable: %v", err)}
		return nil, append([]string{}, context.testCallWarnings...), false
	}

	context.testCallFunctions = semanticTestCallFunctionInfos(repo)
	context.testCallWarnings = nonNilStrings(repo.Warnings)
	context.testCallOK = true
	return context.testCallFunctions, append([]string{}, context.testCallWarnings...), true
}
