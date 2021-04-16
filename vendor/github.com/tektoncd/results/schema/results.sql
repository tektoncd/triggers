-- Copyright 2020 The Tekton Authors
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--      http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

CREATE TABLE results (
	parent varchar(64),
	id varchar(64),

	name varchar(64),
	annotations BLOB,

	created_time timestamp default current_timestamp not null,
	updated_time timestamp default current_timestamp not null,
	
	etag varchar(128),

	PRIMARY KEY(parent, id)
);
CREATE UNIQUE INDEX results_by_name ON results(parent, name);

CREATE TABLE records (
	parent varchar(64),
	result_id varchar(64),
	id varchar(64),

	result_name varchar(64),
	name varchar(64),
	data BLOB,

	created_time timestamp default current_timestamp not null,
	updated_time timestamp default current_timestamp not null,

	etag varchar(128),

	PRIMARY KEY(parent, result_id, id),
	FOREIGN KEY(parent, result_id) REFERENCES results(parent, id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX records_by_name ON records(parent, result_name, name);
