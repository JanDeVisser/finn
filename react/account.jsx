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

import React from 'react';
import { DateFormat, Money, DisplayReference, TransactionList } from "../javascript/main";

export class AccountView extends React.Component {
    constructor(props) {
        super(props);
        this.accountid = props.accountid;
        this.state = {};
    }

    componentDidMount() {
        fetch('http://localhost:8080/json/account/' + this.accountid)
            .then((response) => { return response.json()})
            .then((account) => { this.setState(account) });
    }

    render() {
        return (<div>
            <h1>{this.state.AccName}</h1>
            <div>{this.state.Description}</div>
            <table>
                <tbody>
                <tr><td>Account #</td><td>{this.state.AccNr}</td></tr>
                <tr><td>Currency</td><td>{this.state.Currency}</td></tr>
                <tr><td>Opening Date</td><td><DateFormat value={this.state.OpeningDate}/></td></tr>
                <tr><td>Opening Balance</td><td><Money value={this.state.OpeningBalance}/></td></tr>
                <tr><td>Current Balance</td><td><Money value={this.state.CurrentBalance}/></td></tr>
                </tbody>
            </table>
        </div>);
    }
}

export class AccountRow extends React.Component {
    constructor(props) {
        super(props);
    }

    render() {
        const acc = this.props.value;
        const inst = this.props.parent;
        return (
            <tr>
                <td><a href={'/account/' + acc.Ident}>{acc.AccName}</a></td>
                <td><a href={'/institution/' + acc.InstIdent}>{acc.InstName}</a></td>
                <td>{acc.AccNr}</td>
                <td>{acc.Description}</td>
                <td><Money value={acc.CurrentBalance}/></td>
                <td><DateFormat value={acc.OpeningDate}/></td>
                <td><Money value={acc.OpeningBalance}/></td>
            </tr>
        );
    }
}

export class AccountsList extends React.Component {
    constructor(props) {
        super(props);
        this.totalBalance = 0.0;
        this.state = { acclist: [] };
    }

    componentDidMount() {
        let href = "?joinparent=institution";
        for (let k in this.props) {
            if (this.props.hasOwnProperty(k)) {
                href += "&" + encodeURIComponent(k) + "=" + encodeURIComponent(this.props[k])
            }
        }
        fetch('http://localhost:8080/json/account' + href)
            .then((response) => { return response.json()})
            .then((acclist) => { this.setState({ acclist: acclist }) });
    }

    getTotals() {
        return (
            <tr>
                <td colSpan="4">T O T A L</td>
                <td><Money value={this.totalBalance}/></td>
                <td colSpan="2">&nbsp;</td>
            </tr>
        );
    }

    getRows() {
        return this.state.acclist.map((acc) => {
            this.totalBalance += acc.CurrentBalance;
            return <AccountRow key={acc[0].Ident} value={acc[0]} parent={acc[1]}/>
        });
    }

    render() {
        console.log(this.state)
        return (
            <table className="datatable">
                <thead>
                <tr>
                    <th>Name</th>
                    <th>Institution</th>
                    <th>Number</th>
                    <th>Description</th>
                    <th>Current Balance</th>
                    <th>Opening Date</th>
                    <th>Opening Balance</th>
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

export function AccountBlock(props) {
    const accountid = props.id;
    const uploadUrl = `/account/upload/${accountid}`;
    return (
        <div>
            <AccountView accountid={accountid}/>
            <h2>Transactions</h2>
            <TransactionList accountid={accountid}/>
            <h2>Import Transactions</h2>
            <form encType="multipart/form-data" action={uploadUrl} method="post">
                <input type="file" name="schema"/>
                <input type="submit" value="upload"/>
            </form>
        </div>
    );
}

export function AccountsListBlock(props) {
    return (
        <div>
            <h2>Accounts</h2>
            <AccountsList {...this.props}/>
        </div>
    );
}


