<?xml version="1.0" encoding="iso-8859-1"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
                
                <!-- the first non-alpha token is a separator tokens if it appears
                between two formatting tokens -->

<xsl:template match="/">
<a><xsl:number value="5 to 8" format="1;1)"/></a>
</xsl:template>

</xsl:stylesheet>