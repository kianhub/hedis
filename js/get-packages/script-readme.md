# React Native Directories

## Endpoint

`https://reactnative.directory/api/libraries?ios=true&skipTools=true&skipTemplates=true&offset=${someOffset}`
(it's 30 at a time)

## Context

These are react native npm packages that are selected to be working with react native and also maintained.
They are filtered to be supporting ios, are not tools (dev tools), and not templates.

## In script

In the script, all of there packages from react native directories should be pulled (we only really need "npmPkg" field here), so we gather a list of all the package names from this directory

# Github Advisories Database

## Endpoint

`https://api.github.com/advisories?ecosystem=npm&per_page=100`
headers: `{"Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28"}`
and then also, since this is paginated, it has a "Link" header in the response for the rel next page

## Context

These are all npm github vulnerability advisories, these are important because we will check which ones of the react native packages have security vulnerabilities

## In script

In the script, we want check all the react native directories we have grabbed by name against the github security advisories, in the response (array of json objects), there is an array field called "vulnerabilities" with an example below as to how that looks like

```
"vulnerabilities": [
            {
                "package": {
                    "ecosystem": "npm",
                    "name": "@anthropic-ai/claude-code"
                },
                "vulnerable_version_range": ">= 0.2.116, < 1.0.24",
                "first_patched_version": "1.0.24",
                "vulnerable_functions": []
            }
]
```

We want the package name in order to check against the react native directories.
If a package name is found in the advisories, we want to keep track of which versions are vulnerable as well as the ghsa_id, cve_id and url fields from a given github advisory

# NPM registry

## Endpoint

`https://registry.npmjs.org/${npmPkgName}`
The npmPkgName from the react native directories, should be used here

## In Script

From all the react native directories that we have found vulnerabilites for, we want to get data from the npm package registry for them. here, we can also get the exact version numbers of any given package, the only important onces however will only be the ones within the ranges we found in the advisories.
We want to grab all those version numbers for each package that has advisories.
