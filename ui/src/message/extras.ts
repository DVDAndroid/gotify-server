import {IMessageExtras} from '../types';

export enum RenderMode {
    Markdown = 'text/markdown',
    Plain = 'text/plain',
}

export const contentType = (extras?: IMessageExtras): RenderMode => {
    const type = extract(extras, 'client::display', 'contentType');
    const valid = Object.values(RenderMode).includes(type);
    return valid ? type : RenderMode.Plain;
};

export const url = (extras?: IMessageExtras): string | null => {
    const click = extract(extras, 'client::notification', 'click');
    try {
        // extras are not trusted: only allow web links, never e.g. javascript: urls
        const parsed = new URL(click?.url);
        return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? parsed.href : null;
    } catch {
        return null;
    }
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const extract = (extras: IMessageExtras | undefined, key: string, path: string): any => {
    if (!extras) {
        return null;
    }

    if (!extras[key]) {
        return null;
    }

    if (!extras[key][path]) {
        return null;
    }

    return extras[key][path];
};
