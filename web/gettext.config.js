// https://jshmrtn.github.io/vue3-gettext/extraction.html

// const fs = require('fs')
// const locales = []
// fs.readdirSync("./src").forEach(file => {
//     const stat = fs.lstatSync("./src/" + file)
//     if (stat.isDirectory()) {
//         locales.push(file)
//     }
// })
module.exports = {
    input: {
        path: "./src", // only files in this directory are considered for extraction
        include: ["**/*.js", "**/*.ts", "**/*.vue"], // glob patterns to select files for extraction
        exclude: [], // glob patterns to exclude files from extraction
    },
    output: {
        path: "./translations", // output path of all created files
        potPath: "./templates/app.pot", // relative to output.path, so by default "./src/language/messages.pot"
        jsonPath: "./../public/luci-static/quickstart/i18n/", // relative to output.path, so by default "./src/language/translations.json"
        locales: ["en", "zh-cn"],
        flat: false, // don't create subdirectories for locales
        linguas: false, // create a LINGUAS file
        splitJson: true, // create separate json files for each locale. If used, jsonPath must end with a directory, not a file
    },
};