import https from "https";
import readline from "readline";
import semver from "semver";

async function fetchPackageInfo(packageName) {
  const url = `https://registry.npmjs.org/${packageName}`;
  return new Promise((resolve, reject) => {
    https
      .get(url, (res) => {
        let data = "";
        res.on("data", (chunk) => (data += chunk));
        res.on("end", () => {
          try {
            const json = JSON.parse(data);
            resolve(json);
          } catch (error) {
            reject(error);
          }
        });
      })
      .on("error", reject);
  });
}

async function resolveDependencies(
  packageName,
  versionRange = "latest",
  resolved = new Map()
) {
  const pkgInfo = await fetchPackageInfo(packageName);

  let version;
  if (versionRange === "latest") {
    version = pkgInfo["dist-tags"].latest;
  } else {
    // find max satisfying version
    const versions = Object.keys(pkgInfo.versions);
    version = semver.maxSatisfying(versions, versionRange);
    if (!version) {
      throw new Error(
        `No version found for ${packageName} matching range ${versionRange}`
      );
    }
  }

  const key = `${packageName}@${version}`;
  if (resolved.has(key)) {
    return; // break cyclic recursion loops
  }

  const versionInfo = pkgInfo.versions[version];
  if (!versionInfo) {
    throw new Error(`No version info found for ${packageName}@${version}`);
  }

  resolved.set(key, true);

  const dependencies = versionInfo.dependencies || {};
  for (const [dependencyName, dependencyVersionRange] of Object.entries(
    dependencies
  )) {
    let pkg = dependencyName;
    let range = dependencyVersionRange;
    const alias = parseNpmAlias(dependencyVersionRange);
    if (alias) {
      pkg = alias.name;
      range = alias.versionRange;
    }
    await resolveDependencies(pkg, range, resolved);
  }

  return resolved;
}

function askInput(query) {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise((resolve) => {
    rl.question(query, (answer) => {
      rl.close();
      resolve(answer.trim());
    });
  });
}

function parseNpmAlias(versionRange) {
  if (versionRange.startsWith("npm:")) {
    const alias = versionRange.slice(4); // remove 'npm:'
    const atIndex = alias.lastIndexOf("@");
    if (atIndex > 0) {
      const name = alias.slice(0, atIndex);
      const versionRange = alias.slice(atIndex + 1);
      return { name, versionRange };
    } else {
      // if no @ version specified
      return { name: alias, versionRange: "latest" };
    }
  }

  return null;
}

// run
(async () => {
  try {
    const packageName = await askInput("Enter the npm package name: ");
    if (!packageName) {
      console.error("Package name required.");
      process.exit(1);
    }
    const deps = await resolveDependencies(packageName);
    console.log(`All dependencies for ${packageName}:`);

    const sortedDeps = [...deps.keys()].sort();
    console.log(`${packageName} has ${sortedDeps.length - 1} dependencies:`);
    for (const dep of sortedDeps) {
      // dep is in the format package@version
      const [pkg, ...verArr] = dep.split("@");
      const ver = verArr.join("@");
      try {
        const info = await fetchPackageInfo(pkg);
        console.log(info);
        const versionInfo = info.versions[ver];
        const depCount =
          versionInfo && versionInfo.dependencies
            ? Object.keys(versionInfo.dependencies).length
            : 0;
        console.log(`${dep} (dependencies: ${depCount})`);
      } catch (err) {
        console.log(`${dep} (dependencies: unknown)`);
      }
    }
    // Summary log at the end
    console.log(
      `\nSummary: ${packageName} has a total of ${
        sortedDeps.length - 1
      } dependencies.`
    );
  } catch (error) {
    console.error(error);
    process.exit(1);
  }
})();
