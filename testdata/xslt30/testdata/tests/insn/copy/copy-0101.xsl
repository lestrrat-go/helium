<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#shallow-copy?>
<!-- Test for simple identity transformation w/Template match -->                
<xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
</xsl:template>

<xsl:template match="*|@*|comment()|processing-instruction()|text()">
    <xsl:copy>
        <xsl:apply-templates select="*|@*|comment()|processing-instruction()|text()"/>
    </xsl:copy>
</xsl:template>
     
</xsl:stylesheet>