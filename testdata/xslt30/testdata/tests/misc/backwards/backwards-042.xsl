<xsl:stylesheet version="1.0"
		xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
		
		<!-- In range expressions, (M to N), standard function conversion rules apply, which means that in BC mode,
		     M or N can be a sequence of integers, and the first-item rule holds -->
		<!-- Author: Michael Kay -->
		<!-- See also: spec bugzilla entry 7424 --> 

<xsl:template match="/">
  <out><xsl:value-of select="(1 to 5) to (3,4)" separator=","/></out>
</xsl:template>

</xsl:stylesheet>
