<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE:   Test that in 1.0 mode xsl:value uses first-value semantics -->


<xsl:template match="/">
  <z xsl:version="1.0"><xsl:value-of select="1 to 5"/></z>
</xsl:template>
 
</xsl:stylesheet>
