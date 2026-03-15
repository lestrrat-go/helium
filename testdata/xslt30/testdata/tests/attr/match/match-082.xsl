<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:f="http://f.com/">

    <!--
        Purpose: Test that parens are not allowed at the top level of a pattern.
        This test is about XSLT 2.0 behavior throwing an error on syntax, parens become allowed in XSLT 3.0. 
    -->

    <xsl:template match="(doc|cod)">
        <ok />
    </xsl:template>

    <xsl:template match="doc">
        <wrong />
    </xsl:template>

    <xsl:template match="/">
        <xsl:apply-templates />
    </xsl:template>

</xsl:stylesheet>
