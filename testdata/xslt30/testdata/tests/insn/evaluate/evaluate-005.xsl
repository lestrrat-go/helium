<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- test xsl:evaluate() with local range variables. -->
<!-- Test was formerly saxon082.xsl -->
 
    
  <xsl:template name="main">
  <xsl:variable name="a"/><xsl:variable name="b"/><xsl:variable name="c"/>
  <out>
    <xsl:evaluate xpath="'for $a in (1,2), $b in (3,4) return $a'"/>
  </out>
  </xsl:template>
 

  
</xsl:stylesheet>
