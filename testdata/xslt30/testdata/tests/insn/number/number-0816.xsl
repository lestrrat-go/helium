<?xml version="1.0" encoding="iso-8859-1"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">
                
                <!-- the first and last non-alpha tokens are *not* separator tokens, and shouldn't
                     be used when there are fewer tokens than numbers to be formatted -->

<xsl:template match="/">
<a><xsl:number value="5 to 8" format="(1)"/></a>
</xsl:template>

</xsl:stylesheet>