# Pull Request

## Description

<!-- Provide a brief description of the changes in this PR -->

## Type of Change

<!-- Check all that apply -->

- [ ] New resource
- [ ] New data source
- [ ] Bug fix
- [ ] Enhancement to existing resource/data source
- [ ] Documentation update
- [ ] Refactoring/code cleanup
- [ ] CI/CD changes
- [ ] Other (please describe):

## Resources/Data Sources Modified

<!-- List the resources or data sources added or modified -->

- `jamfplatform_<resource_name>`
- `data.jamfplatform_<data_source_name>`

## Testing

### Automated Tests

- [ ] Added Go unit tests (`schema_test.go`, `input_builders_test.go`, `state_builders_test.go`, and helpers/upgrader tests where relevant)
- [ ] Added Go acceptance tests (`resource_acceptance_test.go` / `datasource_acceptance_test.go`, with `//go:build acceptance`) using `internal/testhelpers` factories
- [ ] `make fix fmt lint test` passes locally
- [ ] `make generate` run and regenerated `docs/` committed alongside the source

### Manual Testing

- [ ] Tested `terraform apply` against a test Jamf instance
- [ ] Tested resource/data source CRUD operations (Create, Read, Update, Delete)
- [ ] Verified resources appear correctly in Jamf Pro UI

### Screenshots

<!-- Include screenshots showing the resources in the Jamf Pro UI -->

<details>
<summary>Resource in Jamf Pro UI</summary>

<!-- Paste screenshot(s) here -->

</details>

## Checklist

- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings or errors
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing Go unit and acceptance tests pass locally with my changes
- [ ] Any dependent changes have been merged and published

## Additional Context

<!-- Add any other context about the PR here -->

## Related Issues

<!-- Link any related issues here using #issue_number -->

Fixes #
Relates to #

---

## For Reviewers

<!-- This section is for reviewers to use during PR review -->

### Review Checklist

- [ ] Code quality and style
- [ ] Test coverage is adequate
- [ ] Documentation is clear and complete
- [ ] Unit tests (and acceptance, where the `acceptance` gate is approved) pass in CI
- [ ] Screenshots show expected behavior in Jamf Pro UI
- [ ] No breaking changes (or properly documented)
