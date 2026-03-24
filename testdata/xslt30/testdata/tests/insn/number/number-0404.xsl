<?xml version="1.0"?> 
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test xsl:number with value= a sequence -->
  <xsl:template match="doc">
    <out>
      <xsl:number value="(10 to 20)" format="1,1"/>
    </out>
  </xsl:template>
  
  <xsl:strip-space elements="*"/>
   
</xsl:stylesheet>
