<xsl:stylesheet version="2.0" 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec ser#text?>
<xsl:output method="text" encoding="utf-8"/>

<xsl:template match="/">
 <xsl:text>&lt;doc&gt;&#x10300;-</xsl:text>
 <xsl:copy-of select="doc"/>
 <xsl:text>&lt;/doc&gt;</xsl:text> 
</xsl:template>

</xsl:stylesheet>
