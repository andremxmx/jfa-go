import { _get } from "../modules/common.js";

interface Meta {
    name: string;
}

interface quantityString {
    singular: string;
    plural: string;
}

export interface LangFile {
    meta: Meta;
    strings: { [key: string]: string };
    notifications: { [key: string]: string };
    quantityStrings: { [key: string]: quantityString };
}

export class lang implements Lang {
    private _lang: LangFile;
    constructor(lang: LangFile) {
        this._lang = lang;
    }

    get = (sect: string, key: string): string => {
        if (sect == "quantityStrings" || sect == "meta") { return ""; }
        return this._lang[sect][key];
    }

    strings = (key: string): string => this.get("strings", key)
    notif = (key: string): string => this.get("notifications", key)

    var = (sect: string, key: string, ...subs: string[]): string => {
        if (sect == "quantityStrings" || sect == "meta") { return ""; }
        let str = this._lang[sect][key];
        for (let sub of subs) {
            str = str.replace("{n}", sub);
        }
        return str;
    }

    quantity = (key: string, number: number): string => {
        if (number == 1) {
            return this._lang.quantityStrings[key].singular.replace("{n}", ""+number)
        }
        return this._lang.quantityStrings[key].plural.replace("{n}", ""+number);
    }
}

export const loadLangSelector = (page: string) => {
    if (page == "admin") {
        const ev = new CustomEvent("timefmt-change");
        const setTimefmt = (fmt: string) => {
            document.dispatchEvent(ev);
            localStorage.setItem("timefmt", fmt);
        };
        const t12 = document.getElementById("lang-12h") as HTMLInputElement;
        t12.onchange = () => setTimefmt("12h");
        const t24 = document.getElementById("lang-24h") as HTMLInputElement;
        t24.onchange = () => setTimefmt("24h");

        const preference = localStorage.getItem("timefmt");
        if (preference == "12h") {
            t12.checked = true;
            t24.checked = false;
        } else if (preference == "24h") {
            t24.checked = true;
            t12.checked = false;
        }
    }
    _get("/lang/" + page, null, (req: XMLHttpRequest) => {
        if (req.readyState == 4) {
            if (req.status != 200) {
                document.getElementById("lang-dropdown").remove();
                return;
            }
            const list = document.getElementById("lang-list") as HTMLDivElement;
            let innerHTML = '';
            for (let code in req.response) {
                innerHTML += `<a href="?lang=${code}" class="button input ~neutral field mb-half lang-link">${req.response[code]}</a>`;
            }
            list.innerHTML = innerHTML;
        }
    });
};
