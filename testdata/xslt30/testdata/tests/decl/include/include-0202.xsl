<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test xsl:apply-imports with parameters (XSLT 1.1 feature) -->

<?spec xslt#apply-imports?>  
  <xsl:import href="include-0202a.xsl"/>
  
  <xsl:template match="doc">
    <out>
     <xsl:apply-imports>
       <xsl:with-param name="magic" select="81"/>
     </xsl:apply-imports>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
