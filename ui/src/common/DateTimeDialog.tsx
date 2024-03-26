import Button from '@material-ui/core/Button';
import Dialog from '@material-ui/core/Dialog';
import DialogActions from '@material-ui/core/DialogActions';
import DialogContent from '@material-ui/core/DialogContent';
import DialogTitle from '@material-ui/core/DialogTitle';
import React, {useEffect} from 'react';
import dayjs, {Dayjs} from "dayjs";
import {DateTimePicker} from "@material-ui/pickers";
import TextField from "@material-ui/core/TextField";

interface IProps {
    title: string;
    defaultDatetime?: string;
    fClose: VoidFunction;
    fOnSubmit: (datetime: Date | undefined) => void;
}

export default function DateTimeDialog({title, defaultDatetime, fClose, fOnSubmit}: IProps) {
    const datetimeState = React.useState<Dayjs | null>(dayjs(new Date()));
    let datetime = datetimeState[0];
    const setDatetime = datetimeState[1];

    useEffect(() => {
        let newDatetime = new Date();
        if (defaultDatetime) {
            newDatetime = new Date(defaultDatetime);
        }
        newDatetime.setMinutes(newDatetime.getMinutes() + 15);
        newDatetime.setSeconds(0);
        newDatetime.setMilliseconds(0);
        setDatetime(dayjs(newDatetime));
    }, [defaultDatetime])

    const submitAndClose = () => {
        fOnSubmit(datetime?.toDate());
        fClose();
    };
    return (
        <Dialog
            disableEnforceFocus // https://github.com/mui/material-ui-pickers/issues/1852#issuecomment-682521200
            open={true}
            onClose={fClose}
            aria-labelledby="form-dialog-title"
            className="datetime-dialog">
            <DialogTitle id="form-dialog-title">{title}</DialogTitle>
            <DialogContent>
                <DateTimePicker value={datetime}
                                allowSameDateSelection
                                ampm={false}
                                inputFormat={"DD/MM/YYYY HH:mm:ss"}
                                mask={"__/__/____ __:__:__"}
                                disablePast
                                onChange={date => setDatetime(date)}
                                renderInput={(props) =>
                                    <TextField variant="outlined" {...props} helperText={null}/>
                                }
                />
            </DialogContent>
            <DialogActions>
                {defaultDatetime && <Button
                    onClick={() => {
                        datetime = null;
                        submitAndClose();
                    }}
                    title={"Delete postponement"}
                    variant={"outlined"}
                    color={"secondary"}
                    className="cancel">
                    Delete
                </Button>}
                <Button
                    onClick={submitAndClose}
                    autoFocus
                    color="primary"
                    variant="contained"
                    className="confirm">
                    OK
                </Button>
                <Button onClick={fClose} className="cancel">
                    Cancel
                </Button>
            </DialogActions>
        </Dialog>
    );
}
