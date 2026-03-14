<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
    
    <xsl:template match="/">
        <out>
            <xsl:apply-templates/>
        </out>
    </xsl:template>
    <xsl:template match="xsl:stylesheet" expand-text="true">
        <expandtext>
            <count>{count(*)}</count>
            <xsl:text disable-output-escaping="true">&lt;test1/&gt;</xsl:text>
        </expandtext>
        <xsl:apply-templates/>
    </xsl:template>
    <xsl:template match="xsl:template" expand-text="false">
        <notexpandtext>
            <count>{count(*)}</count>
            <xsl:text disable-output-escaping="true">&lt;test2/&gt;</xsl:text>
        </notexpandtext>
    </xsl:template>
</xsl:stylesheet>