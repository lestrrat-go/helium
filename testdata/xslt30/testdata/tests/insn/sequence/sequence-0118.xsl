<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE:   Test that in 2.0 mode the default separator in xsl:value of is a single space-->


<xsl:template match="/">
  <z><xsl:value-of select="1 to 5"/></z>
</xsl:template>
 
</xsl:stylesheet>
