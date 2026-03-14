<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
    <!-- Purpose: Test case that uses xml:base in the content of a new element to verify 
       that the base URI of the new element is the value of xml:base attribute resolved 
       against the base URI of the xsl:element in the stylesheet. -->

    <t:variable name="var" select="/doc/namespace[1]"/>

    <t:template match="/doc">
        <out>
            <t:element name="e1" xml:base="http://www.one.org/">
                <t:value-of select="resolve-uri('002.xml')"/>
            </t:element>
            <t:element name="e2" xml:base="http://www.two.org/">
                <t:value-of select="resolve-uri($var)"/>
            </t:element>
        </out>
    </t:template>
</t:transform>
