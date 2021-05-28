



















###### Issue

#### What type of PR is this?

/kind cleanup

#### What this PR does / why we need it:

The package that we currently use for embedding binary data (go-bindata) is largely unmaintained and we have had to update references to the package (ref: https://github.com/kubernetes/kubernetes/issues/96169). Go 1.16 has built-in support for embedding files in the compiled binaries.

This pull request explores removing the dependency on go-bindata. Currently, we use go-bindata for packaging the following:
- [x] kubectl translations data
- [ ] Conformance test fixtures

#### Which issue(s) this PR fixes:

Fixes #99150

#### Special notes for your reviewer:

##### kubectl translations

Go 1.16's embed directive doesn't allow embedding files from parent directories. Hence, moving the translations data to inside the i18n package.

Logically speaking as well, kubectl related artifacts should be inside the kubectl package. I think historically the files were at the root of k/k because of the generate script.

##### Conformance test fixtures

The conformance test fixtures are spread over several packages and hence not as trivial to embed.

	"test/conformance/testdata/..."
	"test/e2e/testing-manifests/..."
	"test/e2e_node/testing-manifests/..."
	"test/images/..."
	"test/fixtures/..."

One idea

##### References

https://golang.org/pkg/embed/
https://go.googlesource.com/proposal/+/master/design/draft-embed.md

#### Does this PR introduce a user-facing change?

```release-note
NONE
```

#### Additional documentation e.g., KEPs (Kubernetes Enhancement Proposals), usage docs, etc.:

```docs
NONE
```

/assign
/priority important-soon

/sig architecture
/cc @dims

/sig cli
/area kubectl
/cc @soltysh
