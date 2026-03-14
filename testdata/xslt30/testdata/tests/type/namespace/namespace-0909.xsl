<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                >

<!-- PURPOSE: List the id's of the namespace nodes for an element -->
<!-- rewritten 2006-03-13 to avoid any dependency on the actual choice of generate-id() -->
     

<xsl:template match="/">
<out>
  <xsl:value-of select="count(distinct-values(for $x in //namespace::* return generate-id($x)))"/>
</out>
</xsl:template>
 
</xsl:stylesheet>
