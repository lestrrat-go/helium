<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:foo="http://ns.example.com/sch002"
    exclude-result-prefixes="foo">


    <!-- Purpose: Test for importing two different schemas with the same @namespace but different import 
        precedence. Cases tested:
        - elements, attributes or types with the same names but different properties exist 
        in the two schemas 
        - elements, attributes or types with different names exist in the two schemas.-->

    <xsl:import href="import-schema-177a.xsl"/>
    <xsl:import-schema namespace="http://ns.example.com/sch002" schema-location="schema002.xsd"/>

    <xsl:variable name="var1" as="foo:testType*">
        <xsl:sequence
            select="(foo:testType(2000),foo:testType(2001),foo:testType(2002),foo:testType(2003))"/>
    </xsl:variable>

    <xsl:variable name="var2" as="element()">
        <foo:complexTest xsl:type="foo:complexTestType" foo:type="333">
            <subject>math</subject>
            <size>50</size>
        </foo:complexTest>
    </xsl:variable>

    <xsl:variable name="var3" as="attribute()">
        <xsl:attribute name="foo:type" type="foo:testType">444</xsl:attribute>
    </xsl:variable>

    <xsl:template match="/doc">
        <out>
            <type>
                <xsl:value-of select="sum($var1)"/>
            </type>
            <elem>
                <xsl:value-of select="$var2 instance of schema-element(foo:complexTest)"/>
                <xsl:text> * </xsl:text>
                <xsl:value-of select="$var2/@foo:type instance of schema-attribute(foo:type)"/>
                <xsl:text> * </xsl:text>
                <xsl:copy-of select="$var2"/>
            </elem>
            <attrib>
                <xsl:value-of select="$var3 instance of schema-attribute(foo:type)"/>
                <xsl:text> * </xsl:text>
                <xsl:value-of select="$var3"/>
            </attrib>
        </out>
    </xsl:template>


</xsl:stylesheet>
