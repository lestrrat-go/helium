<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
  <!-- Purpose: Test an "xsl:choose" with a complex true 
       expression in its "when" section. -->
    
<xsl:template match="doc">
  <out>
    <xsl:choose>
     <xsl:when test = "(((((2*10)-4)+9) div 5) div 2) > 2">
       Test executed successfully.
     </xsl:when>
    </xsl:choose>
  </out>
</xsl:template>

</xsl:stylesheet>
