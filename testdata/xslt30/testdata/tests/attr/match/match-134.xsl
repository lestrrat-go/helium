<?xml version="1.0"?>
<xsl:stylesheet 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

    <!-- Purpose: grouping pattern with atomic value as context item -->
  

<xsl:template name="main">
 <out>
 	<xsl:for-each-group select="1 to 20" group-starting-with=".[. mod 3 = 0]">
 	    <g><xsl:copy-of select="current-group()"/></g>
 	</xsl:for-each-group>
 </out>
</xsl:template>



</xsl:stylesheet>
