const semver = require("semver");

const version = "^2.0.0";
const range_to_check = ">= 1.5.0  < 2.5.0-canary-1";

//console.log(semver.satisfies("canary", range_to_check));
//console.log(semver.validRange(version));
//console.log(semver.validRange(range_to_check));
console.log(semver.validRange("canary"));
console.log(semver.valid("canary"));
//console.log(semver.intersects(version, "latest"));
