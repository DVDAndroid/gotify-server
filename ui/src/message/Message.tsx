import {Button, Theme, useMediaQuery, useTheme} from '@mui/material';
import IconButton from '@mui/material/IconButton';
import {makeStyles} from 'tss-react/mui';
import Typography from '@mui/material/Typography';
import {ExpandLess, ExpandMore, History, Schedule, Update} from '@mui/icons-material';
import Delete from '@mui/icons-material/Delete';
import React from 'react';
import TimeAgo from 'react-timeago';
import Container from '../common/Container';
import {Markdown} from '../common/Markdown';
import * as config from '../config';
import {IMessageExtras} from '../types';
import {contentType, RenderMode, url} from './extras';
import {TimeAgoFormatter} from '../common/TimeAgoFormatter';

const PREVIEW_LENGTH = 500;

const useStyles = makeStyles()((theme: Theme) => ({
    header: {
        display: 'flex',
        width: '100%',
        alignItems: 'start',
        alignContent: 'center',
        paddingBottom: 5,
        wordBreak: 'break-all',
    },
    headerTitle: {
        flex: 1,
    },
    trash: {
        marginTop: -15,
        marginRight: -15,
    },
    wrapperPadding: {
        marginBottom: theme.spacing(2),
        [theme.breakpoints.down('sm')]: {
            marginBottom: theme.spacing(1),
        },
    },
    messageContentWrapper: {
        minWidth: 200,
        width: '100%',
    },
    image: {
        width: 50,
        height: 50,
        [theme.breakpoints.down('md')]: {
            width: 30,
            height: 30,
        },
    },
    date: {
        [theme.breakpoints.down('md')]: {
            order: 1,
            flexBasis: '100%',
            opacity: 0.7,
        },
    },
    imageWrapper: {
        marginRight: 15,
        width: 50,
        height: 50,
    },
    plainContent: {
        whiteSpace: 'pre-wrap',
    },
    content: {
        maxHeight: PREVIEW_LENGTH,
        wordBreak: 'break-all',
        overflowY: 'hidden',
        '&.expanded': {
            maxHeight: 'none',
        },
        '& p': {
            margin: 0,
            wordBreak: 'break-word',
        },
        '& a': {
            color: '#ff7f50',
        },
        '& pre': {
            overflow: 'auto',
            borderRadius: '0.25em',
            backgroundColor:
                theme.palette.mode === 'dark' ? 'rgba(255,255,255,0.05)' : 'rgba(0,0,0,0.05)',
            padding: theme.spacing(1),
        },
        '& img': {
            maxWidth: '100%',
        },
    },
}));

interface IProps {
    title: string;
    image?: string;
    date: string;
    content: string;
    priority: number;
    appName: string;
    postponedAt?: string | null;
    fDelete: VoidFunction;
    fPostpone: VoidFunction;
    extras?: IMessageExtras;
    expanded: boolean;
    onExpand: (expand: boolean) => void;
}

const priorityColor = (priority: number) => {
    if (priority >= 4 && priority <= 7) {
        return 'rgba(230, 126, 34, 0.7)';
    } else if (priority > 7) {
        return '#e74c3c';
    } else {
        return 'transparent';
    }
};

const Message = ({
    fDelete,
    fPostpone,
    title,
    date,
    postponedAt,
    image,
    priority,
    content,
    extras,
    appName,
    onExpand,
    expanded: initialExpanded,
}: IProps) => {
    const theme = useTheme();
    const contentRef = React.useRef<HTMLDivElement | null>(null);
    const {classes} = useStyles();
    const [expanded, setExpanded] = React.useState(initialExpanded);
    const [isOverflowing, setOverflowing] = React.useState(false);
    const smallHeader = useMediaQuery(theme.breakpoints.down('md'));

    const refreshOverflowing = React.useCallback(() => {
        const ref = contentRef.current;
        if (!ref) {
            return;
        }
        setOverflowing((overflowing) => overflowing || ref.scrollHeight > ref.clientHeight);
    }, [contentRef, setOverflowing]);

    const onContentRef = React.useCallback(
        (ref: HTMLDivElement | null) => {
            contentRef.current = ref;
            refreshOverflowing();
        },
        [contentRef, refreshOverflowing]
    );

    React.useEffect(() => void onExpand(expanded), [expanded]);

    const togglePreviewHeight = () => setExpanded((b) => !b);

    const renderContent = () => {
        switch (contentType(extras)) {
            case RenderMode.Markdown:
                return <Markdown onImageLoaded={refreshOverflowing}>{content}</Markdown>;
            case RenderMode.Plain:
            default:
                return <span className={classes.plainContent}>{content}</span>;
        }
    };
    return (
        <div className={`${classes.wrapperPadding} message`}>
            <Container
                style={{
                    display: 'flex',
                    flexWrap: 'wrap',
                    borderLeftColor: priorityColor(priority),
                    borderLeftWidth: 6,
                    borderLeftStyle: 'solid',
                }}>
                {smallHeader ? (
                    <HeaderSmall
                        fDelete={fDelete}
                        fPostpone={fPostpone}
                        title={title}
                        appName={appName}
                        image={image}
                        date={date}
                        postponedAt={postponedAt}
                        extras={extras}
                    />
                ) : (
                    <HeaderWide
                        fDelete={fDelete}
                        fPostpone={fPostpone}
                        title={title}
                        appName={appName}
                        image={image}
                        date={date}
                        postponedAt={postponedAt}
                        extras={extras}
                    />
                )}

                <div className={classes.messageContentWrapper}>
                    <Typography
                        component="div"
                        ref={onContentRef}
                        className={`${classes.content} content ${
                            isOverflowing && expanded ? 'expanded' : ''
                        }`}>
                        {renderContent()}
                    </Typography>
                </div>
                {isOverflowing && (
                    <Button
                        style={{marginTop: 16}}
                        onClick={togglePreviewHeight}
                        variant="contained"
                        color="primary"
                        size="large"
                        fullWidth={true}
                        startIcon={expanded ? <ExpandLess /> : <ExpandMore />}>
                        {expanded ? 'Read Less' : 'Read More'}
                    </Button>
                )}
            </Container>
        </div>
    );
};

