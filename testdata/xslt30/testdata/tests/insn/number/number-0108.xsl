<?xml version="1.0"?> 

<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0"
exclude-result-prefixes="xs">

  <!--test start-at with @value (xslt 3.0), using signed integers in @start-at -->
  
  <xsl:param name="p" as="xs:integer*" select="1,4,5,8,10,12,15,19,22,33,44"/>

  <xsl:template name="main">
    <out>
      <xsl:number value="$p" start-at="-0  1 -2  3  -4  5 -6 7 -8  9"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
