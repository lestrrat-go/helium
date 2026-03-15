<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

    <!--
        Purpose: Parentheses in patterns become legal in XSLT 3.0. They have the same priority.
    -->

    <xsl:template match="(doc|cod)">
        <wrong />
    </xsl:template>

    <xsl:template match="doc">
        <ok />
    </xsl:template>

    <xsl:template match="/">
        <xsl:apply-templates />
    </xsl:template>

</xsl:stylesheet>
