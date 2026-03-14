<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:foo="http://www.foo.com" xmlns:t="http://www.w3.org/1999/XSL/Transform"
    exclude-result-prefixes="foo" version="2.0">
    <!-- Purpose: test 3 global variables and 1 stylesheet function that reference each other -->

    <t:function name="foo:a">
        <t:value-of select="$z"/>
    </t:function>

    <t:param name="y">
        <t:value-of select="foo:a()"/>
    </t:param>

    <t:param name="x">
        <t:value-of select="$y"/>
    </t:param>

    <t:param name="z">
        <t:value-of select="'foo'"/>
    </t:param>

    <t:template match="/">
        <out>
            <t:value-of select="$x"/>
        </out>
    </t:template>
</t:transform>
