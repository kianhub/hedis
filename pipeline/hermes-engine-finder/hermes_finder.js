const fs = require("fs");
const path = require("path");

const basePath = path.join(__dirname, "..", "..", "pipeline", "react-natives");

const reactFolders = [
    "rn069",
    "rn070",
    "rn071",
    "rn072",
    "rn073",
    "rn074",
    "rn075",
    "rn076",
    "rn077",
    "rn078",
    "rn079",
];

async function findHermesCompiler() {
    let bytecodeVersions = new Set();

    for (const reactFolder of reactFolders) {
        const targetPath = path.join(
            basePath,
            reactFolder,
            "node_modules",
            "react-native",
            "sdks",
            "hermesc",
            "osx-bin",
            "hermesc"
        );

        if (fs.existsSync(targetPath)) {
            console.log(`🟢 Hermes compiler found in ${targetPath}`);
            const { execSync } = require("child_process");
            try {
                const version = execSync(`${targetPath} -version`)
                    .toString()
                    .trim();
                const bytecodeVersionMatch = version.match(
                    /HBC bytecode version: (\d+)/
                );
                const bytecodeVersion = bytecodeVersionMatch
                    ? bytecodeVersionMatch[1]
                    : null;
                if (bytecodeVersion) {
                    bytecodeVersions.add(bytecodeVersion);
                    console.log(`🔵 Bytecode version: ${bytecodeVersion}`);
                }
            } catch (error) {
                console.error(`Error getting version: ${error.message}`);
            }
        } else {
            console.log(`🔴 Hermes compiler not found in ${targetPath}`);
        }
    }

    console.log(
        `Bytecode versions: ${Array.from(bytecodeVersions).join(", ")}`
    );
}

async function touchNewFiles() {
    for (const reactFolder of reactFolders) {
        const { execSync } = require("child_process");

        const targetPath = path.join(basePath, reactFolder);
        try {
            const res = execSync(`touch ${targetPath}/baseline_entry.js`);
            console.log(res.toString());
        } catch (error) {
            console.error(`Error touching file: ${error.message}`);
        }
    }
}

findHermesCompiler();
//touchNewFiles();
