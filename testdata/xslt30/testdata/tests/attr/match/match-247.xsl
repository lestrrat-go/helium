<xsl:stylesheet version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    
    
    <xsl:template match="node() | / | @* | method[@href = 'ref1']">
        <xsl:call-template name="identity"/>
    </xsl:template>
    
    <xsl:template match="method"/>
    
    <xsl:template name="identity">
        <xsl:copy>
            <xsl:apply-templates select="@* | node() " />
        </xsl:copy>
    </xsl:template>
    
</xsl:stylesheet>
