const fs = require("fs");
const path = require("path");

const fetchFn =
    typeof fetch === "function"
        ? fetch
        : (...args) =>
              import("node-fetch").then(({ default: f }) => f(...args));

async function fetchJSON(url) {
    const res = await fetchFn(url, {
        headers: { "user-agent": "react-native-directory-scraper" },
    });
    if (!res.ok) {
        throw new Error(`Request failed ${res.status} → ${url}`);
    }
    return res.json();
}

const NUMERIC_VERSION = /^\d+\.\d+\.\d+$/;

async function getAllLibraries() {
    const libs = [];
    const PAGE = 30;
    let offset = 0;
    let total = Infinity;

    while (offset < total) {
        const url =
            `https://reactnative.directory/api/libraries` +
            `?ios=true&skipTools=true&skipTemplates=true&offset=${offset}`;

        const { libraries = [], total: t = 0 } = await fetchJSON(url);
        total = t;
        libs.push(...libraries);
        offset += PAGE;
        console.log(`Fetched ${libs.length}/${total} directory entries…`);
    }
    return libs.filter((l) => l.npmPkg);
}

async function buildPackageData(npmPkg) {
    const data = await fetchJSON(
        `https://registry.npmjs.org/${encodeURIComponent(npmPkg)}`
    );

    const versions = Object.keys(data.versions || {}).filter((v) =>
        NUMERIC_VERSION.test(v)
    );

    const depMap = {};
    const devMap = {};
    const peerMap = {};
    const scriptMap = {};

    for (const v of versions) {
        const meta = data.versions[v] || {};
        if (meta.dependencies) depMap[v] = meta.dependencies;
        if (meta.devDependencies) devMap[v] = meta.devDependencies;
        if (meta.peerDependencies) peerMap[v] = meta.peerDependencies;
        if (meta.scripts) scriptMap[v] = meta.scripts;
    }

    const authorUrl =
        typeof data.author === "object" && data.author
            ? data.author.url ?? null
            : null;

    const repo =
        typeof data.repository === "object"
            ? data.repository.url
            : data.repository ?? null;

    return {
        package_name: data.name || npmPkg,
        package_version: versions,
        package_author: authorUrl, // spelling kept as requested
        package_homepage: data.homepage ?? null,
        package_repository: repo,
        package_dependencies: depMap,
        package_dev_dependencies: devMap,
        package_peer_dependencies: peerMap,
        package_scripts: scriptMap,
    };
}

async function main() {
    try {
        const libs = await getAllLibraries();

        const results = [];
        const CONCURRENCY = 10;
        let index = 0;

        async function worker() {
            while (index < libs.length) {
                const i = index++;
                const pkg = libs[i].npmPkg;
                try {
                    const obj = await buildPackageData(pkg);
                    results.push(obj);
                    console.log(`✔ processed ${pkg}`);
                } catch (e) {
                    console.warn(`✖ failed ${pkg}: ${e.message}`);
                }
            }
        }

        // launch a small pool of concurrent workers so we get some parallelization in this thing
        await Promise.all(Array.from({ length: CONCURRENCY }, worker));

        const outFile = path.join(
            __dirname,
            "react-native-directory-packages.json"
        );
        fs.writeFileSync(outFile, JSON.stringify(results, null, 2));
        console.log(`\n✅ Saved ${results.length} packages to ${outFile}`);
    } catch (err) {
        console.error(err);
        process.exitCode = 1;
    }
}

main();
