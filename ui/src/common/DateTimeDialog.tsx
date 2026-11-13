import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import TextField from '@mui/material/TextField';
import React from 'react';

interface IProps {
    title: string;
    /** The current date, if there is one: enables the button to remove it. */
    defaultDatetime?: string | null;
    fClose: VoidFunction;
    /** Called with the chosen date, or with undefined if the date should be removed. */
    fOnSubmit: (datetime: Date | undefined) => void;
}

// the value of <input type="datetime-local"> is a local date time: YYYY-MM-DDTHH:mm
const toInputValue = (date: Date): string => {
    const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
    return local.toISOString().slice(0, 16);
};

const initialDate = (defaultDatetime?: string | null): Date => {
    const existing = defaultDatetime ? new Date(defaultDatetime) : null;
    if (existing && existing.getTime() > Date.now()) {
        return existing;
    }
    return new Date(Date.now() + 15 * 60000);
};

export default function DateTimeDialog({title, defaultDatetime, fClose, fOnSubmit}: IProps) {
    const [value, setValue] = React.useState(() => toInputValue(initialDate(defaultDatetime)));
    const datetime = new Date(value);
    const valid = value !== '' && datetime.getTime() > Date.now();

    const submitAndClose = (date: Date | undefined) => {
        fOnSubmit(date);
        fClose();
    };

    return (
        <Dialog
            open={true}
            onClose={fClose}
            aria-labelledby="datetime-dialog-title"
            className="datetime-dialog">
            <DialogTitle id="datetime-dialog-title">{title}</DialogTitle>
            <DialogContent>
                <TextField
                    autoFocus
                    margin="dense"
                    className="datetime"
                    type="datetime-local"
                    value={value}
                    onChange={(e) => setValue(e.target.value)}
                    error={!valid}
                    helperText={valid ? undefined : 'Choose a date in the future'}
                    slotProps={{htmlInput: {min: toInputValue(new Date())}}}
                    fullWidth
                />
            </DialogContent>
            <DialogActions>
                {defaultDatetime && (
                    <Button
                        onClick={() => submitAndClose(undefined)}
                        title="Remove the postponement"
                        color="secondary"
                        variant="outlined"
                        className="remove">
                        Remove
                    </Button>
                )}
                <Button onClick={fClose} className="cancel">
                    Cancel
                </Button>
                <Button
                    onClick={() => submitAndClose(datetime)}
                    disabled={!valid}
                    color="primary"
                    variant="contained"
                    className="confirm">
                    OK
                </Button>
            </DialogActions>
        </Dialog>
    );
}
