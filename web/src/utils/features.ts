export const feature = (feat:string) => {
    return !Array.isArray(window.quickstart_features) || window.quickstart_features.indexOf(feat) != -1
}
