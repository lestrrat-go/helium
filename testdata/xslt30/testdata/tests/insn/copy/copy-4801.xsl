<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
<?spec xslt#shallow-copy?>
<!-- Test for simple identity transformation: xsl:copy with select attribute  -->                
<xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
</xsl:template>

<xsl:template match="*|@*|comment()|processing-instruction()|text()">
    <xsl:copy select=".">
        <xsl:apply-templates select="*|@*|comment()|processing-instruction()|text()"/>
    </xsl:copy>
</xsl:template>
     
</xsl:stylesheet>