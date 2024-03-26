import IconButton from '@material-ui/core/IconButton';
import {createStyles, Theme, withStyles, WithStyles} from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography';
import {History, Schedule, Update, Delete} from "@material-ui/icons";
import React from 'react';
import TimeAgo from 'react-timeago';
import Container from '../common/Container';
import * as config from '../config';
import {Markdown} from '../common/Markdown';
import {RenderMode, contentType, url} from './extras';
import {IMessageExtras} from '../types';

const styles = (theme: Theme) =>
    createStyles({
        header: {
            display: 'flex',
            flexWrap: 'wrap',
            marginBottom: 0,
        },
        headerTitle: {
            flex: 1,
        },
        actionIcon: {
            marginTop: -15,
            marginRight: -15,
        },
        wrapperPadding: {
            padding: 12,
        },
        messageContentWrapper: {
            width: '100%',
            maxWidth: 585,
        },
        image: {
            marginRight: 15,
            [theme.breakpoints.down('sm')]: {
                width: 32,
                height: 32,
            },
        },
        date: {
            [theme.breakpoints.down('sm')]: {
                order: 1,
                flexBasis: '100%',
                opacity: 0.7,
            },
        },
        imageWrapper: {
            display: 'flex',
        },
        plainContent: {
            whiteSpace: 'pre-wrap',
        },
        content: {
            wordBreak: 'break-all',
            '& p': {
                margin: 0,
            },
            '& a': {
                color: '#ff7f50',
            },
            '& pre': {
                overflow: 'auto',
            },
            '& img': {
                maxWidth: '100%',
            },
        },
        link: {
            color: 'inherit',
            textDecoration: 'none',
        },
    });

interface IProps {
    title: string;
    image?: string;
    date: string;
    content: string;
    priority: number;
    postponed_at?: string;
    fDelete: VoidFunction;
    fPostpone: VoidFunction;
    extras?: IMessageExtras;
    height: (height: number) => void;
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

class Message extends React.PureComponent<IProps & WithStyles<typeof styles>> {
    private node: HTMLDivElement | null = null;

    public componentDidMount = () =>
        this.props.height(this.node ? this.node.getBoundingClientRect().height : 0);

    private renderContent = () => {
        const content = this.props.content;
        switch (contentType(this.props.extras)) {
            case RenderMode.Markdown:
                return <Markdown>{content}</Markdown>;
            case RenderMode.Plain:
            default:
                return <span className={this.props.classes.plainContent}>{content}</span>;
        }
    };

    private renderTitle = () => {
        const title = <Typography className={`${this.props.classes.headerTitle} title`} variant="h5">
            {this.props.title}
        </Typography>
        const notificationUrl = url(this.props.extras);
        if (notificationUrl) {
            return <a href={notificationUrl}
                      target={'_blank'}
                      rel={'noreferrer'}
                      className={`${this.props.classes.headerTitle} ${this.props.classes.link} title`}>
                {title}
            </a>
        }
        return title;
    }

    private renderPostponeIcon = (fPostpone: VoidFunction, postponed_at: string | undefined) => {
        let icon = <Schedule/>;
        let title = "Postpone message";
        if (postponed_at) {
            const postponedDate = new Date(postponed_at);
            title  = "Postponed at " +  postponedDate.toLocaleString();
            if (postponedDate.getTime() > new Date().getTime()) {
                icon = <Update style={{color: '#03a9f4'}}/>;
            } else {
                icon = <History style={{color: '#ffc107'}}/>;
            }
        }
        return <IconButton onClick={fPostpone}
                           title={title}
                           className={this.props.classes.actionIcon}>
            {icon}
        </IconButton>;
    }

    public render(): React.ReactNode {
        const {fDelete, fPostpone, classes, date, image, priority, postponed_at} = this.props;

        return (
            <div className={`${classes.wrapperPadding} message`} ref={(ref) => (this.node = ref)}>
                <Container
                    style={{
                        display: 'flex',
                        borderLeftColor: priorityColor(priority),
                        borderLeftWidth: 6,
                        borderLeftStyle: 'solid',
                    }}>
                    <div className={classes.imageWrapper}>
                        {image !== null ? (
                            <img
                                src={config.get('url') + image}
                                alt="app logo"
                                width="70"
                                height="70"
                                className={classes.image}
                            />
                        ) : null}
                    </div>
                    <div className={classes.messageContentWrapper}>
                        <div className={classes.header}>
                            {this.renderTitle()}
                            <Typography variant="body1" className={classes.date}>
                                <TimeAgo date={date}title={new Date(date).toLocaleString()} />
                            </Typography>
                            {this.renderPostponeIcon(fPostpone, postponed_at)}
                            <IconButton onClick={fDelete} className={`${classes.actionIcon} delete`}>
                                <Delete/>
                            </IconButton>
                        </div>
                        <Typography component="div" className={`${classes.content} content`}>
                            {this.renderContent()}
                        </Typography>
                    </div>
                </Container>
            </div>
        );
    }
}

export default withStyles(styles, {withTheme: true})(Message);
