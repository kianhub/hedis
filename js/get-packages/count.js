// For Node.js environment to read files
const fs = require("fs");
const path = require("path"); // Helps in resolving file paths

function countPackageData(data) {
  if (!Array.isArray(data)) {
    console.error("Input must be an array.");
    return {
      firstLevelCount: 0,
      totalVersionCount: 0,
    };
  }

  const firstLevelCount = data.length;
  let totalVersionCount = 0;

  for (const item of data) {
    if (item && Array.isArray(item.package_version)) {
      totalVersionCount += item.package_version.length;
    }
  }

  return {
    firstLevelCount: firstLevelCount,
    totalVersionCount: totalVersionCount,
  };
}

// --- File Reading Implementation ---
const filePath = path.join(__dirname, "react-native-directory-packages.json"); // Assumes packages.json is in the same directory

try {
  // Read the file synchronously
  const fileContent = fs.readFileSync(filePath, "utf8");

  // Parse the JSON content
  const jsonData = JSON.parse(fileContent);

  // Process the data
  const counts = countPackageData(jsonData);

  console.log(`Total first-level array count: ${counts.firstLevelCount}`);
  console.log(`Total versions count: ${counts.totalVersionCount}`);
} catch (error) {
  if (error.code === "ENOENT") {
    console.error(`Error: File not found at ${filePath}`);
  } else if (error instanceof SyntaxError) {
    console.error(`Error parsing JSON: Invalid JSON format in ${filePath}`);
  } else {
    console.error(`An unexpected error occurred: ${error.message}`);
  }
}
