/*
 * This file is part of Finn.
 *
 * Copyright (c) 2019 Jan de Visser <jan@finiandarcy.com>
 *
 * Finn is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Finn is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Finn.  If not, see <https://www.gnu.org/licenses/>.
 */

// npx babel --watch react --out-dir javascript  --presets=@babel/preset-env,@babel/preset-react
// npx browserify javascript\main.js javascript\account.js javascript\category.js -u javascript\bundle.js -o javascript\bundle.js
// npx watchify javascript\main.js javascript\account.js javascript\category.js -v -u javascript\bundle.js -o javascript\bundle.js

import React from 'react';
import { render } from 'react-dom';
import { AccountBlock, AccountsListBlock } from "../javascript/account";
import { CategoryBlock } from "../javascript/category";

export function Money(props) {
    const amt = props["value"]
    if ((typeof(amt) === "undefined") || (amt === 0.0)) {
        return <span>&nbsp;</span>;
    } else {
        return <span>{amt.toFixed(2)}</span>;
    }
}

export function DateFormat(props) {
    const d = props["value"];
    if ((typeof(d) === "undefined") || (d === "")) {
        return <span>&nbsp;</span>
    } else {
        const dateObj = new Date(d);
        const options = { year: 'numeric', month: 'short', day: 'numeric' };
        const dateStr = new Intl.DateTimeFormat('en-CA', options).format(dateObj);
        return <span>{dateStr}</span>;
    }
}

export function DisplayReference(props) {
    const obj = props["value"];
    const ref = props["reference"];
    const label = props["label"] || "Name";
    const id = props["id"] || "Ident"

    const refValue = obj[ref];
    if (refValue) {
        let href = `/${ref.toLowerCase()}/${refValue[id]}`;
        return <a href={href}>{refValue[label]}</a>
    } else {
        return <span>&nbsp;</span>
    }
}

export function renderComponent(id, type, props) {
    let domContainer = document.querySelector('#' + id);
    if (domContainer) {
        ReactDOM.render(React.createElement(type, props), domContainer);
    }
}


export class TransactionRow extends React.Component {
    constructor(props) {
        super(props);
    }

    render() {
        const tx = this.props.value;
        return (
            <tr>
                <td>{tx.TXType}</td>
                <td><DateFormat value={tx.Date}/></td>
                <td>{tx.Description}</td>
                <td><Money value={tx.Credit}/></td>
                <td><Money value={tx.Debit}/></td>
                <td><DisplayReference value={tx} reference="Contact"/></td>
                <td><DisplayReference value={tx} reference="Category"/></td>
                <td><DisplayReference value={tx} reference="Project"/></td>
            </tr>
        );
    }
}

export class TransactionList extends React.Component {
    constructor(props) {
        super(props);
        this.totalCredit = 0.0;
        this.totalDebit = 0.0;
        this.state = { txlist: [] };
    }

    componentDidMount() {
        let href = "?";
        for (let k in this.props) {
            if (this.props.hasOwnProperty(k)) {
                href += encodeURIComponent(k) + "=" + encodeURIComponent(this.props[k]) + "&"
            }
        }
        fetch('http://localhost:8080/json/transaction' + href)
            .then((response) => { return response.json()})
            .then((txlist) => { this.setState({ txlist: txlist }) });
    }

    getTotals() {
        return (
                <tr>
                    <td colSpan="3">T O T A L</td>
                    <td><Money value={this.totalCredit}/></td>
                    <td><Money value={this.totalDebit}/></td>
                    <td colSpan="3">&nbsp;</td>
                </tr>
            );
    }

    getRows() {
        return this.state.txlist.map((tx) => {
            tx = tx[0];
            this.totalDebit += tx.Debit;
            this.totalCredit += tx.Credit;
            return <TransactionRow key={tx.Ident} value={tx}/>
        });
    }


    render() {
        return (
            <table className="datatable">
                <thead>
                    <tr>
                        <th>T</th>
                        <th>Date</th>
                        <th>Description</th>
                        <th>In</th>
                        <th>Out</th>
                        <th>Contact</th>
                        <th>Category</th>
                        <th>Project</th>
                    </tr>
                </thead>
                <tbody>
                    { this.getRows() }
                </tbody>
                <tfoot>
                    { this.getTotals() }
                </tfoot>
            </table>
        );
    }
}

export class Topbar extends React.Component {
    constructor(props) {
        super(props);
    }

    render() {
        return (
            <div id="topbar" className="topbar">
                <nav>
                    <ul>
                        <li><a className="navitem" href="/">Home</a></li>
                    </ul>
                </nav>
            </div>
        )
    }
}

export class Sidebar extends React.Component {
    constructor(props) {
        super(props);
    }

    render() {
        return (
            <div id="sidebar" className="sidebar">
                <li><a href="/institutions">Institutions</a></li>
                <li><a href="/accounts">Accounts</a></li>
                <li><a href="/categories">Categories</a></li>
                <li><a href="/projects">Projects</a></li>
            </div>
        );
    };
}

export class Document extends React.Component {
    constructor(props) {
        super(props);
    }

    render() {
        return (
            <div id="document" className="document">
                <div id="documentwrapper" className="documentwrapper">
                    <div id="mainpagewrapper" className="mainpagewrapper">
                        <div className="mainpage" id="mainpage">
                            {this.props.page}
                        </div>
                    </div>
                </div>
                <div id="sidebarwrapper" className="sidebarwrapper">
                    <Sidebar/>
                </div>
            </div>
        )
    }
}

export class App extends React.Component {
    constructor(props) {
        super(props);
    }

    render() {
        return (
            <div>
                <Topbar/>
                <Document page={this.props.page}/>
            </div>
        )
    }
}

const blockmap = {
    "account": AccountBlock,
    "accounts": AccountsListBlock,
    "category": CategoryBlock
};

const path = document.location.pathname.split("/");
let props = {};
const key = parseInt(path[path.length - 1]);
if (!isNaN(key)) {
    props.id = key
} else {
    let params = (new URL(document.location)).searchParams;
    for (const [key, value] of params) {
        props.key = value;
    }
}

const kind = path[1];
const block = ((kind, props) => {
    let block;
    try {
        block = new blockmap[kind](props);
    } catch {
        block = blockmap[kind](props);
    }
    return block;
})(kind, props);

render(React.createElement(App, {page: block}), document.querySelector('#container'));
