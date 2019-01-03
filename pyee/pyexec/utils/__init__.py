# Copyright 2018 ICON Foundation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Utilities

Functions and classes in this module don't have any external dependencies.
"""

import hashlib
import re
from typing import Any, Union

from ..icon_constant import BUILTIN_SCORE_ADDRESS_MAPPER


def int_to_bytes(n: int) -> bytes:
    length = byte_length_of_int(n)
    return n.to_bytes(length, byteorder='big', signed=True)


def byte_length_of_int(n: int):
    return (n.bit_length() + 8) // 8


def is_lowercase_hex_string(value: str) -> bool:
    """Check whether value is hexadecimal format or not

    :param value: text
    :return: True(lowercase hexadecimal) otherwise False
    """

    try:
        result = re.match('[0-9a-f]+', value)
        return len(result.group(0)) == len(value)
    except:
        pass

    return False


def sha3_256(data: bytes) -> bytes:
    return hashlib.sha3_256(data).digest()


def to_camel_case(snake_str: str) -> str:
    str_array = snake_str.split('_')
    return str_array[0] + ''.join(sub.title() for sub in str_array[1:])


def check_error_response(result: Any):
    return isinstance(result, dict) and result.get('error')


def get_main_type_from_annotations_type(annotations_type: type) -> type:
    main_type = None

    if hasattr(annotations_type, '__origin__') and annotations_type.__origin__ is not Union:
        return annotations_type.__origin__

    # in python 3.7, _subs_tree method has excluded.
    if hasattr(annotations_type, '__args__'):
        annotations = annotations_type.__args__
        for annotation_type in annotations:
            if annotation_type is not None:
                main_type = annotation_type
                break
    else:
        main_type = annotations_type
    return main_type


def is_builtin_score(score_address: str) -> bool:
    return score_address in BUILTIN_SCORE_ADDRESS_MAPPER.values()
