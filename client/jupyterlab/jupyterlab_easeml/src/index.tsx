import {ILayoutRestorer, JupyterFrontEnd, JupyterFrontEndPlugin, ILabShell} from '@jupyterlab/application';
import {ICommandPalette, MainAreaWidget, WidgetTracker} from '@jupyterlab/apputils';
import {Message} from '@phosphor/messaging';
import {Widget} from '@phosphor/widgets';
import Vue from 'vue';
import EASEML_VUE from './Iframe.vue'
import { ReactWidget } from "@jupyterlab/apputils";
import * as React from "react";
//import { defaultIconRegistry } from '@jupyterlab/ui-components';
//import iconSvg from './icon/icon.svg';
import { ILauncher } from '@jupyterlab/launcher';
//import PropTypes from 'prop-types';
import Button from 'react-bootstrap/Button';
import 'bootstrap/dist/css/bootstrap.min.css';

class Easeml extends Widget {
    /**
     * Construct a new Easeml widget.
     */
    constructor() {
        super();
    }

    /**
     * Handle update requests for the widget.
     */
    async onUpdateRequest(msg: Message): Promise<void> {
        new Vue({
            el: this.node,
            render: h => h(EASEML_VUE)
        })
    }
}

class SidebarButton extends React.Component{
    render() {
        return (
            <div
                style={{
                    marginTop: "25px",
                    background: "#FFFFFF",
                    color: "#000000",
                    fontFamily: "Helvetica",
                    height: "100%",
                    display: "flex",
                    flexDirection: "column"
                }}
            >
            <Button href="#" data-commandlinker-command="easeml:openside" variant="primary" size="sm" block>
                open easeml client
            </Button>
            </div>
        );
    }
}


/**
 * Activate the jupyterlab_easml widget extension.
 */
function activate(  app: JupyterFrontEnd, 
                    palette: ICommandPalette, 
                    restorer: ILayoutRestorer,
                    labShell: ILabShell,
                    launcher: ILauncher | null
                     ) {
    console.log('JupyterLab extension jupyterlab_easml extension is activated!');



    function easmlOpen(){
        // Declare a widget variable
        let widget: MainAreaWidget<Easeml>;

        if (!widget || widget.isDisposed) {
            // Create a new widget if one does not exist
            const content = new Easeml();
            widget = new MainAreaWidget({content});
            widget.id = 'easeml-jupyterlab';
            widget.title.label = 'Easeml';
            widget.title.closable = true;
        }
        if (!tracker.has(widget)) {
            // Track the state of the widget for later restoration
            tracker.add(widget);
        }
        if (!widget.isAttached) {
            // Attach the widget to the main work area if it's not there
            app.shell.add(widget, 'main');
        }
        widget.content.update();

        // Activate the widget
        app.shell.activateById(widget.id);
    }


    // Add an application command
    const command: string = 'easeml:open';
    app.commands.addCommand(command, {
        label: 'Open easeml',
        execute: easmlOpen
    });

    // Add an application command
    const command2: string = 'easeml:openside';
    app.commands.addCommand(command2, {
        label: 'Open easeml and close side',
        execute: () => {
            easmlOpen();
            labShell.collapseLeft();
        }
    });

    
    if (launcher) { 
       launcher.add({ 
         command: "easeml:open", 
         category: 'Other', 
         rank: 0 
       }); 
    }

    const side_widget = ReactWidget.create(
        <SidebarButton/>
      );

    console.log(app.commands.listCommands())

    side_widget.id = "easeml-sidebar";
    side_widget.title.iconClass = "jp-SpreadsheetIcon jp-SideBar-tabIcon";
    side_widget.title.caption = "SIDE BAR";
    side_widget.title.closable = true;

    restorer.add(side_widget, side_widget.id);
    labShell.add(side_widget, "left");
            
    // Add the command to the palette.
    palette.addItem({command, category: 'START EASEML'});

    // Track and restore the widget state
    let tracker = new WidgetTracker<MainAreaWidget<Easeml>>({
        namespace: 'vue'
    });
    restorer.restore(tracker, {
        command,
        name: () => 'vue'
    });
}

/**
 * Initialization data for the jupyterlab_vue extension.
 */
const extension: JupyterFrontEndPlugin<void> = {
    id: 'jupyterlab_easeml',
    autoStart: true,
    requires: [ICommandPalette, ILayoutRestorer,ILabShell],
    optional: [ILauncher],
    activate: activate
};

export default extension;
