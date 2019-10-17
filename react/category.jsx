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
import { Money, TransactionList } from "../javascript/main";

export class CategoryView extends React.Component {
    constructor(props) {
        super(props);
        this.categoryid = props.categoryid;
        this.state = {}
    }

    componentDidMount() {
        fetch('http://localhost:8080/json/category/' + this.categoryid)
            .then((response) => { return response.json()})
            .then((category) => { this.setState(category) });
    }

    render() {
        return <div>
            <h1>{this.state.Name}</h1>
            <div>{this.state.Description}</div>
            <table>
                <tbody>
                <tr><td>Current Balance</td><td><Money value={this.state.CurrentBalance}/></td></tr>
                </tbody>
            </table>
        </div>;
    }
}


export function CategoryBlock(props) {
    const categoryid = props.id;
    return (
        <div>
            <CategoryView categoryid={categoryid}/>
            <h2>Transactions</h2>
            <TransactionList categoryid={categoryid}/>
        </div>
    );
}
