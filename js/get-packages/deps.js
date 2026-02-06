const fs = require("fs");
const path = require("path");

const packageJsonPath = "./keybase-package.json";

try {
  const packageJsonContent = fs.readFileSync(packageJsonPath, "utf8");
  const packageJson = JSON.parse(packageJsonContent);

  // 1) Count the amount of "dependencies"
  let dependencyCount = 0;
  if (packageJson.dependencies) {
    dependencyCount = Object.keys(packageJson.dependencies).length;
    console.log(`Amount of dependencies: ${dependencyCount}`);
  } else {
    console.log('No "dependencies" section found.');
  }

  // 2) Check if and which versions of specified packages are present
  const packagesToCheck = [
    "react-native-mmkv",
    "semver",
    "better-auth",
    "axios",
    "react-native-keys",
    "react-native-document-picker",
    "i18next",
    "@sentry/react-native",
    "@nozbe/watermelondb",
    "crypto-js",
    "react-native-fast-image",
    "react-native-reanimated",
    "react-native-webview",
    "zod",
    "immer",
    "parse",
  ];

  console.log("\nChecking for specific packages:");
  if (packageJson.dependencies) {
    packagesToCheck.forEach((pkg) => {
      if (packageJson.dependencies[pkg]) {
        console.log(`- ${pkg}: ${packageJson.dependencies[pkg]}`);
      } else {
        console.log(`- ${pkg}: Not found`);
      }
    });
  } else {
    console.log('No "dependencies" section to check packages in.');
  }
} catch (error) {
  if (error.code === "ENOENT") {
    console.error(`Error: The file "${packageJsonPath}" was not found.`);
  } else if (error instanceof SyntaxError) {
    console.error(
      `Error: Invalid JSON in "${packageJsonPath}".`,
      error.message,
    );
  } else {
    console.error(`An unexpected error occurred:`, error.message);
  }
}
