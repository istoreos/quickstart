import axios from "axios";
import { App } from "vue";
import { createGettext,Language } from "vue3-gettext";

const config: {language: Language|undefined, numberFormat:Intl.NumberFormat } = {
    language: undefined,
    numberFormat: new Intl.NumberFormat('en',{notation:"compact"})
}

export const createI18n = async (app: App) => {
    const language = window.vue_lang
    const t = new Date().getTime()
    let translation = {};
    try {
        const res = await axios({
            url: window.vue_lang_data,
            method: "GET"
        })
        if (res.data) {
            translation = res.data
        }
    } catch (error) {
        console.log(error);
    }
    const gettext = createGettext({
        defaultLanguage: language,
        mutedLanguages: ["zh-cn"],
        translations: translation,
        setGlobalProperties: false,
        provideDirective: false,
        provideComponent: false,
    })
    app.use(gettext)
    config.language = gettext
    const { $gettext } = gettext
    window.$i18n = $gettext

    try {
        config.numberFormat = new Intl.NumberFormat(language, {notation:"compact"})
    } catch (e) {
        console.error("Intl.NumberFormat unsupported lang", language, e)
    }
}

const useGettext = (): Language => {
    if (config.language)
        return config.language
    throw new Error("I18N Uninitialized!")
}

const useGettextLazy = () => {
    return {
        $gettext: (msgid: string, parameters?: {
            [key: string]: string;
        }, disableHtmlEscaping?: boolean):string => {
            if (config.language)
                return config.language.$gettext(msgid, parameters, disableHtmlEscaping)
            throw new Error("I18N Uninitialized!")
        },
        $ngettext: (msgid: string, plural: string, n: number, parameters?: {
            [key: string]: string;
        }, disableHtmlEscaping?: boolean):string => {
            if (config.language)
                return config.language.$ngettext(msgid, plural, n, parameters, disableHtmlEscaping)
            throw new Error("I18N Uninitialized!")
        }
    }
}

const formatNumber = (number: number): string => {
    return typeof(number)==='number'?config.numberFormat.format(number):'?';
}

export {
    useGettext,
    useGettextLazy,
    formatNumber,
}


