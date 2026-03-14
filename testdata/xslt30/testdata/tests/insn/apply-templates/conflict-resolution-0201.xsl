<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <?spec xslt#conflict?>
    <!-- Test for conflict Resolution: nonsimple (predicate)
     No conflict warnings should be seen. -->

    <xsl:template match="doc">
        <out>
            <xsl:apply-templates select="file"/>
        </out>
    </xsl:template>

    <xsl:template match="file[@test='false']">
        <xsl:text>Match-w/qualified-attribute-value</xsl:text>
    </xsl:template>

    <xsl:template match="file">
        <xsl:text>Match-on-node-name,</xsl:text>
    </xsl:template>

    <xsl:template match="*">
        <xsl:text>Match-of-wildcard,</xsl:text>
    </xsl:template>

    <xsl:template match="node()">
        <xsl:text>Match-of-node,</xsl:text>
    </xsl:template>

</xsl:stylesheet>
