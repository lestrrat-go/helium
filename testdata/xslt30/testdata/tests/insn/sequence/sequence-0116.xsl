<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


<!-- PURPOSE:   Basic test of "as" attribute on xsl:template -->


<xsl:template match="doc" as="element()">
  <xsl:variable name="q" select="15"/>
  <zzz><xsl:value-of select="$q" separator=","/></zzz>
</xsl:template>
 
</xsl:stylesheet>
