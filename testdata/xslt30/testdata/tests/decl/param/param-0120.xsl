<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com" xmlns:t="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="foo" version="3.0">
    <!-- Purpose: test 3 global variables and 1 stylesheet function that reference each other, testing use of static="no", "false" and "0" -->

    <t:function name="foo:a">
        <t:value-of select="$z"/>
    </t:function>

    <t:param name="y" static="no  ">
        <t:value-of select="foo:a()"/>
    </t:param>

    <t:param name="x" static="false ">
        <t:value-of select="$y"/>
    </t:param>

    <t:param name="z" static=" 0">
        <t:value-of select="'foo'"/>
    </t:param>

    <t:template match="/">
        <out>
            <t:value-of select="$x"/>
        </out>
    </t:template>
</t:transform>
