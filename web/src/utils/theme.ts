export const getTheme = () => {
    const bodyTheme = document.body.getAttribute("theme")
    if (bodyTheme) {
        switch (bodyTheme) {
            case "dark":
            case "light":
                return bodyTheme
        }
    }
    return window.matchMedia("(prefers-color-scheme: dark)")?.matches ? "dark" : "light"
}

export const isDark = () => {
    return getTheme() == "dark"
}
export const isLight = () => {
    return getTheme() == "light"
}