type HeaderProps = Pick<
    IProps,
    'appName' | 'image' | 'fDelete' | 'fPostpone' | 'date' | 'postponedAt' | 'title' | 'extras'
>;

// The title links to client::notification::click::url if set, see https://gotify.net/docs/msgextras#clickurl
const Title = ({title, extras}: Pick<IProps, 'title' | 'extras'>) => {
    const href = url(extras);
    const text = (
        <Typography className="title" variant="h5" sx={{lineHeight: 1.2}}>
            {title}
        </Typography>
    );
    return href ? (
        <a
            href={href}
            target="_blank"
            rel="noreferrer"
            style={{color: 'inherit', textDecoration: 'none'}}>
            {text}
        </a>
    ) : (
        text
    );
};

const PostponeButton = ({postponedAt, fPostpone}: Pick<IProps, 'postponedAt' | 'fPostpone'>) => {
    const {classes} = useStyles();
    let icon = <Schedule />;
    let title = 'Postpone message';
    if (postponedAt) {
        const postponed = new Date(postponedAt);
        title = `Postponed to ${postponed.toLocaleString()}`;
        icon =
            postponed.getTime() > Date.now() ? (
                <Update color="info" />
            ) : (
                <History color="warning" />
            );
    }
    return (
        <IconButton
            onClick={fPostpone}
            title={title}
            style={{padding: 14}}
            className={`${classes.trash} postpone`}
            size="large">
            {icon}
        </IconButton>
    );
};

const HeaderWide = ({
    appName,
    image,
    date,
    postponedAt,
    fDelete,
    fPostpone,
    title,
    extras,
}: HeaderProps) => {
    const {classes} = useStyles();

    return (
        <div className={classes.header}>
            <div className={classes.imageWrapper}>
                {image !== null ? (
                    <img
                        src={config.get('url') + image}
                        alt={`${appName} logo`}
                        width="50"
                        height="50"
                        className={classes.image}
                    />
                ) : null}
            </div>
            <div className={classes.headerTitle}>
                <Title title={title} extras={extras} />
                <Typography variant="subtitle1" sx={{fontSize: 12, opacity: 0.7}}>
                    {appName}
                </Typography>
            </div>
            <Typography variant="body1" className={classes.date}>
                <TimeAgo
                    date={date}
                    formatter={TimeAgoFormatter.narrow}
                    title={new Date(date).toLocaleString()}
                />
            </Typography>
            <PostponeButton postponedAt={postponedAt} fPostpone={fPostpone} />
            <IconButton
                onClick={fDelete}
                style={{padding: 14}}
                className={`${classes.trash} delete`}
                size="large">
                <Delete />
            </IconButton>
        </div>
    );
};
const HeaderSmall = ({
    appName,
    image,
    date,
    postponedAt,
    fDelete,
    fPostpone,
    title,
    extras,
}: HeaderProps) => {
    const {classes} = useStyles();

    return (
        <div className={classes.header}>
            <div className={classes.headerTitle}>
                <Title title={title} extras={extras} />
                <Typography variant="subtitle1" sx={{fontSize: 12, opacity: 0.7}}>
                    {appName}
                </Typography>
                <Typography variant="body1" className={classes.date}>
                    <TimeAgo
                        date={date}
                        formatter={TimeAgoFormatter.long}
                        title={new Date(date).toLocaleString()}
                    />
                </Typography>
            </div>
            <div style={{display: 'flex', alignItems: 'end', flexDirection: 'column'}}>
                <div style={{display: 'flex'}}>
                    <PostponeButton postponedAt={postponedAt} fPostpone={fPostpone} />
                    <IconButton
                        onClick={fDelete}
                        style={{padding: 14}}
                        className={`${classes.trash} delete`}
                        size="large">
                        <Delete />
                    </IconButton>
                </div>
                <div style={{width: 30, height: 30}}>
                    {image !== null ? (
                        <img
                            src={config.get('url') + image}
                            alt={`${appName} logo`}
                            className={classes.image}
                        />
                    ) : null}
                </div>
            </div>
        </div>
    );
};

export default Message;
