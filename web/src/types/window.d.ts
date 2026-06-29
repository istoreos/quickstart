interface Window {
    $i18n: (msgid: string, parameters?: {
        [key: string]: string;
    } | undefined, disableHtmlEscaping?: boolean | undefined) => string
    vue_lang: string
    vue_lang_data: string
    vue_base: string,
    quickstart_features?: string[],
    quickstart_configs?: any,
}