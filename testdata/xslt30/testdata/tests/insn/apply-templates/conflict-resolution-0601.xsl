<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <?spec xslt#patterns?>
    <!-- Test use of a variable within a pattern (XSLT 2.0) -->

    <xsl:param name="p" select="2"/>

    <xsl:template match="*">
        <xsl:copy>
            <xsl:apply-templates/>
        </xsl:copy>
    </xsl:template>

    <xsl:template match="*[@id=$p]">
        <xsl:copy>
            <xsl:attribute name="special">yes</xsl:attribute>
            <xsl:apply-templates/>
        </xsl:copy>
    </xsl:template>

</xsl:stylesheet>
