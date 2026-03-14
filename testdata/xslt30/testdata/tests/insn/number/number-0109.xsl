<?xml version="1.0"?> 

<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0"
exclude-result-prefixes="xs">

  <!--test invlaid start-at attribute -->
  
  <xsl:param name="p" as="xs:integer*" select="1,4,5,8,10,12,15,19,22,33,44"/>

  <xsl:template name="main">
    <out>
      <xsl:number value="$p" start-at="1..2"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
